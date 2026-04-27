package ingestion_connectors

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// JiraProjectRef is a discoverable Jira Cloud project for onboarding pickers.
type JiraProjectRef struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
}

// ListJiraProjects returns projects visible to the given API token (Jira Cloud REST v3 project search).
func (s *Service) ListJiraProjects(ctx context.Context, cfg *jiraFeedConfig) ([]JiraProjectRef, error) {
	if cfg == nil {
		return nil, fmt.Errorf("jira: nil config")
	}
	u := fmt.Sprintf("%s/rest/api/3/project/search?orderBy=name&maxResults=50", cfg.SiteBaseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	basic := base64.StdEncoding.EncodeToString([]byte(cfg.Email + ":" + cfg.APIToken))
	req.Header.Set("Authorization", "Basic "+basic)
	req.Header.Set("Accept", "application/json")
	client := s.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("jira project search: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var parsed struct {
		Values []struct {
			ID   string `json:"id"`
			Key  string `json:"key"`
			Name string `json:"name"`
		} `json:"values"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("jira project search json: %w", err)
	}
	out := make([]JiraProjectRef, 0, len(parsed.Values))
	for _, v := range parsed.Values {
		if strings.TrimSpace(v.Key) == "" {
			continue
		}
		out = append(out, JiraProjectRef{ID: v.ID, Key: v.Key, Name: v.Name})
	}
	return out, nil
}

func jiraSearchMaxResults(cfg *jiraFeedConfig) int {
	if cfg == nil || cfg.MaxResults <= 0 {
		return 50
	}
	if cfg.MaxResults > 100 {
		return 100
	}
	return cfg.MaxResults
}
