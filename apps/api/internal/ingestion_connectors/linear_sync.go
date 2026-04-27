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

	"github.com/knowledgelayer/api/internal/ingestion_connectors/families/work_mgmt"
)

const maxLinearIssuesPerSync = 25
const maxLinearCommentsPerIssue = 10

type linearFeedConfig struct {
	APIKey string `json:"linear_api_key"`
}

// ParseLinearFeedConfig parses connector_config_json for Linear feeds.
func ParseLinearFeedConfig(raw json.RawMessage) (*linearFeedConfig, error) {
	var c linearFeedConfig
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, err
	}
	c.APIKey = strings.TrimSpace(c.APIKey)
	if c.APIKey == "" {
		return nil, errors.New("linear: linear_api_key required")
	}
	return &c, nil
}

// ValidateLinearSourceFeedForActivation validates API key and team id.
func ValidateLinearSourceFeedForActivation(feed *SourceFeed) error {
	if feed == nil {
		return errors.New("linear: nil feed")
	}
	if _, err := ParseLinearFeedConfig(feed.ConnectorConfigJSON); err != nil {
		return err
	}
	if strings.TrimSpace(feed.ExternalRef) == "" {
		return errors.New("linear: external_ref required (team id)")
	}
	return nil
}

const linearIssuesQuery = `query($team: String!, $first: Int!, $comments: Int!) {
  issues(filter: { team: { id: { eq: $team } } }, first: $first) {
    nodes {
      id
      identifier
      title
      description
      url
      createdAt
      updatedAt
      state { name }
      assignee { email name }
      team { id key name }
      labels { nodes { name } }
      comments(first: $comments) { nodes { id body createdAt } }
    }
  }
}`

// SyncLinear runs a bounded GraphQL fetch for team issues (v1).
func (s *Service) SyncLinear(ctx context.Context, feedID uuid.UUID) (*IngestionRun, error) {
	feed, err := s.GetSourceFeed(ctx, feedID)
	if err != nil {
		return nil, err
	}
	conn, err := s.GetConnector(ctx, feed.ConnectorID)
	if err != nil {
		return nil, err
	}
	if conn.Type != "linear" {
		return nil, fmt.Errorf("connector is %s, not linear", conn.Type)
	}
	if feed.Status != "active" {
		return nil, errors.New("source feed must be active")
	}
	cfg, err := ParseLinearFeedConfig(feed.ConnectorConfigJSON)
	if err != nil {
		return nil, err
	}
	if err := ValidateLinearSourceFeedForActivation(feed); err != nil {
		return nil, err
	}

	runID, err := s.startIngestionRun(ctx, feedID)
	if err != nil {
		return nil, err
	}

	payload := map[string]any{
		"query": linearIssuesQuery,
		"variables": map[string]any{
			"team":     strings.TrimSpace(feed.ExternalRef),
			"first":    maxLinearIssuesPerSync,
			"comments": maxLinearCommentsPerIssue,
		},
	}
	bodyBytes, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.linear.app/graphql", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", cfg.APIKey)
	resp, err := s.HTTP.Do(req)
	if err != nil {
		s.finalizeIngestionRun(ctx, runID, "failed", 0, 0, 0, 1)
		s.completeSourceFeedSync(ctx, feedID, 1, false)
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		s.finalizeIngestionRun(ctx, runID, "failed", 0, 0, 0, 1)
		s.completeSourceFeedSync(ctx, feedID, 1, false)
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		s.finalizeIngestionRun(ctx, runID, "failed", 0, 0, 0, 1)
		s.completeSourceFeedSync(ctx, feedID, 1, false)
		return nil, fmt.Errorf("linear graphql: status %d: %s", resp.StatusCode, string(respBody))
	}

	var gqlResp struct {
		Data *struct {
			Issues struct {
				Nodes []linearIssueNode `json:"nodes"`
			} `json:"issues"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(respBody, &gqlResp); err != nil {
		s.finalizeIngestionRun(ctx, runID, "failed", 0, 0, 0, 1)
		s.completeSourceFeedSync(ctx, feedID, 1, false)
		return nil, err
	}
	if len(gqlResp.Errors) > 0 {
		s.finalizeIngestionRun(ctx, runID, "failed", 0, 0, 0, 1)
		s.completeSourceFeedSync(ctx, feedID, 1, false)
		return nil, fmt.Errorf("linear graphql: %s", gqlResp.Errors[0].Message)
	}
	if gqlResp.Data == nil {
		s.finalizeIngestionRun(ctx, runID, "failed", 0, 0, 0, 1)
		s.completeSourceFeedSync(ctx, feedID, 1, false)
		return nil, errors.New("linear: empty data")
	}

	ingested, deduped, warnings, errs := 0, 0, 0, 0
	for _, iss := range gqlResp.Data.Issues.Nodes {
		rawPayload, _ := json.Marshal(iss)
		h := hashBytes(rawPayload)
		extID := "linear-issue-" + iss.ID
		var rawObj map[string]any
		_ = json.Unmarshal(rawPayload, &rawObj)
		extra := map[string]any{"linear_issue": rawObj}
		metaJSON, merr := s.buildRawArtifactMetadataJSON(ctx, *conn, *feed, work_mgmt.ArtifactLinearIssue, rawPayload, extra)
		if merr != nil {
			errs++
			continue
		}
		rawID, inserted, ierr := insertRawArtifactRow(ctx, s.pool, feedID, runID, work_mgmt.ArtifactLinearIssue, extID, h, "", metaJSON, nil, nil)
		if ierr != nil {
			errs++
			continue
		}
		if !inserted {
			deduped++
			continue
		}

		commentRefs := make([]string, 0, len(iss.Comments.Nodes))
		for _, c := range iss.Comments.Nodes {
			cPayload, _ := json.Marshal(c)
			ch := hashBytes(cPayload)
			cExt := "linear-comment-" + c.ID
			var cObj map[string]any
			_ = json.Unmarshal(cPayload, &cObj)
			cExtra := map[string]any{"linear_comment": cObj}
			cMeta, ce := s.buildRawArtifactMetadataJSON(ctx, *conn, *feed, work_mgmt.ArtifactLinearComment, cPayload, cExtra)
			if ce == nil {
				_, _, _ = insertRawArtifactRow(ctx, s.pool, feedID, runID, work_mgmt.ArtifactLinearComment, cExt, ch, "", cMeta, nil, nil)
			}
			commentRefs = append(commentRefs, c.ID)
		}

		labels := make([]string, 0, len(iss.Labels.Nodes))
		for _, l := range iss.Labels.Nodes {
			if l.Name != "" {
				labels = append(labels, l.Name)
			}
		}
		assignee := strings.TrimSpace(iss.Assignee.Email)
		if assignee == "" {
			assignee = strings.TrimSpace(iss.Assignee.Name)
		}
		var createdAt, updatedAt *time.Time
		if iss.CreatedAt != "" {
			if t, e := time.Parse(time.RFC3339, iss.CreatedAt); e == nil {
				createdAt = &t
			}
		}
		if iss.UpdatedAt != "" {
			if t, e := time.Parse(time.RFC3339, iss.UpdatedAt); e == nil {
				updatedAt = &t
			}
		}
		norm := work_mgmt.NormalizedWorkItem{
			SourceFeedID:    feedID,
			ConnectorFamily: "work_mgmt",
			ConnectorType:   "linear",
			ExternalRef:     iss.ID,
			Title:           iss.Title + " (" + iss.Identifier + ")",
			BodyText:        iss.Description,
			StatusName:      iss.State.Name,
			AssigneeRef:     assignee,
			TeamRef:         iss.Team.ID,
			ProjectRef:      iss.Team.Key,
			Labels:          labels,
			CommentRefs:     commentRefs,
			CreatedAt:       createdAt,
			UpdatedAt:       updatedAt,
			URLHint:         iss.URL,
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

type linearIssueNode struct {
	ID          string `json:"id"`
	Identifier  string `json:"identifier"`
	Title       string `json:"title"`
	Description string `json:"description"`
	URL         string `json:"url"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
	State       struct {
		Name string `json:"name"`
	} `json:"state"`
	Assignee struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	} `json:"assignee"`
	Team struct {
		ID   string `json:"id"`
		Key  string `json:"key"`
		Name string `json:"name"`
	} `json:"team"`
	Labels struct {
		Nodes []struct {
			Name string `json:"name"`
		} `json:"nodes"`
	} `json:"labels"`
	Comments struct {
		Nodes []struct {
			ID        string `json:"id"`
			Body      string `json:"body"`
			CreatedAt string `json:"createdAt"`
		} `json:"nodes"`
	} `json:"comments"`
}
