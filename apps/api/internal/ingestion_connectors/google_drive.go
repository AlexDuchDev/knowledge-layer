package ingestion_connectors

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"

	"github.com/knowledgelayer/api/internal/ingestion_connectors/families/docs_wiki"
)

const (
	maxDriveFileBytes    = 4 << 20
	defaultMaxDriveFiles = 25
	maxDriveFilesCap     = 100
	// RecordTypeGoogleDriveDocument is the normalized_records.record_type for Drive/Docs imports.
	RecordTypeGoogleDriveDocument = "google_drive_document"
)

type googleDriveConnectorConfig struct {
	FolderID        string          `json:"folder_id"`
	ServiceAccount  json.RawMessage `json:"service_account"`
	MaxFilesPerSync int             `json:"max_files_per_sync"`
}

func parseGoogleDriveConfig(raw json.RawMessage) (*googleDriveConnectorConfig, error) {
	var cfg googleDriveConnectorConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cfg.FolderID) == "" {
		return nil, errors.New("connector_config_json.folder_id required")
	}
	if len(cfg.ServiceAccount) == 0 {
		return nil, errors.New("connector_config_json.service_account required (full service account JSON object)")
	}
	if cfg.MaxFilesPerSync <= 0 {
		cfg.MaxFilesPerSync = defaultMaxDriveFiles
	}
	if cfg.MaxFilesPerSync > maxDriveFilesCap {
		cfg.MaxFilesPerSync = maxDriveFilesCap
	}
	return &cfg, nil
}

func newDriveService(ctx context.Context, saJSON []byte) (*drive.Service, error) {
	creds, err := google.CredentialsFromJSON(ctx, saJSON, drive.DriveReadonlyScope)
	if err != nil {
		return nil, fmt.Errorf("google credentials: %w", err)
	}
	client := oauth2.NewClient(ctx, creds.TokenSource)
	svc, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}
	return svc, nil
}

func readLimited(r io.Reader) ([]byte, bool, error) {
	b, err := io.ReadAll(io.LimitReader(r, maxDriveFileBytes+1))
	if err != nil {
		return nil, false, err
	}
	if len(b) > maxDriveFileBytes {
		return b[:maxDriveFileBytes], true, nil
	}
	return b, false, nil
}

func fetchDriveFileBody(ctx context.Context, svc *drive.Service, f *drive.File) (content []byte, exportMime string, truncated bool, skip bool, err error) {
	switch f.MimeType {
	case "application/vnd.google-apps.folder":
		return nil, "", false, true, nil
	case "application/vnd.google-apps.document":
		resp, err := svc.Files.Export(f.Id, "text/plain").Context(ctx).Download()
		if err != nil {
			return nil, "", false, false, err
		}
		defer resp.Body.Close()
		b, trunc, err := readLimited(resp.Body)
		return b, "text/plain", trunc, false, err
	case "application/vnd.google-apps.spreadsheet":
		resp, err := svc.Files.Export(f.Id, "text/csv").Context(ctx).Download()
		if err != nil {
			return nil, "", false, false, err
		}
		defer resp.Body.Close()
		b, trunc, err := readLimited(resp.Body)
		return b, "text/csv", trunc, false, err
	default:
		if strings.HasPrefix(f.MimeType, "text/") || f.MimeType == "application/json" ||
			f.MimeType == "application/javascript" || f.MimeType == "application/xml" {
			resp, err := svc.Files.Get(f.Id).Context(ctx).Download()
			if err != nil {
				return nil, "", false, false, err
			}
			defer resp.Body.Close()
			b, trunc, err := readLimited(resp.Body)
			return b, f.MimeType, trunc, false, err
		}
		return nil, "", false, true, nil
	}
}

func driveRawMetadata(f *drive.File, exportMime string, truncated bool, content []byte) json.RawMessage {
	owners := make([]string, 0, len(f.Owners))
	for _, o := range f.Owners {
		if o == nil {
			continue
		}
		if o.EmailAddress != "" {
			owners = append(owners, o.EmailAddress)
		} else if o.DisplayName != "" {
			owners = append(owners, o.DisplayName)
		}
	}
	lastMod := ""
	if f.LastModifyingUser != nil {
		if f.LastModifyingUser.EmailAddress != "" {
			lastMod = f.LastModifyingUser.EmailAddress
		} else {
			lastMod = f.LastModifyingUser.DisplayName
		}
	}
	m := map[string]any{
		"connector":     "google_drive",
		"file_id":       f.Id,
		"name":          f.Name,
		"mime_type":     f.MimeType,
		"export_mime":   exportMime,
		"parents":       f.Parents,
		"owners":        owners,
		"last_modifier": lastMod,
		"web_view_link": f.WebViewLink,
		"truncated":     truncated,
		"content_text":  string(content),
	}
	b, _ := json.Marshal(m)
	return b
}

type normalizedDriveDoc struct {
	RecordKind        string         `json:"record_kind"`
	Title             string         `json:"title"`
	ExternalRef       string         `json:"external_ref"`
	SourceModifiedAt  string         `json:"source_modified_at,omitempty"`
	ParentFolderIDs   []string       `json:"parent_folder_ids,omitempty"`
	OwnerEmails       []string       `json:"owner_emails,omitempty"`
	LastModifyingUser string         `json:"last_modifying_user,omitempty"`
	MimeType          string         `json:"mime_type"`
	ExportMime        string         `json:"export_mime,omitempty"`
	BodyText          string         `json:"body_text"`
	WebViewLink       string         `json:"web_view_link,omitempty"`
	DownstreamHint    map[string]any `json:"downstream_hint"`
}

// buildNormalizedDriveRecord returns JSON payload and normalized_records.record_type.
// Google Docs exports use the shared docs_wiki family shape (docs_page); other files keep google_drive_document.
func buildNormalizedDriveRecord(feedID uuid.UUID, f *drive.File, exportMime string, content []byte, truncated bool) ([]byte, string, error) {
	owners := make([]string, 0, len(f.Owners))
	for _, o := range f.Owners {
		if o == nil {
			continue
		}
		if o.EmailAddress != "" {
			owners = append(owners, o.EmailAddress)
		}
	}
	lastMod := ""
	if f.LastModifyingUser != nil && f.LastModifyingUser.EmailAddress != "" {
		lastMod = f.LastModifyingUser.EmailAddress
	} else if f.LastModifyingUser != nil {
		lastMod = f.LastModifyingUser.DisplayName
	}
	if f.MimeType == "application/vnd.google-apps.document" {
		n := docs_wiki.FromGoogleDriveExport(
			feedID, f.Name, f.Id, exportMime, f.MimeType, string(content),
			f.ModifiedTime, append([]string(nil), f.Parents...), owners, lastMod, f.WebViewLink, truncated,
		)
		b, err := json.Marshal(n)
		return b, docs_wiki.RecordTypeDocsPage, err
	}
	n := normalizedDriveDoc{
		RecordKind:        RecordTypeGoogleDriveDocument,
		Title:             f.Name,
		ExternalRef:       f.Id,
		SourceModifiedAt:  f.ModifiedTime,
		ParentFolderIDs:   append([]string(nil), f.Parents...),
		OwnerEmails:       owners,
		LastModifyingUser: lastMod,
		MimeType:          f.MimeType,
		ExportMime:        exportMime,
		BodyText:          string(content),
		WebViewLink:       f.WebViewLink,
		DownstreamHint: map[string]any{
			"suggested_entity_type": "ReferenceDocument",
			"default_truth_mode":    "mirrored_authority",
		},
	}
	if truncated {
		n.BodyText += "\n\n[truncated at connector byte limit]"
	}
	b, err := json.Marshal(n)
	return b, RecordTypeGoogleDriveDocument, err
}

func parseDriveTime(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil
	}
	return &t
}

// SyncGoogleDrive performs manual sync for a google_drive connector: raw artifacts + normalized records only.
func (s *Service) SyncGoogleDrive(ctx context.Context, feedID uuid.UUID) (*IngestionRun, error) {
	feed, err := s.GetSourceFeed(ctx, feedID)
	if err != nil {
		return nil, err
	}
	conn, err := s.GetConnector(ctx, feed.ConnectorID)
	if err != nil {
		return nil, err
	}
	if conn.Type != "google_drive" {
		return nil, fmt.Errorf("connector is %s, not google_drive", conn.Type)
	}
	if feed.Status != "active" {
		return nil, errors.New("source feed must be active")
	}

	cfg, err := parseGoogleDriveConfig(feed.ConnectorConfigJSON)
	if err != nil {
		return nil, err
	}

	svc, err := newDriveService(ctx, cfg.ServiceAccount)
	if err != nil {
		return nil, err
	}

	runID, err := s.startIngestionRun(ctx, feedID)
	if err != nil {
		return nil, err
	}

	q := fmt.Sprintf("'%s' in parents and trashed = false", cfg.FolderID)
	// Single page only: respect max_files_per_sync as a hard cap per manual sync.
	fileList, err := svc.Files.List().Q(q).PageSize(int64(cfg.MaxFilesPerSync)).
		Fields("files(id,name,mimeType,modifiedTime,parents,owners,lastModifyingUser,webViewLink)").
		Context(ctx).Do()
	if err != nil {
		s.finalizeIngestionRun(ctx, runID, "failed", 0, 0, 0, 1)
		s.completeSourceFeedSync(ctx, feedID, 1, false)
		return nil, fmt.Errorf("drive list: %w", err)
	}

	ingested := 0
	deduped := 0
	warnings := 0
	errs := 0

	for _, f := range fileList.Files {
		if f == nil {
			continue
		}
		content, exportMime, truncated, skip, ferr := fetchDriveFileBody(ctx, svc, f)
		if skip {
			warnings++
			continue
		}
		if ferr != nil {
			errs++
			continue
		}
		if truncated {
			warnings++
		}

		meta := driveRawMetadata(f, exportMime, truncated, content)
		metaWithGov, merr := appendFeedGovernanceToRawJSON(*feed, meta)
		if merr != nil {
			errs++
			continue
		}
		h := hashBytes(metaWithGov)
		extID := f.Id

		var authorRef *string
		if f.LastModifyingUser != nil {
			if f.LastModifyingUser.EmailAddress != "" {
				authorRef = &f.LastModifyingUser.EmailAddress
			} else if f.LastModifyingUser.DisplayName != "" {
				authorRef = &f.LastModifyingUser.DisplayName
			}
		}
		srcTime := parseDriveTime(f.ModifiedTime)

		rawID, inserted, qerr := insertRawArtifactRow(ctx, s.pool, feedID, runID, "google_drive_file", extID, h, "", metaWithGov, srcTime, authorRef)
		if qerr != nil {
			errs++
			continue
		}
		if !inserted {
			deduped++
			continue
		}

		normPayload, recordType, nerr := buildNormalizedDriveRecord(feedID, f, exportMime, content, truncated)
		if nerr != nil {
			errs++
			continue
		}
		nh := hashBytes(normPayload)
		tag, xerr := s.pool.Exec(ctx, `
			INSERT INTO normalized_records (raw_artifact_id, source_feed_id, record_type, structured_payload_json, record_hash, source_timestamp, detected_author_ref, normalization_version)
			VALUES ($1,$2,$3,$4,$5,$6,$7,1)
			ON CONFLICT (source_feed_id, record_hash) DO NOTHING`,
			rawID, feedID, recordType, normPayload, nh, srcTime, authorRef)
		if xerr != nil {
			errs++
			continue
		}
		if tag.RowsAffected() == 0 {
			deduped++
			continue
		}
		ingested++
	}

	status := syncRunStatusFromCounts(ingested, errs)
	s.finalizeIngestionRun(ctx, runID, status, ingested, deduped, warnings, errs)
	s.completeSourceFeedSync(ctx, feedID, errs, false)

	return s.GetIngestionRun(ctx, runID)
}

// SyncSourceFeed runs connector-specific manual sync for all registered connector types.
func (s *Service) SyncSourceFeed(ctx context.Context, feedID uuid.UUID) (*IngestionRun, error) {
	feed, err := s.GetSourceFeed(ctx, feedID)
	if err != nil {
		return nil, err
	}
	conn, err := s.GetConnector(ctx, feed.ConnectorID)
	if err != nil {
		return nil, err
	}
	switch conn.Type {
	case "telegram":
		return s.SyncTelegram(ctx, feedID)
	case "slack":
		return s.SyncSlack(ctx, feedID)
	case "mattermost":
		return s.SyncMattermost(ctx, feedID)
	case "notion":
		return s.SyncNotion(ctx, feedID)
	case "jira":
		return s.SyncJira(ctx, feedID)
	case "trello":
		return s.SyncTrello(ctx, feedID)
	case "fireflies":
		return s.SyncFireflies(ctx, feedID)
	case "google_calendar":
		return s.SyncGoogleCalendar(ctx, feedID)
	case "microsoft_365":
		return s.SyncMicrosoft365(ctx, feedID)
	case "gmail":
		return s.SyncGmail(ctx, feedID)
	case "intercom":
		return s.SyncIntercom(ctx, feedID)
	case "google_drive":
		return s.SyncGoogleDrive(ctx, feedID)
	case "confluence":
		return s.SyncConfluence(ctx, feedID)
	case "asana":
		return s.SyncAsana(ctx, feedID)
	case "linear":
		return s.SyncLinear(ctx, feedID)
	case "hubspot":
		return s.SyncHubSpot(ctx, feedID)
	case "zendesk":
		return s.SyncZendesk(ctx, feedID)
	case "http_url":
		return s.SyncHTTPURL(ctx, feedID)
	case "filesystem":
		return s.SyncFilesystem(ctx, feedID)
	default:
		return nil, fmt.Errorf("manual sync not implemented for connector type %q", conn.Type)
	}
}
