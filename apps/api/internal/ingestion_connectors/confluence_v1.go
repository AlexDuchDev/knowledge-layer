package ingestion_connectors

import (
	"encoding/json"
	"errors"
	"strings"
)

// confluenceFeedConfig is v1 Confluence Cloud (REST) source feed configuration.
type confluenceFeedConfig struct {
	BaseURL  string `json:"confluence_base_url"`  // e.g. https://your-domain.atlassian.net/wiki
	Auth     string `json:"confluence_auth"`      // PAT or OAuth access token (Bearer)
	FeedKind string `json:"confluence_feed_kind"` // space | page_collection | content_tree
}

func parseConfluenceFeedConfig(raw json.RawMessage) (*confluenceFeedConfig, error) {
	var c confluenceFeedConfig
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, err
	}
	c.BaseURL = strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	c.Auth = strings.TrimSpace(c.Auth)
	c.FeedKind = strings.ToLower(strings.TrimSpace(c.FeedKind))
	if c.BaseURL == "" {
		return nil, errors.New("confluence: confluence_base_url required")
	}
	if c.Auth == "" {
		return nil, errors.New("confluence: confluence_auth required")
	}
	switch c.FeedKind {
	case "space", "page_collection", "content_tree":
	default:
		return nil, errors.New("confluence: confluence_feed_kind must be space, page_collection, or content_tree")
	}
	return &c, nil
}

// ValidateConfluenceSourceFeedForActivation validates an active Confluence feed.
func ValidateConfluenceSourceFeedForActivation(feed *SourceFeed) error {
	if feed == nil {
		return errors.New("confluence: nil feed")
	}
	cfg, err := parseConfluenceFeedConfig(feed.ConnectorConfigJSON)
	if err != nil {
		return err
	}
	ref := strings.TrimSpace(feed.ExternalRef)
	if ref == "" {
		return errors.New("confluence: external_ref required")
	}
	switch cfg.FeedKind {
	case "space":
		return nil
	case "page_collection":
		ids := splitCommaIDs(ref)
		if len(ids) == 0 {
			return errors.New("confluence page_collection: external_ref must list at least one page id")
		}
	case "content_tree":
		if ref == "" {
			return errors.New("confluence content_tree: external_ref must be root page id")
		}
	}
	return nil
}

func splitCommaIDs(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
