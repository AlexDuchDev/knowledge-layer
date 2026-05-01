package ingestion_connectors

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/knowledgelayer/api/internal/ingestion_connectors/families/chat"
	"github.com/knowledgelayer/api/internal/ingestion_connectors/families/docs_wiki"
	"github.com/knowledgelayer/api/internal/ingestion_connectors/families/meeting"
)

// ProcessQueuedRawArtifact completes normalization for a raw row when it was deferred or retried.
// Idempotent: if a normalized_records row already exists for raw_artifact_id, returns nil.
func (s *Service) ProcessQueuedRawArtifact(ctx context.Context, rawID uuid.UUID) error {
	raw, err := s.GetRawArtifact(ctx, rawID)
	if err != nil {
		return err
	}
	var exists bool
	err = s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM normalized_records WHERE raw_artifact_id=$1)`, rawID).Scan(&exists)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	switch raw.ArtifactType {
	case "telegram_update":
		return s.normalizeTelegramUpdateArtifact(ctx, raw)
	case "slack_message":
		return s.normalizeSlackMessageArtifact(ctx, raw)
	case "mattermost_post":
		return s.normalizeMattermostPostArtifact(ctx, raw)
	case "google_drive_file":
		return s.normalizeGoogleDriveArtifact(ctx, raw)
	case "notion_page":
		return s.normalizeNotionPageArtifact(ctx, raw)
	case docs_wiki.ArtifactConfluencePage:
		return s.normalizeConfluencePageArtifact(ctx, raw)
	case "google_calendar_event":
		return s.normalizeGoogleCalendarEventArtifact(ctx, raw)
	case "fireflies_transcript":
		return s.normalizeFirefliesTranscriptArtifact(ctx, raw)
	case artifactHTTPURLPage:
		return s.normalizeHTTPURLPageArtifact(ctx, raw)
	case artifactFilesystemFile:
		return s.normalizeFilesystemFileArtifact(ctx, raw)
	case "manual_text", "manual_file", "manual_url", "manual_youtube":
		return s.normalizeManualArtifactFromMeta(ctx, raw)
	default:
		// Permanent no-op: avoid Asynq infinite retries for types without async normalizers.
		return nil
	}
}

func (s *Service) normalizeHTTPURLPageArtifact(ctx context.Context, raw *RawArtifact) error {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw.MetadataJSON, &top); err != nil {
		return fmt.Errorf("http_url raw metadata: %w", err)
	}
	pageRaw, ok := top["http_url_page"]
	if !ok {
		return errors.New("http_url_page missing from raw metadata_json")
	}
	var page struct {
		URL         string `json:"url"`
		Title       string `json:"title"`
		ContentText string `json:"content_text"`
	}
	if err := json.Unmarshal(pageRaw, &page); err != nil {
		return fmt.Errorf("http_url_page payload: %w", err)
	}
	if strings.TrimSpace(page.URL) == "" {
		return errors.New("http_url_page missing url")
	}
	norm := docs_wiki.NormalizedDocPage{
		SourceFeedID:    raw.SourceFeedID,
		ConnectorFamily: "docs_wiki",
		ConnectorType:   "http_url",
		Title:           firstNonEmptyString(page.Title, page.URL),
		ExternalRef:     page.URL,
		LastModifiedAt:  raw.SourceCreatedAt,
		BodyText:        page.ContentText,
		WebViewLink:     page.URL,
		DownstreamHint: map[string]any{
			"suggested_entity_type": "ReferenceDocument",
			"default_truth_mode":    "mirrored_authority",
		},
	}
	normPayload, err := json.Marshal(norm)
	if err != nil {
		return err
	}
	return s.insertNormalizedRecord(ctx, raw, docs_wiki.RecordTypeDocsPage, normPayload, hashBytes(normPayload), raw.SourceCreatedAt, raw.SourceAuthorRef)
}

func (s *Service) normalizeFilesystemFileArtifact(ctx context.Context, raw *RawArtifact) error {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw.MetadataJSON, &top); err != nil {
		return fmt.Errorf("filesystem raw metadata: %w", err)
	}
	fileRaw, ok := top["filesystem_file"]
	if !ok {
		return errors.New("filesystem_file missing from raw metadata_json")
	}
	var file struct {
		Path        string `json:"path"`
		ContentText string `json:"content_text"`
	}
	if err := json.Unmarshal(fileRaw, &file); err != nil {
		return fmt.Errorf("filesystem_file payload: %w", err)
	}
	if strings.TrimSpace(file.Path) == "" {
		return errors.New("filesystem_file missing path")
	}
	norm := docs_wiki.NormalizedDocPage{
		SourceFeedID:    raw.SourceFeedID,
		ConnectorFamily: "docs_wiki",
		ConnectorType:   "filesystem",
		Title:           filepath.Base(file.Path),
		ExternalRef:     file.Path,
		LastModifiedAt:  raw.SourceCreatedAt,
		BodyText:        file.ContentText,
		DownstreamHint: map[string]any{
			"suggested_entity_type": "ReferenceDocument",
			"default_truth_mode":    "mirrored_authority",
		},
	}
	normPayload, err := json.Marshal(norm)
	if err != nil {
		return err
	}
	return s.insertNormalizedRecord(ctx, raw, docs_wiki.RecordTypeDocsPage, normPayload, hashBytes(normPayload), raw.SourceCreatedAt, raw.SourceAuthorRef)
}

func (s *Service) normalizeTelegramUpdateArtifact(ctx context.Context, raw *RawArtifact) error {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw.MetadataJSON, &top); err != nil {
		return fmt.Errorf("telegram raw metadata: %w", err)
	}
	updRaw, ok := top["telegram_update"]
	if !ok {
		return errors.New("telegram_update missing from raw metadata_json")
	}
	var u tgUpdate
	if err := json.Unmarshal(updRaw, &u); err != nil {
		return fmt.Errorf("telegram_update payload: %w", err)
	}
	if u.Message == nil {
		return errors.New("telegram update has no message")
	}

	normMsg := chat.FromTelegramUpdate(raw.SourceFeedID, u.UpdateID, u.Message.MessageID, u.Message.Date, u.Message.Text, u.Message.Chat.ID)
	normPayload, err := json.Marshal(normMsg)
	if err != nil {
		return err
	}
	nh := hashBytes(normPayload)
	return s.insertNormalizedRecord(ctx, raw, chat.RecordTypeChatMessage, normPayload, nh, raw.SourceCreatedAt, raw.SourceAuthorRef)
}

func (s *Service) normalizeSlackMessageArtifact(ctx context.Context, raw *RawArtifact) error {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw.MetadataJSON, &top); err != nil {
		return fmt.Errorf("slack raw metadata: %w", err)
	}
	msgRaw, ok := top["slack_message"]
	if !ok {
		return errors.New("slack_message missing from raw metadata_json")
	}
	var msg slackHistoryMessage
	if err := json.Unmarshal(msgRaw, &msg); err != nil {
		return fmt.Errorf("slack_message payload: %w", err)
	}
	threadTs := msg.ThreadTs
	if threadTs == "" {
		threadTs = msg.Ts
	}
	channelRef := raw.ExternalArtifactIDOrFallback()
	if idx := strings.LastIndex(channelRef, "-"); idx > 0 {
		channelRef = channelRef[len("slack-"):idx]
	}
	norm := chat.FromSlackMessage(raw.SourceFeedID, channelRef, msg.Ts, msg.User, msg.Text, threadTs)
	normPayload, err := json.Marshal(norm)
	if err != nil {
		return err
	}
	return s.insertNormalizedRecord(ctx, raw, chat.RecordTypeChatMessage, normPayload, hashBytes(normPayload), raw.SourceCreatedAt, raw.SourceAuthorRef)
}

func (s *Service) normalizeMattermostPostArtifact(ctx context.Context, raw *RawArtifact) error {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw.MetadataJSON, &top); err != nil {
		return fmt.Errorf("mattermost raw metadata: %w", err)
	}
	postRaw, ok := top["mattermost_post"]
	if !ok {
		return errors.New("mattermost_post missing from raw metadata_json")
	}
	var p struct {
		ID        string `json:"id"`
		CreateAt  int64  `json:"create_at"`
		UserID    string `json:"user_id"`
		Message   string `json:"message"`
		RootID    string `json:"root_id"`
		ChannelID string `json:"channel_id"`
	}
	if err := json.Unmarshal(postRaw, &p); err != nil {
		return fmt.Errorf("mattermost_post payload: %w", err)
	}
	if strings.TrimSpace(p.ID) == "" {
		return errors.New("mattermost_post missing id")
	}
	ch := strings.TrimSpace(p.ChannelID)
	if ch == "" {
		ext := raw.ExternalArtifactIDOrFallback()
		if strings.HasPrefix(ext, "mm-") {
			ext = strings.TrimPrefix(ext, "mm-")
		}
		if idx := strings.LastIndex(ext, ":"); idx > 0 {
			ch = ext[:idx]
		}
	}
	if ch == "" {
		return errors.New("mattermost_post missing channel id")
	}
	norm := chat.FromMattermostPost(raw.SourceFeedID, ch, p.ID, p.RootID, p.UserID, p.Message, p.CreateAt)
	normPayload, err := json.Marshal(norm)
	if err != nil {
		return err
	}
	var posted *time.Time
	if p.CreateAt > 0 {
		t := time.UnixMilli(p.CreateAt).UTC()
		posted = &t
	}
	return s.insertNormalizedRecord(ctx, raw, chat.RecordTypeChatMessage, normPayload, hashBytes(normPayload), posted, raw.SourceAuthorRef)
}

func (s *Service) normalizeGoogleDriveArtifact(ctx context.Context, raw *RawArtifact) error {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw.MetadataJSON, &top); err != nil {
		return fmt.Errorf("google_drive raw metadata: %w", err)
	}
	var meta struct {
		FileID       string   `json:"file_id"`
		Name         string   `json:"name"`
		MimeType     string   `json:"mime_type"`
		ExportMime   string   `json:"export_mime"`
		Parents      []string `json:"parents"`
		Owners       []string `json:"owners"`
		LastModifier string   `json:"last_modifier"`
		WebViewLink  string   `json:"web_view_link"`
		Truncated    bool     `json:"truncated"`
		ContentText  string   `json:"content_text"`
	}
	if err := json.Unmarshal(raw.MetadataJSON, &meta); err != nil {
		return fmt.Errorf("google_drive payload: %w", err)
	}
	if strings.TrimSpace(meta.FileID) == "" {
		return errors.New("google_drive_file missing file_id")
	}
	norm := docs_wiki.FromGoogleDriveExport(
		raw.SourceFeedID,
		meta.Name,
		meta.FileID,
		meta.ExportMime,
		meta.MimeType,
		meta.ContentText,
		timeRFC3339(raw.SourceCreatedAt),
		meta.Parents,
		meta.Owners,
		meta.LastModifier,
		meta.WebViewLink,
		meta.Truncated,
	)
	if strings.TrimSpace(norm.BodyText) == "" {
		return errors.New("google_drive_file missing content_text")
	}
	recordType := RecordTypeGoogleDriveDocument
	if meta.MimeType == "application/vnd.google-apps.document" {
		recordType = docs_wiki.RecordTypeDocsPage
	}
	normPayload, err := json.Marshal(norm)
	if err != nil {
		return err
	}
	return s.insertNormalizedRecord(ctx, raw, recordType, normPayload, hashBytes(normPayload), raw.SourceCreatedAt, raw.SourceAuthorRef)
}

func (s *Service) normalizeNotionPageArtifact(ctx context.Context, raw *RawArtifact) error {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw.MetadataJSON, &top); err != nil {
		return fmt.Errorf("notion raw metadata: %w", err)
	}
	pageRaw, ok := top["notion_page"]
	if !ok {
		return errors.New("notion_page missing from raw metadata_json")
	}
	var page struct {
		PageID string `json:"page_id"`
		Title  string `json:"title"`
		Body   string `json:"body"`
	}
	if err := json.Unmarshal(pageRaw, &page); err != nil {
		return fmt.Errorf("notion_page payload: %w", err)
	}
	if strings.TrimSpace(page.PageID) == "" {
		return errors.New("notion_page missing page_id")
	}
	modifiedAt := raw.SourceCreatedAt
	norm := docs_wiki.NormalizedDocPage{
		SourceFeedID:    raw.SourceFeedID,
		ConnectorFamily: "docs_wiki",
		ConnectorType:   "notion",
		Title:           firstNonEmptyString(page.Title, "Untitled"),
		ExternalRef:     page.PageID,
		LastModifiedAt:  modifiedAt,
		BodyText:        page.Body,
		DownstreamHint: map[string]any{
			"suggested_entity_type": "ReferenceDocument",
		},
	}
	normPayload, err := json.Marshal(norm)
	if err != nil {
		return err
	}
	return s.insertNormalizedRecord(ctx, raw, docs_wiki.RecordTypeDocsPage, normPayload, hashBytes(normPayload), modifiedAt, raw.SourceAuthorRef)
}

func (s *Service) normalizeConfluencePageArtifact(ctx context.Context, raw *RawArtifact) error {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw.MetadataJSON, &top); err != nil {
		return fmt.Errorf("confluence raw metadata: %w", err)
	}
	pageRaw, ok := top["confluence_page"]
	if !ok {
		return errors.New("confluence_page missing from raw metadata_json")
	}
	var doc confluencePageParsed
	if err := json.Unmarshal(pageRaw, &doc); err != nil {
		return fmt.Errorf("confluence_page payload: %w", err)
	}
	if strings.TrimSpace(doc.ID) == "" {
		return errors.New("confluence_page missing id")
	}
	var mod *time.Time
	if t := firstNonEmptyTime(doc.Version.When, doc.History.LastUpdated.When); t != "" {
		if parsed, err := parseConfluenceTime(t); err == nil {
			mod = &parsed
		}
	}
	parentRefs := ancestorIDs(doc.Ancestors)
	parentRef := ""
	if len(parentRefs) > 0 {
		parentRef = parentRefs[len(parentRefs)-1]
	}
	labels := make([]string, 0, len(doc.Metadata.Labels.Results))
	for _, l := range doc.Metadata.Labels.Results {
		if l.Name != "" {
			labels = append(labels, l.Name)
		}
	}
	editor := strings.TrimSpace(doc.History.LastUpdated.By.DisplayName)
	if editor == "" {
		editor = strings.TrimSpace(doc.History.LastUpdated.By.Username)
	}
	norm := docs_wiki.NormalizedDocPage{
		SourceFeedID:    raw.SourceFeedID,
		ConnectorFamily: "docs_wiki",
		ConnectorType:   "confluence",
		Title:           doc.Title,
		ExternalRef:     doc.ID,
		ExternalDocID:   doc.ID,
		ParentRef:       parentRef,
		ParentRefs:      parentRefs,
		LastModifiedAt:  mod,
		EditorRef:       editor,
		Labels:          labels,
		BodyText:        doc.Body.Storage.Value,
		WebViewLink:     doc.Links.WebUI,
		DownstreamHint: map[string]any{
			"suggested_entity_type": "ReferenceDocument",
			"default_truth_mode":    "mirrored_authority",
		},
	}
	normPayload, err := json.Marshal(norm)
	if err != nil {
		return err
	}
	return s.insertNormalizedRecord(ctx, raw, docs_wiki.RecordTypeDocsPage, normPayload, hashBytes(normPayload), mod, raw.SourceAuthorRef)
}

func (s *Service) normalizeGoogleCalendarEventArtifact(ctx context.Context, raw *RawArtifact) error {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw.MetadataJSON, &top); err != nil {
		return fmt.Errorf("google_calendar raw metadata: %w", err)
	}
	eventRaw, ok := top["google_calendar_event"]
	if !ok {
		return errors.New("google_calendar_event missing from raw metadata_json")
	}
	var event struct {
		ID       string `json:"id"`
		Summary  string `json:"summary"`
		HtmlLink string `json:"htmlLink"`
		Start    struct {
			DateTime string `json:"dateTime"`
			Date     string `json:"date"`
		} `json:"start"`
		End struct {
			DateTime string `json:"dateTime"`
			Date     string `json:"date"`
		} `json:"end"`
	}
	if err := json.Unmarshal(eventRaw, &event); err != nil {
		return fmt.Errorf("google_calendar_event payload: %w", err)
	}
	if strings.TrimSpace(event.ID) == "" {
		return errors.New("google_calendar_event missing id")
	}
	start, end := parseEventTimes(event.Start.DateTime, event.Start.Date, event.End.DateTime, event.End.Date)
	norm := meeting.NormalizedCalendarEvent{
		SourceFeedID:    raw.SourceFeedID,
		ConnectorFamily: "meeting",
		ConnectorType:   "google_calendar",
		ExternalRef:     event.ID,
		Summary:         event.Summary,
		StartAt:         start,
		EndAt:           end,
		HTMLLink:        event.HtmlLink,
	}
	if proj, topic, ok := meeting.ParseCalendarSummaryProjectTopic(event.Summary); ok {
		norm.ParsedProjectTitle = proj
		norm.ParsedMeetingTopic = topic
		norm.TitleParseOK = true
	}
	normPayload, err := json.Marshal(norm)
	if err != nil {
		return err
	}
	return s.insertNormalizedRecord(ctx, raw, meeting.RecordTypeCalendarEvent, normPayload, hashBytes(normPayload), start, raw.SourceAuthorRef)
}

func (s *Service) normalizeFirefliesTranscriptArtifact(ctx context.Context, raw *RawArtifact) error {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw.MetadataJSON, &top); err != nil {
		return fmt.Errorf("fireflies raw metadata: %w", err)
	}
	trRaw, ok := top["fireflies_transcript"]
	if !ok {
		return errors.New("fireflies_transcript missing from raw metadata_json")
	}
	var tr struct {
		ID    string `json:"id"`
		Title string `json:"title"`
		Date  string `json:"date"`
	}
	if err := json.Unmarshal(trRaw, &tr); err != nil {
		return fmt.Errorf("fireflies_transcript payload: %w", err)
	}
	if strings.TrimSpace(tr.ID) == "" {
		return errors.New("fireflies_transcript missing id")
	}
	var started *time.Time
	if tr.Date != "" {
		if t, err := time.Parse(time.RFC3339, tr.Date); err == nil {
			started = &t
		}
	}
	norm := meeting.NormalizedMeetingTranscript{
		SourceFeedID:    raw.SourceFeedID,
		ConnectorFamily: "meeting",
		ConnectorType:   "fireflies",
		ExternalRef:     tr.ID,
		Title:           tr.Title,
		StartedAt:       started,
		BodyText:        tr.Title,
	}
	normPayload, err := json.Marshal(norm)
	if err != nil {
		return err
	}
	return s.insertNormalizedRecord(ctx, raw, meeting.RecordTypeTranscript, normPayload, hashBytes(normPayload), started, raw.SourceAuthorRef)
}

// PersistNormalizedRecord is the canonical write path for normalized_records.
// All connector adapters MUST go through it (or insertNormalizedRecord which
// is just a wrapper) so the chunks-rebuild + embedding-enqueue hook fires
// uniformly. Inserting via raw `INSERT INTO normalized_records ...` from a
// new sync file would skip the hook and silently leave the record invisible
// to retrieval — surface that as a code-review reject.
//
// Returns (normID, true) on fresh insert, (uuid.Nil, false, nil) on dedup
// conflict, error otherwise. Callers that don't need the id can ignore both
// return values.
func (s *Service) PersistNormalizedRecord(
	ctx context.Context,
	rawArtifactID uuid.UUID,
	sourceFeedID uuid.UUID,
	recordType string,
	normPayload []byte,
	recordHash string,
	sourceTimestamp *time.Time,
	detectedAuthorRef *string,
) (uuid.UUID, bool, error) {
	var normID uuid.UUID
	err := s.pool.QueryRow(ctx, `
		INSERT INTO normalized_records (
			raw_artifact_id, source_feed_id, record_type, structured_payload_json, record_hash, source_timestamp, detected_author_ref, normalization_version
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,1)
		ON CONFLICT (source_feed_id, record_hash) DO NOTHING
		RETURNING id`,
		rawArtifactID, sourceFeedID, recordType, normPayload, recordHash, sourceTimestamp, detectedAuthorRef,
	).Scan(&normID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, false, nil
		}
		return uuid.Nil, false, err
	}
	s.fireOnNormalizedRecordPersisted(ctx, normID)
	return normID, true, nil
}

func (s *Service) insertNormalizedRecord(
	ctx context.Context,
	raw *RawArtifact,
	recordType string,
	normPayload []byte,
	recordHash string,
	sourceTimestamp *time.Time,
	detectedAuthorRef *string,
) error {
	_, _, err := s.PersistNormalizedRecord(ctx, raw.ID, raw.SourceFeedID, recordType, normPayload, recordHash, sourceTimestamp, detectedAuthorRef)
	return err
}

func (r *RawArtifact) ExternalArtifactIDOrFallback() string {
	if r.ExternalArtifactID != nil && strings.TrimSpace(*r.ExternalArtifactID) != "" {
		return strings.TrimSpace(*r.ExternalArtifactID)
	}
	return ""
}

func firstNonEmptyString(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

func timeRFC3339(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func parseEventTimes(startDateTime, startDate, endDateTime, endDate string) (*time.Time, *time.Time) {
	parseOne := func(dateTime, date string) *time.Time {
		if dateTime != "" {
			if t, err := time.Parse(time.RFC3339, dateTime); err == nil {
				return &t
			}
		}
		if date != "" {
			if t, err := time.Parse("2006-01-02", date); err == nil {
				return &t
			}
		}
		return nil
	}
	return parseOne(startDateTime, startDate), parseOne(endDateTime, endDate)
}
