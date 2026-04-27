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

// ZendeskViewRef is a saved view (external_ref when zendesk_feed_kind=view).
type ZendeskViewRef struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
}

// ListZendeskViews returns active views (for view-mode feeds).
func (s *Service) ListZendeskViews(ctx context.Context, subdomain, email, apiToken string) ([]ZendeskViewRef, error) {
	subdomain = strings.TrimSpace(strings.TrimSuffix(subdomain, ".zendesk.com"))
	email = strings.TrimSpace(email)
	apiToken = strings.TrimSpace(apiToken)
	if subdomain == "" || email == "" || apiToken == "" {
		return nil, fmt.Errorf("zendesk: subdomain, email, and api_token required")
	}
	auth := base64.StdEncoding.EncodeToString([]byte(email + "/token:" + apiToken))
	u := fmt.Sprintf("https://%s.zendesk.com/api/v2/views.json", subdomain)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Basic "+auth)
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
		return nil, fmt.Errorf("zendesk: status %d: %s", resp.StatusCode, string(body))
	}
	var parsed struct {
		Views []struct {
			ID     int64  `json:"id"`
			Title  string `json:"title"`
			Active bool   `json:"active"`
		} `json:"views"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("zendesk: decode: %w", err)
	}
	var out []ZendeskViewRef
	for _, v := range parsed.Views {
		if !v.Active || v.Title == "" {
			continue
		}
		out = append(out, ZendeskViewRef{ID: v.ID, Title: v.Title})
	}
	return out, nil
}
