package ingestion_connectors

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// TrelloBoardRef is a board the token can access (onboarding picker).
type TrelloBoardRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
}

// ListTrelloBoards calls Trello REST for boards visible to the member (API key + token).
func (s *Service) ListTrelloBoards(ctx context.Context, cfg *trelloFeedConfig) ([]TrelloBoardRef, error) {
	if cfg == nil {
		return nil, fmt.Errorf("trello: nil config")
	}
	u := fmt.Sprintf("https://api.trello.com/1/members/me/boards?fields=id,name,url&key=%s&token=%s",
		url.QueryEscape(cfg.APIKey), url.QueryEscape(cfg.Token))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
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
		return nil, fmt.Errorf("trello: status %d: %s", resp.StatusCode, string(body))
	}
	var boards []TrelloBoardRef
	if err := json.Unmarshal(body, &boards); err != nil {
		return nil, fmt.Errorf("trello: decode boards: %w", err)
	}
	return boards, nil
}
