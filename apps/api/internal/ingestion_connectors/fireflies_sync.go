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

	"github.com/knowledgelayer/api/internal/ingestion_connectors/families/meeting"
)

type firefliesFeedConfig struct {
	APIKey string `json:"fireflies_api_key"`
}

// ValidateFirefliesSourceFeedForActivation checks API key for active feeds.
func ValidateFirefliesSourceFeedForActivation(feed *SourceFeed) error {
	if feed == nil {
		return errors.New("fireflies: nil feed")
	}
	_, err := parseFirefliesFeedConfig(feed.ConnectorConfigJSON)
	return err
}

func parseFirefliesFeedConfig(raw json.RawMessage) (*firefliesFeedConfig, error) {
	var c firefliesFeedConfig
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, err
	}
	if strings.TrimSpace(c.APIKey) == "" {
		return nil, errors.New("fireflies: fireflies_api_key required")
	}
	return &c, nil
}

// SyncFireflies lists recent transcripts via GraphQL (v1; API surface may evolve).
func (s *Service) SyncFireflies(ctx context.Context, feedID uuid.UUID) (*IngestionRun, error) {
	feed, err := s.GetSourceFeed(ctx, feedID)
	if err != nil {
		return nil, err
	}
	conn, err := s.GetConnector(ctx, feed.ConnectorID)
	if err != nil {
		return nil, err
	}
	if conn.Type != "fireflies" {
		return nil, fmt.Errorf("connector is %s, not fireflies", conn.Type)
	}
	if feed.Status != "active" {
		return nil, errors.New("source feed must be active")
	}
	cfg, err := parseFirefliesFeedConfig(feed.ConnectorConfigJSON)
	if err != nil {
		return nil, err
	}

	runID, err := s.startIngestionRun(ctx, feedID)
	if err != nil {
		return nil, err
	}

	q := `{"query":"{ transcripts { id title date } }"}`
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.fireflies.ai/graphql", bytes.NewReader([]byte(q)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")

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
		return nil, fmt.Errorf("fireflies: status %d: %s", resp.StatusCode, string(body))
	}

	var gql struct {
		Data struct {
			Transcripts []struct {
				ID    string `json:"id"`
				Title string `json:"title"`
				Date  string `json:"date"`
			} `json:"transcripts"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(body, &gql); err != nil {
		s.finalizeIngestionRun(ctx, runID, "failed", 0, 0, 0, 1)
		s.completeSourceFeedSync(ctx, feedID, 1, false)
		return nil, err
	}
	if len(gql.Errors) > 0 {
		s.finalizeIngestionRun(ctx, runID, "failed", 0, 0, 0, 1)
		s.completeSourceFeedSync(ctx, feedID, 1, false)
		return nil, fmt.Errorf("fireflies graphql: %s", gql.Errors[0].Message)
	}

	ingested := 0
	deduped := 0
	warnings := 0
	errs := 0

	const maxT = 15
	for i, tr := range gql.Data.Transcripts {
		if i >= maxT {
			warnings++
			break
		}
		rawPayload, _ := json.Marshal(tr)
		h := hashBytes(rawPayload)
		extID := "ff-" + tr.ID
		var rawObj map[string]any
		_ = json.Unmarshal(rawPayload, &rawObj)
		extra := map[string]any{"fireflies_transcript": rawObj}
		metaJSON, merr := s.buildRawArtifactMetadataJSON(ctx, *conn, *feed, meeting.ArtifactKindTranscript, rawPayload, extra)
		if merr != nil {
			errs++
			continue
		}
		rawID, inserted, ierr := insertRawArtifactRow(ctx, s.pool, feedID, runID, "fireflies_transcript", extID, h, "", metaJSON, nil, nil)
		if ierr != nil {
			errs++
			continue
		}
		if !inserted {
			deduped++
			continue
		}
		var started *time.Time
		if tr.Date != "" {
			if t, err := time.Parse(time.RFC3339, tr.Date); err == nil {
				started = &t
			}
		}
		norm := meeting.NormalizedMeetingTranscript{
			SourceFeedID:    feedID,
			ConnectorFamily: "meeting",
			ConnectorType:   "fireflies",
			ExternalRef:     tr.ID,
			Title:           tr.Title,
			StartedAt:       started,
			BodyText:        tr.Title,
		}
		normPayload, _ := json.Marshal(norm)
		nh := hashBytes(normPayload)
		tag, qerr := s.pool.Exec(ctx, `
			INSERT INTO normalized_records (raw_artifact_id, source_feed_id, record_type, structured_payload_json, record_hash)
			VALUES ($1,$2,$3,$4,$5)
			ON CONFLICT (source_feed_id, record_hash) DO NOTHING`,
			rawID, feedID, meeting.RecordTypeTranscript, normPayload, nh)
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
