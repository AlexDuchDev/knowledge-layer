package ingestion_connectors

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ConfluenceSpaceRef is a space key + name for onboarding pickers (feed_kind=space).
type ConfluenceSpaceRef struct {
	Key  string `json:"key"`
	Name string `json:"name"`
	Type string `json:"type,omitempty"`
}

// ListConfluenceSpaces lists spaces using Confluence Cloud REST (Bearer token).
func (s *Service) ListConfluenceSpaces(ctx context.Context, baseURL, auth string) ([]ConfluenceSpaceRef, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	auth = strings.TrimSpace(auth)
	if baseURL == "" || auth == "" {
		return nil, fmt.Errorf("confluence: base_url and auth required")
	}
	u := baseURL + "/rest/api/space?limit=50&type=global,knowledge_base,personal"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+auth)
	req.Header.Set("Accept", "application/json")
	resp, err := s.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("confluence: status %d: %s", resp.StatusCode, string(body))
	}
	var parsed struct {
		Results []struct {
			Key  string `json:"key"`
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("confluence: decode: %w", err)
	}
	out := make([]ConfluenceSpaceRef, 0, len(parsed.Results))
	for _, r := range parsed.Results {
		if r.Key == "" {
			continue
		}
		out = append(out, ConfluenceSpaceRef{Key: r.Key, Name: r.Name, Type: r.Type})
	}
	return out, nil
}
