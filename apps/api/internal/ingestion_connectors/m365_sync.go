package ingestion_connectors

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/knowledgelayer/api/internal/ingestion_connectors/families/microsoft365"
)

const (
	m365MailBodyMaxRunes = 32000
	m365FilesMaxItems    = 25
	m365CalendarMaxItems = 50
)

type m365FeedConfig struct {
	Product       string `json:"m365_product"` // outlook | teams | onedrive | sharepoint | calendar
	AccessToken   string `json:"graph_access_token"`
	RefreshToken  string `json:"graph_refresh_token,omitempty"`
	ExpiryRFC3339 string `json:"graph_token_expiry,omitempty"`
	// Files (OneDrive / SharePoint)
	M365FilesScope       string `json:"m365_files_scope,omitempty"` // folder | library | subtree | search
	M365SearchQuery      string `json:"m365_search_query,omitempty"`
	M365SearchMaxResults int    `json:"m365_search_max_results,omitempty"`
	// Outlook mail
	MailFolderID     string `json:"mail_folder_id,omitempty"`
	SharedMailboxUPN string `json:"shared_mailbox_upn,omitempty"`
	MailFilterQuery  string `json:"mail_filter_query,omitempty"`
	// Calendar
	TimeWindowHours int `json:"time_window_hours,omitempty"` // default 168, max 720
}

// ValidateMicrosoft365SourceFeedForActivation validates Graph token and product for active feeds.
func ValidateMicrosoft365SourceFeedForActivation(feed *SourceFeed) error {
	if feed == nil {
		return errors.New("microsoft_365: nil feed")
	}
	cfg, err := parseM365FeedConfig(feed.ConnectorConfigJSON)
	if err != nil {
		return err
	}
	switch cfg.Product {
	case "teams":
		parts := strings.Split(feed.ExternalRef, "|")
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
			return errors.New("microsoft_365 teams: external_ref must be team_id|channel_id")
		}
	case "onedrive", "sharepoint":
		scope := strings.ToLower(strings.TrimSpace(cfg.M365FilesScope))
		if scope == "" {
			scope = "folder"
		}
		if scope == "search" {
			if strings.TrimSpace(cfg.M365SearchQuery) == "" {
				return errors.New("microsoft_365 files: m365_search_query required when m365_files_scope is search")
			}
			break
		}
		if _, err := m365DriveChildrenGraphPath(feed.ExternalRef); err != nil {
			return err
		}
	case "calendar":
		// external_ref empty or primary|calendar-id both OK
	default:
		// outlook — no extra ref rules
	}
	return nil
}

func parseM365FeedConfig(raw json.RawMessage) (*m365FeedConfig, error) {
	var c m365FeedConfig
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, err
	}
	c.Product = strings.ToLower(strings.TrimSpace(c.Product))
	if strings.TrimSpace(c.AccessToken) == "" && strings.TrimSpace(c.RefreshToken) == "" {
		return nil, errors.New("microsoft_365: graph_access_token or graph_refresh_token required in connector_config_json")
	}
	switch c.Product {
	case "outlook", "teams", "onedrive", "sharepoint", "calendar":
	default:
		return nil, errors.New("microsoft_365: m365_product must be outlook, teams, onedrive, sharepoint, or calendar")
	}
	return &c, nil
}

// SyncMicrosoft365 ingests Outlook mail or Teams channel messages using a delegated Graph access token (v1).
func (s *Service) SyncMicrosoft365(ctx context.Context, feedID uuid.UUID) (*IngestionRun, error) {
	feed, err := s.GetSourceFeed(ctx, feedID)
	if err != nil {
		return nil, err
	}
	conn, err := s.GetConnector(ctx, feed.ConnectorID)
	if err != nil {
		return nil, err
	}
	if conn.Type != "microsoft_365" {
		return nil, fmt.Errorf("connector is %s, not microsoft_365", conn.Type)
	}
	if feed.Status != "active" {
		return nil, errors.New("source feed must be active")
	}
	cfg, err := parseM365FeedConfig(feed.ConnectorConfigJSON)
	if err != nil {
		return nil, err
	}
	if err := s.resolveM365AccessToken(ctx, cfg); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cfg.AccessToken) == "" {
		return nil, errors.New("microsoft_365: no graph access token after refresh")
	}

	runID, err := s.startIngestionRun(ctx, feedID)
	if err != nil {
		return nil, err
	}

	switch cfg.Product {
	case "outlook":
		return s.syncM365Outlook(ctx, feedID, runID, feed, conn, cfg)
	case "teams":
		return s.syncM365Teams(ctx, feedID, runID, feed, conn, cfg)
	case "onedrive", "sharepoint":
		return s.syncM365OneDriveSharePoint(ctx, feedID, runID, feed, conn, cfg)
	case "calendar":
		return s.syncM365Calendar(ctx, feedID, runID, feed, conn, cfg)
	default:
		s.finalizeIngestionRun(ctx, runID, "failed", 0, 0, 0, 1)
		s.completeSourceFeedSync(ctx, feedID, 1, false)
		return nil, errors.New("microsoft_365: unknown product")
	}
}

func (s *Service) syncM365Outlook(ctx context.Context, feedID, runID uuid.UUID, feed *SourceFeed, conn *Connector, cfg *m365FeedConfig) (*IngestionRun, error) {
	base := "https://graph.microsoft.com/v1.0"
	var path string
	upn := strings.TrimSpace(cfg.SharedMailboxUPN)
	folder := strings.TrimSpace(cfg.MailFolderID)
	if upn != "" {
		esc := url.PathEscape(upn)
		if folder != "" {
			path = fmt.Sprintf("/users/%s/mailFolders/%s/messages", esc, url.PathEscape(folder))
		} else {
			path = fmt.Sprintf("/users/%s/messages", esc)
		}
	} else {
		if folder != "" {
			path = fmt.Sprintf("/me/mailFolders/%s/messages", url.PathEscape(folder))
		} else {
			path = "/me/messages"
		}
	}
	sel := "id,subject,bodyPreview,body,receivedDateTime,sentDateTime,conversationId,internetMessageId,from,toRecipients,ccRecipients,hasAttachments,parentFolderId"
	q := url.Values{}
	q.Set("$top", "15")
	q.Set("$select", sel)
	if fq := strings.TrimSpace(cfg.MailFilterQuery); fq != "" {
		q.Set("$filter", fq)
	}
	u := base + path + "?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		s.finalizeIngestionRun(ctx, runID, "failed", 0, 0, 0, 1)
		s.completeSourceFeedSync(ctx, feedID, 1, false)
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.AccessToken)
	resp, err := s.HTTP.Do(req)
	if err != nil {
		s.finalizeIngestionRun(ctx, runID, "failed", 0, 0, 0, 1)
		s.completeSourceFeedSync(ctx, feedID, 1, false)
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.finalizeIngestionRun(ctx, runID, "failed", 0, 0, 0, 1)
		s.completeSourceFeedSync(ctx, feedID, 1, false)
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		s.finalizeIngestionRun(ctx, runID, "failed", 0, 0, 0, 1)
		s.completeSourceFeedSync(ctx, feedID, 1, false)
		return nil, fmt.Errorf("graph mail: status %d: %s", resp.StatusCode, string(body))
	}
	var parsed struct {
		Value []struct {
			ID                string       `json:"id"`
			Subject           string       `json:"subject"`
			BodyPreview       string       `json:"bodyPreview"`
			Body              m365MailBody `json:"body"`
			ReceivedDateTime  string       `json:"receivedDateTime"`
			SentDateTime      string       `json:"sentDateTime"`
			ConversationID    string       `json:"conversationId"`
			InternetMessageID string       `json:"internetMessageId"`
			HasAttachments    bool         `json:"hasAttachments"`
			ParentFolderID    string       `json:"parentFolderId"`
			From              struct {
				EmailAddress struct {
					Address string `json:"address"`
				} `json:"emailAddress"`
			} `json:"from"`
			ToRecipients []m365Recipient `json:"toRecipients"`
			CcRecipients []m365Recipient `json:"ccRecipients"`
		} `json:"value"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		s.finalizeIngestionRun(ctx, runID, "failed", 0, 0, 0, 1)
		s.completeSourceFeedSync(ctx, feedID, 1, false)
		return nil, err
	}

	ingested, deduped, warnings, errs := 0, 0, 0, 0
	for _, m := range parsed.Value {
		rawPayload, _ := json.Marshal(m)
		h := hashBytes(rawPayload)
		extID := "m365-mail-" + m.ID
		var rawObj map[string]any
		_ = json.Unmarshal(rawPayload, &rawObj)
		extra := map[string]any{"m365_message": rawObj}
		metaJSON, merr := s.buildRawArtifactMetadataJSON(ctx, *conn, *feed, microsoft365.ArtifactKindGraphJSON, rawPayload, extra)
		if merr != nil {
			errs++
			continue
		}
		rawID, inserted, ierr := insertRawArtifactRow(ctx, s.pool, feedID, runID, microsoft365.ArtifactM365MailMessage, extID, h, "", metaJSON, nil, nil)
		if ierr != nil {
			errs++
			continue
		}
		if !inserted {
			deduped++
			continue
		}
		var recv *time.Time
		if m.ReceivedDateTime != "" {
			if t, err := time.Parse(time.RFC3339, m.ReceivedDateTime); err == nil {
				recv = &t
			}
		}
		bodyText := truncateRunes(strings.TrimSpace(m.Body.Content), m365MailBodyMaxRunes)
		var attachRefs []string
		if m.HasAttachments {
			attachRefs = []string{"has_attachments:true"}
		}
		norm := microsoft365.NormalizedMailMessage{
			SourceFeedID:       feedID,
			ConnectorFamily:    "microsoft365",
			ConnectorType:      "outlook",
			ExternalRef:        m.ID,
			ExternalThreadID:   m.ConversationID,
			InternetMessageID:  m.InternetMessageID,
			Subject:            m.Subject,
			BodyPreview:        m.BodyPreview,
			BodyText:           bodyText,
			ReceivedAt:         recv,
			FromRef:            m.From.EmailAddress.Address,
			ToRefs:             m365RecipientEmails(m.ToRecipients),
			CCRefs:             m365RecipientEmails(m.CcRecipients),
			FolderOrMailboxRef: m.ParentFolderID,
			AttachmentRefs:     attachRefs,
		}
		normPayload, _ := json.Marshal(norm)
		nh := hashBytes(normPayload)
		tag, qerr := s.pool.Exec(ctx, `
			INSERT INTO normalized_records (raw_artifact_id, source_feed_id, record_type, structured_payload_json, record_hash)
			VALUES ($1,$2,$3,$4,$5)
			ON CONFLICT (source_feed_id, record_hash) DO NOTHING`,
			rawID, feedID, microsoft365.RecordTypeMailMessage, normPayload, nh)
		if qerr != nil {
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

type m365MailBody struct {
	ContentType string `json:"contentType"`
	Content     string `json:"content"`
}

type m365Recipient struct {
	EmailAddress struct {
		Address string `json:"address"`
	} `json:"emailAddress"`
}

func m365RecipientEmails(rs []m365Recipient) []string {
	if len(rs) == 0 {
		return nil
	}
	out := make([]string, 0, len(rs))
	for _, r := range rs {
		if a := strings.TrimSpace(r.EmailAddress.Address); a != "" {
			out = append(out, a)
		}
	}
	return out
}

func truncateRunes(s string, max int) string {
	if max <= 0 || s == "" {
		return s
	}
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	if len(runes) > max {
		return string(runes[:max]) + "\n\n[truncated at connector rune limit]"
	}
	return s
}

func (s *Service) syncM365Teams(ctx context.Context, feedID, runID uuid.UUID, feed *SourceFeed, conn *Connector, cfg *m365FeedConfig) (*IngestionRun, error) {
	parts := strings.Split(feed.ExternalRef, "|")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		s.finalizeIngestionRun(ctx, runID, "failed", 0, 0, 0, 1)
		s.completeSourceFeedSync(ctx, feedID, 1, false)
		return nil, errors.New("microsoft_365 teams: external_ref must be team_id|channel_id")
	}
	teamID, channelID := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	u := fmt.Sprintf("https://graph.microsoft.com/v1.0/teams/%s/channels/%s/messages?$top=15", teamID, channelID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		s.finalizeIngestionRun(ctx, runID, "failed", 0, 0, 0, 1)
		s.completeSourceFeedSync(ctx, feedID, 1, false)
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.AccessToken)
	resp, err := s.HTTP.Do(req)
	if err != nil {
		s.finalizeIngestionRun(ctx, runID, "failed", 0, 0, 0, 1)
		s.completeSourceFeedSync(ctx, feedID, 1, false)
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.finalizeIngestionRun(ctx, runID, "failed", 0, 0, 0, 1)
		s.completeSourceFeedSync(ctx, feedID, 1, false)
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		s.finalizeIngestionRun(ctx, runID, "failed", 0, 0, 0, 1)
		s.completeSourceFeedSync(ctx, feedID, 1, false)
		return nil, fmt.Errorf("graph teams: status %d: %s", resp.StatusCode, string(body))
	}
	var parsed struct {
		Value []struct {
			ID   string `json:"id"`
			Body struct {
				Content string `json:"content"`
			} `json:"body"`
			CreatedDateTime string `json:"createdDateTime"`
		} `json:"value"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		s.finalizeIngestionRun(ctx, runID, "failed", 0, 0, 0, 1)
		s.completeSourceFeedSync(ctx, feedID, 1, false)
		return nil, err
	}

	ingested, deduped, warnings, errs := 0, 0, 0, 0
	for _, m := range parsed.Value {
		rawPayload, _ := json.Marshal(m)
		h := hashBytes(rawPayload)
		extID := "m365-teams-" + m.ID
		var rawObj map[string]any
		_ = json.Unmarshal(rawPayload, &rawObj)
		extra := map[string]any{"m365_teams_message": rawObj}
		metaJSON, merr := s.buildRawArtifactMetadataJSON(ctx, *conn, *feed, microsoft365.ArtifactKindGraphJSON, rawPayload, extra)
		if merr != nil {
			errs++
			continue
		}
		rawID, inserted, ierr := insertRawArtifactRow(ctx, s.pool, feedID, runID, "m365_teams_message", extID, h, "", metaJSON, nil, nil)
		if ierr != nil {
			errs++
			continue
		}
		if !inserted {
			deduped++
			continue
		}
		norm := microsoft365.NormalizedTeamsMessage{
			SourceFeedID:    feedID,
			ConnectorFamily: "microsoft365",
			ConnectorType:   "teams",
			ExternalRef:     m.ID,
			TeamID:          teamID,
			ChannelID:       channelID,
			BodyPreview:     m.Body.Content,
			CreatedAt:       m.CreatedDateTime,
		}
		normPayload, _ := json.Marshal(norm)
		nh := hashBytes(normPayload)
		tag, qerr := s.pool.Exec(ctx, `
			INSERT INTO normalized_records (raw_artifact_id, source_feed_id, record_type, structured_payload_json, record_hash)
			VALUES ($1,$2,$3,$4,$5)
			ON CONFLICT (source_feed_id, record_hash) DO NOTHING`,
			rawID, feedID, microsoft365.RecordTypeTeamsMessage, normPayload, nh)
		if qerr != nil {
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
