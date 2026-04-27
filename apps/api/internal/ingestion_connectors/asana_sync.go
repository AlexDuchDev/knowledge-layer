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

	"github.com/google/uuid"

	"github.com/knowledgelayer/api/internal/ingestion_connectors/families/work_mgmt"
)

const maxAsanaTasksPerSync = 25
const maxAsanaStoriesPerTask = 15

type asanaFeedConfig struct {
	PAT string `json:"asana_personal_access_token"`
}

// ParseAsanaFeedConfig parses connector_config_json for Asana feeds.
func ParseAsanaFeedConfig(raw json.RawMessage) (*asanaFeedConfig, error) {
	var c asanaFeedConfig
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, err
	}
	c.PAT = strings.TrimSpace(c.PAT)
	if c.PAT == "" {
		return nil, errors.New("asana: asana_personal_access_token required")
	}
	return &c, nil
}

// ValidateAsanaSourceFeedForActivation validates PAT and project gid.
func ValidateAsanaSourceFeedForActivation(feed *SourceFeed) error {
	if feed == nil {
		return errors.New("asana: nil feed")
	}
	if _, err := ParseAsanaFeedConfig(feed.ConnectorConfigJSON); err != nil {
		return err
	}
	if strings.TrimSpace(feed.ExternalRef) == "" {
		return errors.New("asana: external_ref required (project gid)")
	}
	return nil
}

// SyncAsana lists tasks in a project and ingests task + story payloads (v1).
func (s *Service) SyncAsana(ctx context.Context, feedID uuid.UUID) (*IngestionRun, error) {
	feed, err := s.GetSourceFeed(ctx, feedID)
	if err != nil {
		return nil, err
	}
	conn, err := s.GetConnector(ctx, feed.ConnectorID)
	if err != nil {
		return nil, err
	}
	if conn.Type != "asana" {
		return nil, fmt.Errorf("connector is %s, not asana", conn.Type)
	}
	if feed.Status != "active" {
		return nil, errors.New("source feed must be active")
	}
	cfg, err := ParseAsanaFeedConfig(feed.ConnectorConfigJSON)
	if err != nil {
		return nil, err
	}
	if err := ValidateAsanaSourceFeedForActivation(feed); err != nil {
		return nil, err
	}

	runID, err := s.startIngestionRun(ctx, feedID)
	if err != nil {
		return nil, err
	}

	projectGID := url.PathEscape(strings.TrimSpace(feed.ExternalRef))
	opt := url.QueryEscape("name,notes,permalink_url,completed,assignee.name,assignee.email,created_by.name,created_by.email,created_at,modified_at,due_on")
	u := fmt.Sprintf("https://app.asana.com/api/1.0/projects/%s/tasks?limit=%d&opt_fields=%s", projectGID, maxAsanaTasksPerSync, opt)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.PAT)
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
		return nil, fmt.Errorf("asana tasks: status %d: %s", resp.StatusCode, string(body))
	}

	var listParsed struct {
		Data []json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &listParsed); err != nil {
		s.finalizeIngestionRun(ctx, runID, "failed", 0, 0, 0, 1)
		s.completeSourceFeedSync(ctx, feedID, 1, false)
		return nil, err
	}

	ingested, deduped, warnings, errs := 0, 0, 0, 0
	projectRef := strings.TrimSpace(feed.ExternalRef)

	for _, rawTask := range listParsed.Data {
		var t asanaTask
		if err := json.Unmarshal(rawTask, &t); err != nil || t.GID == "" {
			errs++
			continue
		}
		taskPayload := rawTask
		h := hashBytes(taskPayload)
		extID := "asana-task-" + t.GID
		var rawObj map[string]any
		_ = json.Unmarshal(taskPayload, &rawObj)
		extra := map[string]any{"asana_task": rawObj}
		metaJSON, merr := s.buildRawArtifactMetadataJSON(ctx, *conn, *feed, work_mgmt.ArtifactAsanaTask, taskPayload, extra)
		if merr != nil {
			errs++
			continue
		}
		rawID, inserted, ierr := insertRawArtifactRow(ctx, s.pool, feedID, runID, work_mgmt.ArtifactAsanaTask, extID, h, "", metaJSON, nil, nil)
		if ierr != nil {
			errs++
			continue
		}
		if !inserted {
			deduped++
			continue
		}

		storyRefs, sErr := s.syncAsanaStories(ctx, cfg.PAT, feedID, runID, conn, feed, t.GID)
		if sErr > 0 {
			errs += sErr
		}

		assignee := strings.TrimSpace(t.Assignee.Name)
		if assignee == "" {
			assignee = strings.TrimSpace(t.Assignee.Email)
		}
		creator := strings.TrimSpace(t.CreatedBy.Name)
		if creator == "" {
			creator = strings.TrimSpace(t.CreatedBy.Email)
		}
		var createdAt, modAt *time.Time
		if t.CreatedAt != "" {
			if ts, e := time.Parse(time.RFC3339, t.CreatedAt); e == nil {
				createdAt = &ts
			}
		}
		if t.ModifiedAt != "" {
			if ts, e := time.Parse(time.RFC3339, t.ModifiedAt); e == nil {
				modAt = &ts
			}
		}
		statusName := "open"
		if t.Completed {
			statusName = "completed"
		}
		norm := work_mgmt.NormalizedWorkItem{
			SourceFeedID:    feedID,
			ConnectorFamily: "work_mgmt",
			ConnectorType:   "asana",
			ExternalRef:     t.GID,
			Title:           t.Name,
			BodyText:        t.Notes,
			StatusName:      statusName,
			AssigneeRef:     assignee,
			CreatorRef:      creator,
			ProjectRef:      projectRef,
			CommentRefs:     storyRefs,
			CreatedAt:       createdAt,
			UpdatedAt:       modAt,
			URLHint:         t.PermalinkURL,
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

type asanaTask struct {
	GID          string `json:"gid"`
	Name         string `json:"name"`
	Notes        string `json:"notes"`
	Completed    bool   `json:"completed"`
	PermalinkURL string `json:"permalink_url"`
	CreatedAt    string `json:"created_at"`
	ModifiedAt   string `json:"modified_at"`
	Assignee     struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"assignee"`
	CreatedBy struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"created_by"`
}

func (s *Service) syncAsanaStories(ctx context.Context, pat string, feedID, runID uuid.UUID, conn *Connector, feed *SourceFeed, taskGID string) (storyRefs []string, errs int) {
	opt := url.QueryEscape("text,type,created_at,created_by.name")
	u := fmt.Sprintf("https://app.asana.com/api/1.0/tasks/%s/stories?limit=%d&opt_fields=%s", url.PathEscape(taskGID), maxAsanaStoriesPerTask, opt)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, 1
	}
	req.Header.Set("Authorization", "Bearer "+pat)
	resp, err := s.HTTP.Do(req)
	if err != nil {
		return nil, 1
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil || resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, 1
	}
	var parsed struct {
		Data []json.RawMessage `json:"data"`
	}
	if json.Unmarshal(body, &parsed) != nil {
		return nil, 1
	}
	out := make([]string, 0, len(parsed.Data))
	for _, raw := range parsed.Data {
		var st struct {
			GID string `json:"gid"`
		}
		if json.Unmarshal(raw, &st) != nil || st.GID == "" {
			continue
		}
		h := hashBytes(raw)
		extID := "asana-story-" + st.GID
		var rawObj map[string]any
		_ = json.Unmarshal(raw, &rawObj)
		extra := map[string]any{"asana_story": rawObj}
		metaJSON, merr := s.buildRawArtifactMetadataJSON(ctx, *conn, *feed, work_mgmt.ArtifactAsanaStory, raw, extra)
		if merr != nil {
			errs++
			continue
		}
		_, _, _ = insertRawArtifactRow(ctx, s.pool, feedID, runID, work_mgmt.ArtifactAsanaStory, extID, h, "", metaJSON, nil, nil)
		out = append(out, st.GID)
	}
	return out, errs
}
