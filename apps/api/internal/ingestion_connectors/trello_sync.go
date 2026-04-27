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

	"github.com/google/uuid"

	"github.com/knowledgelayer/api/internal/ingestion_connectors/families/work_mgmt"
)

type trelloFeedConfig struct {
	APIKey string `json:"trello_api_key"`
	Token  string `json:"trello_token"`
}

// ParseTrelloFeedConfig parses connector_config_json for Trello feeds.
func ParseTrelloFeedConfig(raw json.RawMessage) (*trelloFeedConfig, error) {
	var c trelloFeedConfig
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, err
	}
	if c.APIKey == "" || c.Token == "" {
		return nil, errors.New("trello: trello_api_key and trello_token required")
	}
	return &c, nil
}

func validateTrelloV1Activation(feed *SourceFeed, cfg *trelloFeedConfig) error {
	if feed == nil || cfg == nil {
		return errors.New("trello: missing feed or config")
	}
	if strings.TrimSpace(feed.ExternalRef) == "" {
		return errors.New("trello: external_ref required (board id)")
	}
	return nil
}

// ValidateTrelloSourceFeedForActivation is used by the Trello adapter for active feeds.
func ValidateTrelloSourceFeedForActivation(feed *SourceFeed) error {
	cfg, err := ParseTrelloFeedConfig(feed.ConnectorConfigJSON)
	if err != nil {
		return err
	}
	return validateTrelloV1Activation(feed, cfg)
}

// SyncTrello lists cards on a board (v1 cap 20).
func (s *Service) SyncTrello(ctx context.Context, feedID uuid.UUID) (*IngestionRun, error) {
	feed, err := s.GetSourceFeed(ctx, feedID)
	if err != nil {
		return nil, err
	}
	conn, err := s.GetConnector(ctx, feed.ConnectorID)
	if err != nil {
		return nil, err
	}
	if conn.Type != "trello" {
		return nil, fmt.Errorf("connector is %s, not trello", conn.Type)
	}
	if feed.Status != "active" {
		return nil, errors.New("source feed must be active")
	}
	cfg, err := ParseTrelloFeedConfig(feed.ConnectorConfigJSON)
	if err != nil {
		return nil, err
	}
	if err := validateTrelloV1Activation(feed, cfg); err != nil {
		return nil, err
	}

	runID, err := s.startIngestionRun(ctx, feedID)
	if err != nil {
		return nil, err
	}

	boardID := strings.TrimSpace(feed.ExternalRef)
	u := fmt.Sprintf("https://api.trello.com/1/boards/%s/cards?key=%s&token=%s&limit=20",
		url.PathEscape(boardID), url.QueryEscape(cfg.APIKey), url.QueryEscape(cfg.Token))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
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
		return nil, fmt.Errorf("trello: status %d: %s", resp.StatusCode, string(body))
	}

	var cards []struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Desc        string `json:"desc"`
		ShortURL    string `json:"shortUrl"`
		IDList      string `json:"idList"`
		DateLastAct string `json:"dateLastActivity"`
	}
	if err := json.Unmarshal(body, &cards); err != nil {
		s.finalizeIngestionRun(ctx, runID, "failed", 0, 0, 0, 1)
		s.completeSourceFeedSync(ctx, feedID, 1, false)
		return nil, err
	}

	ingested := 0
	deduped := 0
	warnings := 0
	errs := 0

	for _, c := range cards {
		rawPayload, _ := json.Marshal(c)
		h := hashBytes(rawPayload)
		extID := "trello-" + c.ID
		var rawObj map[string]any
		_ = json.Unmarshal(rawPayload, &rawObj)
		extra := map[string]any{"trello_card": rawObj}
		metaJSON, merr := s.buildRawArtifactMetadataJSON(ctx, *conn, *feed, work_mgmt.ArtifactKindIssue, rawPayload, extra)
		if merr != nil {
			errs++
			continue
		}
		rawID, inserted, ierr := insertRawArtifactRow(ctx, s.pool, feedID, runID, "trello_card", extID, h, "", metaJSON, nil, nil)
		if ierr != nil {
			errs++
			continue
		}
		if !inserted {
			deduped++
			continue
		}
		norm := work_mgmt.NormalizedWorkItem{
			SourceFeedID:    feedID,
			ConnectorFamily: "work_mgmt",
			ConnectorType:   "trello",
			ExternalRef:     c.ID,
			Title:           c.Name,
			BodyText:        c.Desc,
			StatusName:      c.IDList,
			URLHint:         c.ShortURL,
		}
		normPayload, _ := json.Marshal(norm)
		nh := hashBytes(normPayload)
		tag, qerr := s.pool.Exec(ctx, `
			INSERT INTO normalized_records (raw_artifact_id, source_feed_id, record_type, structured_payload_json, record_hash)
			VALUES ($1,$2,$3,$4,$5)
			ON CONFLICT (source_feed_id, record_hash) DO NOTHING`,
			rawID, feedID, work_mgmt.RecordTypeWorkItem, normPayload, nh)
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
