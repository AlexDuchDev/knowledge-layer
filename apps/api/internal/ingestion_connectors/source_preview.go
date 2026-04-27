package ingestion_connectors

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

const previewSampleLimit = 5

// SourceFeedPreview is a bounded, read-only sample for admins before activation. No ingestion rows are written.
type SourceFeedPreview struct {
	ConnectorType string            `json:"connector_type"`
	SourceFeedID  uuid.UUID         `json:"source_feed_id"`
	Status        string            `json:"source_feed_status"`
	Summary       string            `json:"summary"`
	Samples       []PreviewSample   `json:"samples"`
	MetadataHints map[string]string `json:"metadata_hints,omitempty"`
}

type PreviewSample struct {
	Title       string `json:"title,omitempty"`
	Subtitle    string `json:"subtitle,omitempty"`
	MimeOrType  string `json:"mime_or_type,omitempty"`
	PreviewText string `json:"preview_text,omitempty"`
	ExternalRef string `json:"external_ref,omitempty"`
}

// PreviewSourceFeed fetches a small external sample without storing artifacts or changing feed status.
func (s *Service) PreviewSourceFeed(ctx context.Context, feed *SourceFeed, conn *Connector) (*SourceFeedPreview, error) {
	if feed.Status != "draft" {
		return nil, errors.New("preview only allowed while source feed is in draft (not activated)")
	}
	out := &SourceFeedPreview{
		ConnectorType: conn.Type,
		SourceFeedID:  feed.ID,
		Status:        feed.Status,
		Samples:       nil,
		MetadataHints: map[string]string{
			"note": "Preview only — no ingestion, no entities, feed stays draft until you activate.",
		},
	}
	switch conn.Type {
	case "telegram":
		return previewTelegram(s, feed, out)
	case "google_drive":
		return previewGoogleDrive(ctx, s, feed, out)
	default:
		return nil, fmt.Errorf("preview not implemented for connector type %q", conn.Type)
	}
}

func previewTelegram(s *Service, feed *SourceFeed, out *SourceFeedPreview) (*SourceFeedPreview, error) {
	var cfg struct {
		BotToken string `json:"bot_token"`
	}
	if err := json.Unmarshal(feed.ConnectorConfigJSON, &cfg); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	if cfg.BotToken == "" {
		return nil, errors.New("connector_config_json.bot_token required for telegram preview")
	}
	updates, warns, errs := s.fetchTelegramUpdates(cfg.BotToken, 0)
	if errs > 0 && len(updates) == 0 {
		return nil, errors.New("could not reach Telegram API for preview")
	}
	out.Summary = fmt.Sprintf("telegram connector; fetched update batch (%d items, %d warnings)", len(updates), warns)
	n := 0
	for _, u := range updates {
		if n >= previewSampleLimit {
			break
		}
		msg := extractMessage(u)
		var title, sub, prev string
		if tid, ok := msg["chat_id"].(int64); ok {
			title = fmt.Sprintf("chat %d", tid)
		} else {
			title = "telegram message"
		}
		if mid, ok := msg["message_id"].(int); ok {
			sub = fmt.Sprintf("message_id=%d", mid)
		}
		if t, ok := msg["text"].(string); ok {
			prev = strings.TrimSpace(t)
			if len(prev) > 280 {
				prev = prev[:280] + "…"
			}
		}
		out.Samples = append(out.Samples, PreviewSample{
			Title:       title,
			Subtitle:    sub,
			MimeOrType:  "chat_message",
			PreviewText: prev,
			ExternalRef: fmt.Sprintf("update:%v", msg["update_id"]),
		})
		n++
	}
	if len(out.Samples) == 0 {
		out.Summary += "; no messages in current getUpdates window (bot may need user messages or longer timeout)"
	}
	return out, nil
}

func previewGoogleDrive(ctx context.Context, s *Service, feed *SourceFeed, out *SourceFeedPreview) (*SourceFeedPreview, error) {
	cfg, err := parseGoogleDriveConfig(feed.ConnectorConfigJSON)
	if err != nil {
		return nil, err
	}
	svc, err := newDriveService(ctx, cfg.ServiceAccount)
	if err != nil {
		return nil, err
	}
	q := fmt.Sprintf("'%s' in parents and trashed = false", cfg.FolderID)
	list, err := svc.Files.List().Q(q).PageSize(previewSampleLimit).
		Fields("files(id,name,mimeType,modifiedTime,owners)").
		Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("drive list preview: %w", err)
	}
	out.Summary = fmt.Sprintf("google_drive folder preview; up to %d file metadata rows", previewSampleLimit)
	for _, f := range list.Files {
		if f == nil || f.MimeType == "application/vnd.google-apps.folder" {
			continue
		}
		sub := f.ModifiedTime
		if f.MimeType == "application/vnd.google-apps.document" {
			sub += " · Google Doc (export on sync)"
		}
		var prev string
		if strings.HasPrefix(f.MimeType, "text/") || f.MimeType == "application/vnd.google-apps.document" {
			b, _, _, skip, ferr := fetchDriveFileBody(ctx, svc, f)
			if ferr == nil && !skip && len(b) > 0 {
				prev = string(b)
				if len(prev) > 400 {
					prev = prev[:400] + "…"
				}
			}
		}
		out.Samples = append(out.Samples, PreviewSample{
			Title:       f.Name,
			Subtitle:    sub,
			MimeOrType:  f.MimeType,
			PreviewText: prev,
			ExternalRef: f.Id,
		})
		if len(out.Samples) >= previewSampleLimit {
			break
		}
	}
	if len(out.Samples) == 0 {
		out.Summary += "; no ingestible files in first page (empty folder or only subfolders)"
	}
	return out, nil
}
