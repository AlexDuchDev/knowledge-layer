package opensearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

const defaultIndex = "entity_chunks"

// Client is a minimal OpenSearch HTTP client (no extra dependencies).
type Client struct {
	baseURL    string
	index      string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil
	}
	return &Client{
		baseURL: baseURL,
		index:   defaultIndex,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (c *Client) IndexName() string { return c.index }

func (c *Client) EnsureIndex(ctx context.Context) error {
	if c == nil {
		return nil
	}
	mapping := map[string]any{
		"mappings": map[string]any{
			"properties": map[string]any{
				"entity_id":   map[string]any{"type": "keyword"},
				"domain_id":   map[string]any{"type": "keyword"},
				"entity_type": map[string]any{"type": "keyword"},
				"title":       map[string]any{"type": "text"},
				"text":        map[string]any{"type": "text"},
				"updated_at":  map[string]any{"type": "date"},
			},
		},
	}
	body, _ := json.Marshal(mapping)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.baseURL+"/"+c.index, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("opensearch create index %s: %s", resp.Status, string(b))
	}
	return nil
}

// IndexUpsert indexes one document; id is entity_id for replace semantics.
func (c *Client) IndexUpsert(ctx context.Context, entityID uuid.UUID, doc map[string]any) error {
	if c == nil {
		return nil
	}
	body, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/%s/_doc/%s?refresh=wait_for", c.baseURL, c.index, entityID.String())
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("opensearch index %s: %s", resp.Status, string(b))
	}
	return nil
}

func (c *Client) DeleteDocument(ctx context.Context, entityID uuid.UUID) error {
	if c == nil {
		return nil
	}
	url := fmt.Sprintf("%s/%s/_doc/%s", c.baseURL, c.index, entityID.String())
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// 404 is OK
	return nil
}

// SearchEntityIDs runs a bool query with mandatory domain filter; returns ordered entity ids.
func (c *Client) SearchEntityIDs(ctx context.Context, query string, domainIDs []uuid.UUID, entityType string, limit int) ([]uuid.UUID, error) {
	if c == nil {
		return nil, fmt.Errorf("opensearch client not configured")
	}
	if limit <= 0 || limit > 200 {
		limit = 200
	}
	domainStrs := make([]string, 0, len(domainIDs))
	for _, d := range domainIDs {
		domainStrs = append(domainStrs, d.String())
	}
	boolQ := map[string]any{
		"filter": []any{
			map[string]any{"terms": map[string]any{"domain_id": domainStrs}},
		},
		"must": []any{
			map[string]any{
				"multi_match": map[string]any{
					"query":  query,
					"fields": []string{"title^3", "text"},
					"type":   "best_fields",
				},
			},
		},
	}
	if entityType != "" {
		boolQ["filter"] = append(boolQ["filter"].([]any), map[string]any{
			"term": map[string]any{"entity_type": entityType},
		})
	}
	payload := map[string]any{
		"size": limit,
		"query": map[string]any{
			"bool": boolQ,
		},
		"_source": false,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/%s/_search", c.baseURL, c.index)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("opensearch search %s: %s", resp.Status, string(respBody))
	}
	var parsed struct {
		Hits struct {
			Hits []struct {
				ID string `json:"_id"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, err
	}
	out := make([]uuid.UUID, 0, len(parsed.Hits.Hits))
	for _, h := range parsed.Hits.Hits {
		id, err := uuid.Parse(h.ID)
		if err != nil {
			continue
		}
		out = append(out, id)
	}
	return out, nil
}

func (c *Client) Ping(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("opensearch not configured")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL, nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		return fmt.Errorf("opensearch ping: %s", resp.Status)
	}
	return nil
}
