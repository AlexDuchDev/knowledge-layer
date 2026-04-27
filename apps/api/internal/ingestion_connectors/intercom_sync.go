package ingestion_connectors

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/knowledgelayer/api/internal/ingestion_connectors/families/crm_support"
)

type intercomFeedConfig struct {
	AccessToken string `json:"intercom_access_token"`
}

// ValidateIntercomSourceFeedForActivation checks access token for active feeds.
func ValidateIntercomSourceFeedForActivation(feed *SourceFeed) error {
	if feed == nil {
		return errors.New("intercom: nil feed")
	}
	_, err := parseIntercomFeedConfig(feed.ConnectorConfigJSON)
	return err
}

func parseIntercomFeedConfig(raw json.RawMessage) (*intercomFeedConfig, error) {
	var c intercomFeedConfig
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, err
	}
	if strings.TrimSpace(c.AccessToken) == "" {
		return nil, errors.New("intercom: intercom_access_token required in connector_config_json")
	}
	return &c, nil
}

// SyncIntercom lists recent conversations (v1 cap 10).
func (s *Service) SyncIntercom(ctx context.Context, feedID uuid.UUID) (*IngestionRun, error) {
	feed, err := s.GetSourceFeed(ctx, feedID)
	if err != nil {
		return nil, err
	}
	conn, err := s.GetConnector(ctx, feed.ConnectorID)
	if err != nil {
		return nil, err
	}
	if conn.Type != "intercom" {
		return nil, fmt.Errorf("connector is %s, not intercom", conn.Type)
	}
	if feed.Status != "active" {
		return nil, errors.New("source feed must be active")
	}
	cfg, err := parseIntercomFeedConfig(feed.ConnectorConfigJSON)
	if err != nil {
		return nil, err
	}

	runID, err := s.startIngestionRun(ctx, feedID)
	if err != nil {
		return nil, err
	}

	u := "https://api.intercom.io/conversations?per_page=10"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.AccessToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Intercom-Version", "2.10")

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
		return nil, fmt.Errorf("intercom: status %d: %s", resp.StatusCode, string(body))
	}

	var parsed struct {
		Conversations []struct {
			ID                  string `json:"id"`
			Title               string `json:"title"`
			State               string `json:"state"`
			UpdatedAt           int64  `json:"updated_at"`
			ConversationMessage struct {
				Body string `json:"body"`
			} `json:"conversation_message"`
		} `json:"conversations"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		s.finalizeIngestionRun(ctx, runID, "failed", 0, 0, 0, 1)
		s.completeSourceFeedSync(ctx, feedID, 1, false)
		return nil, err
	}

	ingested, deduped, warnings, errs := 0, 0, 0, 0
	for _, c := range parsed.Conversations {
		rawPayload, _ := json.Marshal(c)
		h := hashBytes(rawPayload)
		extID := "intercom-" + c.ID
		var rawObj map[string]any
		_ = json.Unmarshal(rawPayload, &rawObj)
		extra := map[string]any{"intercom_conversation": rawObj}
		metaJSON, merr := s.buildRawArtifactMetadataJSON(ctx, *conn, *feed, crm_support.ArtifactKindIntercomConversation, rawPayload, extra)
		if merr != nil {
			errs++
			continue
		}
		rawID, inserted, ierr := insertRawArtifactRow(ctx, s.pool, feedID, runID, "intercom_conversation", extID, h, "", metaJSON, nil, nil)
		if ierr != nil {
			errs++
			continue
		}
		if !inserted {
			deduped++
			continue
		}
		var updated *time.Time
		if c.UpdatedAt > 0 {
			t := time.Unix(c.UpdatedAt, 0).UTC()
			updated = &t
		}
		norm := crm_support.NormalizedSupportConversation{
			SourceFeedID:    feedID,
			ConnectorFamily: "crm_support",
			ConnectorType:   "intercom",
			ExternalRef:     c.ID,
			Title:           c.Title,
			State:           c.State,
			UpdatedAt:       updated,
			BodyPreview:     stripHTMLPreview(c.ConversationMessage.Body, 500),
		}
		normPayload, _ := json.Marshal(norm)
		nh := hashBytes(normPayload)
		tag, qerr := s.pool.Exec(ctx, `
			INSERT INTO normalized_records (raw_artifact_id, source_feed_id, record_type, structured_payload_json, record_hash)
			VALUES ($1,$2,$3,$4,$5)
			ON CONFLICT (source_feed_id, record_hash) DO NOTHING`,
			rawID, feedID, crm_support.RecordTypeSupportConversation, normPayload, nh)
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

func stripHTMLPreview(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
