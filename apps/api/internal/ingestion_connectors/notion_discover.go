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

// NotionSearchRef is a page or database from workspace search (picker for external_ref).
type NotionSearchRef struct {
	Object string `json:"object"` // page | database
	ID     string `json:"id"`
	Title  string `json:"title"`
}

// ListNotionSearchResults returns recent pages and databases (bounded) for integration token.
func (s *Service) ListNotionSearchResults(ctx context.Context, integrationToken string) ([]NotionSearchRef, error) {
	integrationToken = strings.TrimSpace(integrationToken)
	if integrationToken == "" {
		return nil, fmt.Errorf("notion: integration token required")
	}
	body := map[string]any{"page_size": 50}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.notion.com/v1/search", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+integrationToken)
	req.Header.Set("Notion-Version", notionAPIVersion)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("notion: status %d: %s", resp.StatusCode, string(b))
	}
	var parsed struct {
		Results []json.RawMessage `json:"results"`
	}
	if err := json.Unmarshal(b, &parsed); err != nil {
		return nil, fmt.Errorf("notion: decode: %w", err)
	}
	out := make([]NotionSearchRef, 0, len(parsed.Results))
	for _, r := range parsed.Results {
		var head struct {
			Object string `json:"object"`
			ID     string `json:"id"`
		}
		if json.Unmarshal(r, &head) != nil || head.ID == "" {
			continue
		}
		title := notionSearchItemTitle(r, head.Object)
		out = append(out, NotionSearchRef{Object: head.Object, ID: head.ID, Title: title})
	}
	return out, nil
}

func notionSearchItemTitle(raw []byte, object string) string {
	switch object {
	case "page":
		var page struct {
			Properties map[string]json.RawMessage `json:"properties"`
		}
		if json.Unmarshal(raw, &page) == nil && page.Properties != nil {
			props := notionPageProperties{Properties: page.Properties}
			if t := notionExtractTitle(props); t != "" {
				return t
			}
		}
	case "database":
		var db struct {
			Title []struct {
				PlainText string `json:"plain_text"`
			} `json:"title"`
		}
		if json.Unmarshal(raw, &db) == nil && len(db.Title) > 0 {
			return db.Title[0].PlainText
		}
	}
	return ""
}
