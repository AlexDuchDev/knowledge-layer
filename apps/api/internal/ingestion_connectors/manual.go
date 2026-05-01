package ingestion_connectors

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/knowledgelayer/api/internal/blobstore"
	"github.com/knowledgelayer/api/internal/ingestion_connectors/families/docs_wiki"
)

// ConnectorTypeManual is the connectors.type value for manually-uploaded
// content. The manual adapter validates feeds of this type and the four
// IngestManual* methods on Service write artifacts under it.
const ConnectorTypeManual = "manual"

// Artifact types written to raw_artifacts.artifact_type for manual uploads.
const (
	ArtifactManualText    = "manual_text"
	ArtifactManualFile    = "manual_file"
	ArtifactManualURL     = "manual_url"
	ArtifactManualYouTube = "manual_youtube"
)

// ManualCollectionConfig is the shape stored under
// source_feeds.connector_config_json.collection for a manual feed. Both
// fields are user-editable from the UI; only Label is required.
type ManualCollectionConfig struct {
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

// ParseManualCollectionConfig reads a manual feed's connector_config_json.
// Returns the zero value with no error when the field is empty (legacy /
// unconfigured) — the adapter's ValidateSourceFeedConfig still rejects an
// empty Label, so unconfigured feeds cannot be activated.
func ParseManualCollectionConfig(raw json.RawMessage) (ManualCollectionConfig, error) {
	var cfg struct {
		Collection ManualCollectionConfig `json:"collection"`
	}
	if len(raw) == 0 {
		return ManualCollectionConfig{}, nil
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return ManualCollectionConfig{}, fmt.Errorf("manual: parse collection config: %w", err)
	}
	return cfg.Collection, nil
}

// MarshalManualCollectionConfig produces the JSON shape expected at
// create/patch time.
func MarshalManualCollectionConfig(cfg ManualCollectionConfig) (json.RawMessage, error) {
	return json.Marshal(map[string]any{"collection": cfg})
}

// SetBlobStore wires the optional blob store the manual connector uses to
// persist original file bytes. Other connectors are unaffected.
func (s *Service) SetBlobStore(b blobstore.BlobStore) { s.blob = b }

// ManualCollectionView is the response shape for /api/manual/collections — a
// thin denormalization of source_feed + parsed collection config + counts.
type ManualCollectionView struct {
	FeedID           uuid.UUID  `json:"feed_id"`
	Label            string     `json:"label"`
	Description      string     `json:"description"`
	DomainID         uuid.UUID  `json:"domain_id"`
	SensitivityLevel int        `json:"sensitivity_level"`
	OwnerID          uuid.UUID  `json:"owner_id"`
	Status           string     `json:"status"`
	ArtifactCount    int        `json:"artifact_count"`
	LastUploadAt     *time.Time `json:"last_upload_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}

// CreateManualCollectionInput is what the upload UI submits to create a new
// manual collection. The connector ID is resolved server-side, so callers
// don't need to know it.
type CreateManualCollectionInput struct {
	Label            string    `json:"label"`
	Description      string    `json:"description,omitempty"`
	DomainID         uuid.UUID `json:"domain_id"`
	SensitivityLevel int       `json:"sensitivity_level"`
	OwnerID          uuid.UUID `json:"owner_id,omitempty"`
}

// defaultManualAllowedJobTypes lists the job types a manual collection is
// allowed to feed. Aligned with the docs_page record_type that all four
// artifact normalizers emit; jobs that consume chats/calendar/meetings are
// intentionally absent (manual content is doc-shaped).
var defaultManualAllowedJobTypes = []string{
	"weekly_digest",
	"entity_summarize",
	"decision_extraction",
	"planning_summary",
	"stale_scan",
	"support_trends_extraction",
}

// CreateManualCollection creates and activates a source_feed for the manual
// connector. The feed is immediately ready for uploads — no separate
// activation step.
func (s *Service) CreateManualCollection(ctx context.Context, in CreateManualCollectionInput) (*SourceFeed, error) {
	if strings.TrimSpace(in.Label) == "" {
		return nil, errors.New("label is required")
	}
	if in.OwnerID == uuid.Nil {
		return nil, errors.New("owner_id is required")
	}
	if in.DomainID == uuid.Nil {
		return nil, errors.New("domain_id is required")
	}
	conn, err := s.lookupManualConnector(ctx)
	if err != nil {
		return nil, err
	}
	cfgJSON, err := MarshalManualCollectionConfig(ManualCollectionConfig{
		Label:       strings.TrimSpace(in.Label),
		Description: strings.TrimSpace(in.Description),
	})
	if err != nil {
		return nil, fmt.Errorf("marshal collection config: %w", err)
	}
	allowedJobs, err := json.Marshal(defaultManualAllowedJobTypes)
	if err != nil {
		return nil, err
	}
	create := CreateSourceFeedInput{
		ConnectorID:         conn.ID,
		DisplayName:         in.Label,
		OwnerID:             in.OwnerID,
		DomainID:            in.DomainID,
		SensitivityLevel:    in.SensitivityLevel,
		AllowedJobTypesJSON: allowedJobs,
		IngestionMode:       "ingestion_only",
		SyncMode:            SyncModeManual,
		ExternalRef:         uuid.NewString(),
		KnowledgeScope:      "domain_linked",
		ConnectorConfigJSON: cfgJSON,
	}
	feed, err := s.CreateSourceFeed(ctx, create)
	if err != nil {
		return nil, err
	}
	if err := s.Activate(ctx, feed.ID); err != nil {
		return nil, fmt.Errorf("activate manual collection: %w", err)
	}
	return s.GetSourceFeed(ctx, feed.ID)
}

// ListManualCollections returns manual-collection feeds visible in the given
// domains. Counts and last-upload timestamps are joined from raw_artifacts.
func (s *Service) ListManualCollections(ctx context.Context, domainIDs []uuid.UUID, limit int) ([]ManualCollectionView, error) {
	if len(domainIDs) == 0 {
		return []ManualCollectionView{}, nil
	}
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	rows, err := s.pool.Query(ctx, `
		SELECT f.id, f.connector_config_json, f.domain_id, f.sensitivity_level, f.owner_id, f.status,
		       f.created_at,
		       (SELECT COUNT(*) FROM raw_artifacts r WHERE r.source_feed_id = f.id) AS artifact_count,
		       (SELECT MAX(r.created_at) FROM raw_artifacts r WHERE r.source_feed_id = f.id) AS last_upload_at
		FROM source_feeds f
		JOIN connectors c ON c.id = f.connector_id
		WHERE c.type = $1
		  AND f.domain_id = ANY($2)
		  AND f.status <> 'archived'
		ORDER BY f.created_at DESC
		LIMIT $3`, ConnectorTypeManual, domainIDs, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]ManualCollectionView, 0)
	for rows.Next() {
		var (
			v          ManualCollectionView
			rawCfg     json.RawMessage
			lastUpload *time.Time
		)
		if err := rows.Scan(&v.FeedID, &rawCfg, &v.DomainID, &v.SensitivityLevel, &v.OwnerID, &v.Status,
			&v.CreatedAt, &v.ArtifactCount, &lastUpload); err != nil {
			return nil, err
		}
		cfg, _ := ParseManualCollectionConfig(rawCfg)
		v.Label = cfg.Label
		v.Description = cfg.Description
		v.LastUploadAt = lastUpload
		out = append(out, v)
	}
	return out, rows.Err()
}

// GetManualCollection returns a single collection view including counts.
func (s *Service) GetManualCollection(ctx context.Context, feedID uuid.UUID) (*ManualCollectionView, error) {
	feed, err := s.GetSourceFeed(ctx, feedID)
	if err != nil {
		return nil, err
	}
	conn, err := s.GetConnector(ctx, feed.ConnectorID)
	if err != nil {
		return nil, err
	}
	if conn.Type != ConnectorTypeManual {
		return nil, fmt.Errorf("source_feed %s is not a manual collection (type=%s)", feedID, conn.Type)
	}
	cfg, _ := ParseManualCollectionConfig(feed.ConnectorConfigJSON)
	view := &ManualCollectionView{
		FeedID:           feed.ID,
		Label:            cfg.Label,
		Description:      cfg.Description,
		DomainID:         feed.DomainID,
		SensitivityLevel: feed.SensitivityLevel,
		OwnerID:          feed.OwnerID,
		Status:           feed.Status,
	}
	var (
		count      int
		lastUpload *time.Time
		createdAt  time.Time
	)
	row := s.pool.QueryRow(ctx, `
		SELECT
			(SELECT created_at FROM source_feeds WHERE id=$1),
			(SELECT COUNT(*) FROM raw_artifacts WHERE source_feed_id=$1),
			(SELECT MAX(created_at) FROM raw_artifacts WHERE source_feed_id=$1)`, feedID)
	if err := row.Scan(&createdAt, &count, &lastUpload); err != nil {
		return nil, err
	}
	view.CreatedAt = createdAt
	view.ArtifactCount = count
	view.LastUploadAt = lastUpload
	return view, nil
}

// ManualArtifactSummary is a compact projection of a raw_artifact for the
// collection-detail UI. Title and warnings are pulled from metadata_json.
type ManualArtifactSummary struct {
	ID         uuid.UUID `json:"id"`
	Type       string    `json:"type"`
	Title      string    `json:"title"`
	SourceRef  string    `json:"source_ref,omitempty"`
	Warnings   []string  `json:"warnings,omitempty"`
	Normalized bool      `json:"normalized"`
	CreatedAt  time.Time `json:"created_at"`
}

// ListManualArtifacts returns recent uploads for a collection.
func (s *Service) ListManualArtifacts(ctx context.Context, feedID uuid.UUID, limit int) ([]ManualArtifactSummary, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx, `
		SELECT r.id, r.artifact_type, r.external_artifact_id, r.metadata_json, r.created_at,
		       EXISTS(SELECT 1 FROM normalized_records n WHERE n.raw_artifact_id = r.id) AS normalized
		FROM raw_artifacts r
		WHERE r.source_feed_id = $1
		ORDER BY r.created_at DESC
		LIMIT $2`, feedID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]ManualArtifactSummary, 0)
	for rows.Next() {
		var (
			row      ManualArtifactSummary
			extID    *string
			metaJSON json.RawMessage
		)
		if err := rows.Scan(&row.ID, &row.Type, &extID, &metaJSON, &row.CreatedAt, &row.Normalized); err != nil {
			return nil, err
		}
		if extID != nil {
			row.SourceRef = *extID
		}
		row.Title, row.Warnings = extractManualSummaryFromMeta(metaJSON)
		out = append(out, row)
	}
	return out, rows.Err()
}

// IngestManualTextInput is the request shape for inline text uploads.
type IngestManualTextInput struct {
	Title             string
	Body              string
	SourceAttribution string
	UploaderID        uuid.UUID
}

func (s *Service) IngestManualText(ctx context.Context, feedID uuid.UUID, in IngestManualTextInput) (*RawArtifact, error) {
	if strings.TrimSpace(in.Body) == "" {
		return nil, errors.New("body is required")
	}
	feed, conn, err := s.requireManualFeed(ctx, feedID)
	if err != nil {
		return nil, err
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = manualFirstLine(in.Body, 80)
	}
	if title == "" {
		title = "Pasted text"
	}
	extra := map[string]any{
		"manual_payload": map[string]any{
			"title":              title,
			"content_text":       in.Body,
			"source_attribution": in.SourceAttribution,
			"uploader_id":        in.UploaderID.String(),
			"upload_kind":        "text",
		},
	}
	raw, err := s.persistManualArtifact(ctx, feed, conn, ArtifactManualText, "", in.UploaderID, []byte(in.Body), extra)
	if err != nil {
		return raw, err
	}
	if err := s.normalizeManualToDocsPage(ctx, raw, title, in.Body, ArtifactManualText); err != nil {
		return raw, err
	}
	return raw, nil
}

// IngestManualFileInput is the request shape for binary file uploads.
type IngestManualFileInput struct {
	Filename   string
	MimeType   string // optional override; auto-detected when empty
	Data       []byte
	UploaderID uuid.UUID
}

func (s *Service) IngestManualFile(ctx context.Context, feedID uuid.UUID, in IngestManualFileInput) (*RawArtifact, error) {
	if len(in.Data) == 0 {
		return nil, errors.New("file is empty")
	}
	if int64(len(in.Data)) > MaxManualUploadSize {
		return nil, fmt.Errorf("file exceeds max size of %d bytes", MaxManualUploadSize)
	}
	feed, conn, err := s.requireManualFeed(ctx, feedID)
	if err != nil {
		return nil, err
	}
	mime := strings.TrimSpace(in.MimeType)
	if mime == "" {
		head := in.Data
		if len(head) > 512 {
			head = head[:512]
		}
		mime = DetectManualMimeType(in.Filename, head)
	}
	res, err := ExtractManualBytes(in.Filename, mime, in.Data)
	if err != nil {
		return nil, err
	}
	storageURI := ""
	if s.blob != nil {
		key := fmt.Sprintf("manual/%s/%s/%s", feed.DomainID, feed.ID, uuid.NewString())
		uri, perr := s.blob.Put(ctx, key, mime, bytes.NewReader(in.Data), int64(len(in.Data)))
		if perr != nil {
			return nil, fmt.Errorf("blobstore put: %w", perr)
		}
		storageURI = uri
	}
	extra := map[string]any{
		"manual_payload": map[string]any{
			"title":             res.Title,
			"content_text":      res.BodyText,
			"original_filename": in.Filename,
			"mime_type":         mime,
			"size_bytes":        len(in.Data),
			"storage_uri":       storageURI,
			"warnings":          res.Warnings,
			"uploader_id":       in.UploaderID.String(),
			"upload_kind":       "file",
		},
	}
	raw, err := s.persistManualArtifact(ctx, feed, conn, ArtifactManualFile, in.Filename, in.UploaderID, in.Data, extra)
	if err != nil {
		return raw, err
	}
	if err := s.normalizeManualToDocsPage(ctx, raw, res.Title, res.BodyText, ArtifactManualFile); err != nil {
		return raw, err
	}
	if storageURI != "" {
		_, _ = s.pool.Exec(ctx, `UPDATE raw_artifacts SET storage_uri=$2 WHERE id=$1`, raw.ID, storageURI)
		raw.StorageURI = storageURI
	}
	return raw, nil
}

// IngestManualURL fetches a single web page, extracts readable text, and persists.
func (s *Service) IngestManualURL(ctx context.Context, feedID uuid.UUID, pageURL string, uploaderID uuid.UUID) (*RawArtifact, error) {
	pageURL = strings.TrimSpace(pageURL)
	if pageURL == "" {
		return nil, errors.New("url is required")
	}
	feed, conn, err := s.requireManualFeed(ctx, feedID)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}
	req.Header.Set("User-Agent", "KnowledgeLayerBot/1.0")
	resp, err := s.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch url: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	text, title := ExtractManualHTML(body)
	if strings.TrimSpace(title) == "" {
		title = pageURL
	}
	warnings := []string{}
	if strings.TrimSpace(text) == "" {
		warnings = append(warnings, "url returned no extractable text")
	}
	extra := map[string]any{
		"manual_payload": map[string]any{
			"title":        title,
			"content_text": text,
			"source_url":   pageURL,
			"http_status":  resp.StatusCode,
			"uploader_id":  uploaderID.String(),
			"warnings":     warnings,
			"upload_kind":  "url",
		},
	}
	raw, err := s.persistManualArtifact(ctx, feed, conn, ArtifactManualURL, pageURL, uploaderID, body, extra)
	if err != nil {
		return raw, err
	}
	if err := s.normalizeManualToDocsPage(ctx, raw, title, text, ArtifactManualURL); err != nil {
		return raw, err
	}
	return raw, nil
}

// IngestManualYouTube resolves a YouTube URL/ID, fetches the best caption
// track, and persists the transcript as a docs_page-shaped normalized record.
func (s *Service) IngestManualYouTube(ctx context.Context, feedID uuid.UUID, urlOrID string, uploaderID uuid.UUID) (*RawArtifact, error) {
	feed, conn, err := s.requireManualFeed(ctx, feedID)
	if err != nil {
		return nil, err
	}
	yt, err := FetchManualYouTube(ctx, urlOrID, s.HTTP)
	if err != nil {
		return nil, err
	}
	title := yt.Title
	if title == "" {
		title = "YouTube video " + yt.VideoID
	}
	extra := map[string]any{
		"manual_payload": map[string]any{
			"title":            title,
			"content_text":     yt.BodyText,
			"video_id":         yt.VideoID,
			"video_url":        yt.WebURL,
			"author":           yt.Author,
			"duration_seconds": int(yt.Duration.Seconds()),
			"thumbnail":        yt.Thumbnail,
			"caption_lang":     yt.CaptionLang,
			"published_at":     yt.PublishedAt,
			"uploader_id":      uploaderID.String(),
			"warnings":         yt.Warnings,
			"upload_kind":      "youtube",
		},
	}
	raw, err := s.persistManualArtifact(ctx, feed, conn, ArtifactManualYouTube, yt.VideoID, uploaderID, []byte(yt.BodyText), extra)
	if err != nil {
		return raw, err
	}
	if err := s.normalizeManualToDocsPage(ctx, raw, title, yt.BodyText, ArtifactManualYouTube); err != nil {
		return raw, err
	}
	return raw, nil
}

// PatchManualCollectionInput is the partial-update payload for collection
// metadata. nil fields are left untouched; non-nil-but-empty Label is rejected.
type PatchManualCollectionInput struct {
	Label       *string
	Description *string
}

// PatchManualCollection updates the collection's label and/or description.
// Other source_feed fields (domain, sensitivity, owner) are intentionally
// not editable through this entry point — change those through the source
// feed admin surface so governance audit trails stay consistent.
func (s *Service) PatchManualCollection(ctx context.Context, feedID uuid.UUID, in PatchManualCollectionInput) (*ManualCollectionView, error) {
	feed, _, err := s.requireManualFeed(ctx, feedID)
	if err != nil {
		return nil, err
	}
	cfg, _ := ParseManualCollectionConfig(feed.ConnectorConfigJSON)
	if in.Label != nil {
		l := strings.TrimSpace(*in.Label)
		if l == "" {
			return nil, errors.New("label cannot be empty")
		}
		cfg.Label = l
	}
	if in.Description != nil {
		cfg.Description = strings.TrimSpace(*in.Description)
	}
	cfgJSON, err := MarshalManualCollectionConfig(cfg)
	if err != nil {
		return nil, err
	}
	// PatchSourceFeed handles display_name + connector_config_json. We update
	// the display name to the new label so the source-feed surface and the
	// collection surface show the same string.
	displayName := cfg.Label
	if _, err := s.PatchSourceFeed(ctx, feedID, &displayName, cfgJSON); err != nil {
		return nil, err
	}
	return s.GetManualCollection(ctx, feedID)
}

// DeleteManualArtifact removes a single raw_artifact and (via FK cascade)
// its normalized_record, chunks, and embeddings. The artifact must belong to
// the supplied feed; we enforce that to prevent cross-feed deletes via id-guess.
func (s *Service) DeleteManualArtifact(ctx context.Context, feedID, artifactID uuid.UUID) error {
	if _, _, err := s.requireManualFeed(ctx, feedID); err != nil {
		return err
	}
	tag, err := s.pool.Exec(ctx, `DELETE FROM raw_artifacts WHERE id=$1 AND source_feed_id=$2`, artifactID, feedID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("artifact not found in this collection")
	}
	return nil
}

// RenormalizeManualArtifact rebuilds the normalized_record (and downstream
// chunks/embeddings via the persistence hook) for a single artifact from its
// stored metadata_json. Use this when extraction succeeded but later code
// produces better results, or when the operator manually fixed metadata.
//
// Existing normalized_records and chunks for this artifact are removed first
// so the rebuild produces a clean replacement rather than a stale duplicate.
func (s *Service) RenormalizeManualArtifact(ctx context.Context, feedID, artifactID uuid.UUID) error {
	if _, _, err := s.requireManualFeed(ctx, feedID); err != nil {
		return err
	}
	raw, err := s.GetRawArtifact(ctx, artifactID)
	if err != nil {
		return err
	}
	if raw.SourceFeedID != feedID {
		return errors.New("artifact does not belong to this collection")
	}
	// FK cascade on normalized_records → chunks → embeddings does the cleanup.
	if _, err := s.pool.Exec(ctx, `DELETE FROM normalized_records WHERE raw_artifact_id=$1`, artifactID); err != nil {
		return fmt.Errorf("clear normalized_records: %w", err)
	}
	return s.normalizeManualArtifactFromMeta(ctx, raw)
}

// ManualSearchHit is one row of an in-collection search result.
type ManualSearchHit struct {
	ArtifactID    uuid.UUID `json:"artifact_id"`
	ArtifactType  string    `json:"artifact_type"`
	ArtifactTitle string    `json:"artifact_title"`
	ChunkID       uuid.UUID `json:"chunk_id"`
	Ordinal       int       `json:"ordinal"`
	Snippet       string    `json:"snippet"`
}

// SearchManualCollection runs a keyword search across the chunks of a single
// collection. Designed for the in-collection "search this" box — fast and
// SQL-only, no embeddings dependency. For broader semantic search, the global
// /api/search and /api/ask endpoints already cover collections via the
// existing domain-permission flow.
func (s *Service) SearchManualCollection(ctx context.Context, feedID uuid.UUID, query string, limit int) ([]ManualSearchHit, error) {
	if _, _, err := s.requireManualFeed(ctx, feedID); err != nil {
		return nil, err
	}
	q := strings.TrimSpace(query)
	if q == "" {
		return []ManualSearchHit{}, nil
	}
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	pattern := "%" + strings.ReplaceAll(strings.ReplaceAll(q, `\`, `\\`), `%`, `\%`) + "%"
	rows, err := s.pool.Query(ctx, `
		SELECT r.id, r.artifact_type, r.metadata_json, c.id, c.ordinal, c.text_content
		FROM chunks c
		JOIN normalized_records n ON n.id = c.normalized_record_id
		JOIN raw_artifacts r ON r.id = n.raw_artifact_id
		WHERE n.source_feed_id = $1 AND c.text_content ILIKE $2
		ORDER BY r.created_at DESC, c.ordinal ASC
		LIMIT $3`, feedID, pattern, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	hits := make([]ManualSearchHit, 0)
	for rows.Next() {
		var (
			h        ManualSearchHit
			metaJSON json.RawMessage
			text     string
		)
		if err := rows.Scan(&h.ArtifactID, &h.ArtifactType, &metaJSON, &h.ChunkID, &h.Ordinal, &text); err != nil {
			return nil, err
		}
		title, _ := extractManualSummaryFromMeta(metaJSON)
		h.ArtifactTitle = title
		h.Snippet = manualSnippet(text, q, 240)
		hits = append(hits, h)
	}
	return hits, rows.Err()
}

// manualSnippet returns at most maxLen characters of `text`, centered on the
// first match of `query` (case-insensitive). Falls back to the leading slice
// when query is not found (shouldn't happen given ILIKE, but defensive).
func manualSnippet(text, query string, maxLen int) string {
	t := text
	idx := strings.Index(strings.ToLower(t), strings.ToLower(query))
	if idx < 0 {
		if len(t) > maxLen {
			return strings.TrimSpace(t[:maxLen]) + "…"
		}
		return strings.TrimSpace(t)
	}
	start := idx - maxLen/3
	if start < 0 {
		start = 0
	}
	end := start + maxLen
	if end > len(t) {
		end = len(t)
	}
	prefix := ""
	if start > 0 {
		prefix = "…"
	}
	suffix := ""
	if end < len(t) {
		suffix = "…"
	}
	return prefix + strings.TrimSpace(t[start:end]) + suffix
}

// requireManualFeed loads the feed, enforces it's a manual feed, and returns
// the joined connector. Permission checks are the route layer's job; this
// only enforces the type contract.
func (s *Service) requireManualFeed(ctx context.Context, feedID uuid.UUID) (*SourceFeed, *Connector, error) {
	feed, err := s.GetSourceFeed(ctx, feedID)
	if err != nil {
		return nil, nil, err
	}
	if feed.Status == "archived" {
		return nil, nil, errors.New("collection is archived")
	}
	conn, err := s.GetConnector(ctx, feed.ConnectorID)
	if err != nil {
		return nil, nil, err
	}
	if conn.Type != ConnectorTypeManual {
		return nil, nil, fmt.Errorf("source_feed %s is not a manual collection (type=%s)", feedID, conn.Type)
	}
	return feed, conn, nil
}

// persistManualArtifact runs the start-run / build-metadata / insert-row /
// finalize-run sequence shared by all four ingest methods. The caller passes
// the artifact-type-specific `extra` payload that goes into metadata_json.
func (s *Service) persistManualArtifact(
	ctx context.Context,
	feed *SourceFeed,
	conn *Connector,
	artifactType string,
	externalRef string,
	uploaderID uuid.UUID,
	rawPayload []byte,
	extra map[string]any,
) (*RawArtifact, error) {
	runID, err := s.startIngestionRun(ctx, feed.ID)
	if err != nil {
		return nil, err
	}
	metaJSON, err := s.buildRawArtifactMetadataJSON(ctx, *conn, *feed, artifactType, rawPayload, extra)
	if err != nil {
		s.finalizeIngestionRun(ctx, runID, "failed", 0, 0, 0, 1)
		s.completeSourceFeedSync(ctx, feed.ID, 1, false)
		return nil, err
	}
	hash := hashBytes(metaJSON)
	now := time.Now().UTC()
	uploader := uploaderID.String()
	rawID, inserted, err := insertRawArtifactRow(ctx, s.pool, feed.ID, runID, artifactType, externalRef, hash, "", metaJSON, &now, &uploader)
	if err != nil {
		s.finalizeIngestionRun(ctx, runID, "failed", 0, 0, 0, 1)
		s.completeSourceFeedSync(ctx, feed.ID, 1, false)
		return nil, err
	}
	if !inserted {
		// Duplicate: dedup is on (source_feed_id, content_hash). Look up the
		// existing row so the caller can show "already uploaded" rather than
		// silently swallowing.
		s.finalizeIngestionRun(ctx, runID, "completed", 0, 1, 0, 0)
		s.completeSourceFeedSync(ctx, feed.ID, 0, false)
		existing, lookupErr := s.findExistingRawArtifact(ctx, feed.ID, hash)
		if lookupErr != nil {
			return nil, lookupErr
		}
		return existing, ErrManualArtifactDuplicate
	}
	s.finalizeIngestionRun(ctx, runID, "completed", 1, 0, 0, 0)
	s.completeSourceFeedSync(ctx, feed.ID, 0, false)
	return s.GetRawArtifact(ctx, rawID)
}

// ErrManualArtifactDuplicate is returned by upload methods when content_hash
// collides with an existing artifact in the same feed. Callers should map
// this to HTTP 200 with deduped=true rather than 4xx — the upload itself is
// valid, just redundant.
var ErrManualArtifactDuplicate = errors.New("manual: duplicate artifact (content already in collection)")

func (s *Service) findExistingRawArtifact(ctx context.Context, feedID uuid.UUID, contentHash string) (*RawArtifact, error) {
	var id uuid.UUID
	err := s.pool.QueryRow(ctx, `SELECT id FROM raw_artifacts WHERE source_feed_id=$1 AND content_hash=$2`, feedID, contentHash).Scan(&id)
	if err != nil {
		return nil, err
	}
	return s.GetRawArtifact(ctx, id)
}

// normalizeManualToDocsPage builds a NormalizedDocPage from the extracted
// title/body and inserts it via PersistNormalizedRecord so the chunks +
// embeddings hook fires uniformly with other connectors.
func (s *Service) normalizeManualToDocsPage(ctx context.Context, raw *RawArtifact, title, body, artifactType string) error {
	now := time.Now().UTC()
	norm := docs_wiki.NormalizedDocPage{
		SourceFeedID:    raw.SourceFeedID,
		ConnectorFamily: "docs_wiki",
		ConnectorType:   ConnectorTypeManual,
		Title:           firstNonEmptyString(title, "Untitled"),
		ExternalRef:     raw.ExternalArtifactIDOrFallback(),
		LastModifiedAt:  &now,
		BodyText:        body,
		DownstreamHint: map[string]any{
			"suggested_entity_type": "ReferenceDocument",
			"default_truth_mode":    "mirrored_authority",
			"manual_artifact_type":  artifactType,
		},
	}
	normPayload, err := json.Marshal(norm)
	if err != nil {
		return err
	}
	_, _, err = s.PersistNormalizedRecord(ctx, raw.ID, raw.SourceFeedID, docs_wiki.RecordTypeDocsPage, normPayload, hashBytes(normPayload), &now, raw.SourceAuthorRef)
	return err
}

// normalizeManualArtifactFromMeta re-creates the normalized record from the
// stored metadata_json. Called by the artifact worker when an upload's
// inline normalization failed and Asynq retries the queued task. The
// metadata_json already carries the extracted title + body, so we don't
// re-fetch / re-extract.
func (s *Service) normalizeManualArtifactFromMeta(ctx context.Context, raw *RawArtifact) error {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw.MetadataJSON, &top); err != nil {
		return fmt.Errorf("manual raw metadata: %w", err)
	}
	payloadRaw, ok := top["manual_payload"]
	if !ok {
		return errors.New("manual_payload missing from raw metadata_json")
	}
	var payload struct {
		Title       string `json:"title"`
		ContentText string `json:"content_text"`
	}
	if err := json.Unmarshal(payloadRaw, &payload); err != nil {
		return fmt.Errorf("manual_payload: %w", err)
	}
	title := payload.Title
	if strings.TrimSpace(title) == "" {
		title = "Untitled"
	}
	return s.normalizeManualToDocsPage(ctx, raw, title, payload.ContentText, raw.ArtifactType)
}

// lookupManualConnector returns the row in `connectors` with type='manual'.
// Migration 000045 seeds one; if it is missing we surface a clear error
// rather than auto-creating one (multiple rows would break adapter resolution).
func (s *Service) lookupManualConnector(ctx context.Context) (*Connector, error) {
	var c Connector
	err := s.pool.QueryRow(ctx, `
		SELECT id, type, display_name, auth_mode, status, auth_config_ref, capabilities_json, config_json
		FROM connectors WHERE type=$1`, ConnectorTypeManual).
		Scan(&c.ID, &c.Type, &c.DisplayName, &c.AuthMode, &c.Status, &c.AuthConfigRef, &c.CapabilitiesJSON, &c.ConfigJSON)
	if err != nil {
		return nil, fmt.Errorf("manual connector row not found (run db migrations): %w", err)
	}
	return &c, nil
}

func extractManualSummaryFromMeta(metaJSON json.RawMessage) (string, []string) {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(metaJSON, &top); err != nil {
		return "", nil
	}
	payloadRaw, ok := top["manual_payload"]
	if !ok {
		return "", nil
	}
	var payload struct {
		Title    string   `json:"title"`
		Warnings []string `json:"warnings"`
	}
	_ = json.Unmarshal(payloadRaw, &payload)
	return payload.Title, payload.Warnings
}

func manualFirstLine(s string, max int) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, "\n\r"); i > 0 {
		s = s[:i]
	}
	if len(s) > max {
		s = s[:max]
	}
	return strings.TrimSpace(s)
}
