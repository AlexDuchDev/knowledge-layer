package ingestion_connectors

import (
	"context"
	"encoding/base64"
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

type jiraFeedConfig struct {
	SiteBaseURL string `json:"jira_site_base_url"` // https://your.atlassian.net
	Email       string `json:"jira_email"`
	APIToken    string `json:"jira_api_token"`
	// MaxResults optional; default 50, max 100 for issue search during sync.
	MaxResults int `json:"jira_max_results"`
}

// ParseJiraFeedConfig parses connector_config_json for Jira Cloud (site + email API token).
func ParseJiraFeedConfig(raw json.RawMessage) (*jiraFeedConfig, error) {
	var c jiraFeedConfig
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, err
	}
	c.SiteBaseURL = strings.TrimRight(strings.TrimSpace(c.SiteBaseURL), "/")
	if c.SiteBaseURL == "" {
		return nil, errors.New("jira: jira_site_base_url required")
	}
	if c.Email == "" || c.APIToken == "" {
		return nil, errors.New("jira: jira_email and jira_api_token required")
	}
	return &c, nil
}

func validateJiraV1Activation(feed *SourceFeed, cfg *jiraFeedConfig) error {
	if feed == nil || cfg == nil {
		return errors.New("jira: missing feed or config")
	}
	if strings.TrimSpace(feed.ExternalRef) == "" {
		return errors.New("jira: external_ref required (project key)")
	}
	return nil
}

// ValidateJiraSourceFeedForActivation is used by the Jira adapter for active feeds.
func ValidateJiraSourceFeedForActivation(feed *SourceFeed) error {
	cfg, err := ParseJiraFeedConfig(feed.ConnectorConfigJSON)
	if err != nil {
		return err
	}
	return validateJiraV1Activation(feed, cfg)
}

// SyncJira runs a shallow Jira Cloud search for issues in the configured project (v1).
func (s *Service) SyncJira(ctx context.Context, feedID uuid.UUID) (*IngestionRun, error) {
	feed, err := s.GetSourceFeed(ctx, feedID)
	if err != nil {
		return nil, err
	}
	conn, err := s.GetConnector(ctx, feed.ConnectorID)
	if err != nil {
		return nil, err
	}
	if conn.Type != "jira" {
		return nil, fmt.Errorf("connector is %s, not jira", conn.Type)
	}
	if feed.Status != "active" {
		return nil, errors.New("source feed must be active")
	}
	cfg, err := ParseJiraFeedConfig(feed.ConnectorConfigJSON)
	if err != nil {
		return nil, err
	}
	if err := validateJiraV1Activation(feed, cfg); err != nil {
		return nil, err
	}

	runID, err := s.startIngestionRun(ctx, feedID)
	if err != nil {
		return nil, err
	}

	jql := fmt.Sprintf("project=%s ORDER BY updated DESC", strings.TrimSpace(feed.ExternalRef))
	maxR := jiraSearchMaxResults(cfg)
	u := fmt.Sprintf("%s/rest/api/3/search?jql=%s&maxResults=%d", cfg.SiteBaseURL, url.QueryEscape(jql), maxR)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	basic := base64.StdEncoding.EncodeToString([]byte(cfg.Email + ":" + cfg.APIToken))
	req.Header.Set("Authorization", "Basic "+basic)
	req.Header.Set("Accept", "application/json")

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
		return nil, fmt.Errorf("jira search: status %d: %s", resp.StatusCode, string(body))
	}

	var parsed struct {
		Issues []struct {
			Key    string `json:"key"`
			Fields struct {
				Summary string `json:"summary"`
				Status  struct {
					Name string `json:"name"`
				} `json:"status"`
				Description any `json:"description"`
			} `json:"fields"`
		} `json:"issues"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		s.finalizeIngestionRun(ctx, runID, "failed", 0, 0, 0, 1)
		s.completeSourceFeedSync(ctx, feedID, 1, false)
		return nil, err
	}

	ingested := 0
	deduped := 0
	warnings := 0
	errs := 0

	for _, iss := range parsed.Issues {
		desc := jiraDescriptionPlain(iss.Fields.Description)
		rawPayload, _ := json.Marshal(iss)
		h := hashBytes(rawPayload)
		extID := "jira-" + iss.Key
		var rawObj map[string]any
		_ = json.Unmarshal(rawPayload, &rawObj)
		extra := map[string]any{"jira_issue": rawObj}
		metaJSON, merr := s.buildRawArtifactMetadataJSON(ctx, *conn, *feed, work_mgmt.ArtifactKindIssue, rawPayload, extra)
		if merr != nil {
			errs++
			continue
		}
		rawID, inserted, ierr := insertRawArtifactRow(ctx, s.pool, feedID, runID, "jira_issue", extID, h, "", metaJSON, nil, nil)
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
			ConnectorType:   "jira",
			ExternalRef:     iss.Key,
			Title:           iss.Fields.Summary,
			BodyText:        desc,
			StatusName:      iss.Fields.Status.Name,
			URLHint:         cfg.SiteBaseURL + "/browse/" + iss.Key,
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

func jiraDescriptionPlain(v any) string {
	if v == nil {
		return ""
	}
	// ADF document: try to walk content text
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	var root map[string]any
	if err := json.Unmarshal(b, &root); err != nil {
		return ""
	}
	var out strings.Builder
	jiraWalkADF(root, &out)
	return strings.TrimSpace(out.String())
}

func jiraWalkADF(node map[string]any, b *strings.Builder) {
	if node == nil {
		return
	}
	if t, ok := node["type"].(string); ok && t == "text" {
		if txt, ok := node["text"].(string); ok {
			b.WriteString(txt)
		}
	}
	if ch, ok := node["content"].([]any); ok {
		for _, c := range ch {
			if m, ok := c.(map[string]any); ok {
				jiraWalkADF(m, b)
			}
		}
	}
}
