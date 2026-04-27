package ingestion_connectors

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// LinearTeamRef is a Linear team for onboarding (external_ref = team id).
type LinearTeamRef struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
}

const linearTeamsQuery = `query { teams { nodes { id key name } } }`

// ListLinearTeams returns teams visible to the API key.
func (s *Service) ListLinearTeams(ctx context.Context, apiKey string) ([]LinearTeamRef, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, fmt.Errorf("linear: api key required")
	}
	payload := map[string]any{"query": linearTeamsQuery}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.linear.app/graphql", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", apiKey)
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
		return nil, fmt.Errorf("linear: status %d: %s", resp.StatusCode, string(body))
	}
	var env struct {
		Data struct {
			Teams struct {
				Nodes []LinearTeamRef `json:"nodes"`
			} `json:"teams"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("linear: decode: %w", err)
	}
	if len(env.Errors) > 0 {
		return nil, fmt.Errorf("linear: %s", env.Errors[0].Message)
	}
	return env.Data.Teams.Nodes, nil
}
