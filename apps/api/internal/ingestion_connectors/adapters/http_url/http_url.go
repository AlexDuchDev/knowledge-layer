package http_url

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/knowledgelayer/api/internal/ingestion_connectors"
)

type Adapter struct{}

func (Adapter) ConnectorType() string { return "http_url" }

func (Adapter) ValidateConnectorConfig(ctx context.Context, connector ingestion_connectors.Connector) error {
	_ = ctx
	_ = connector
	return nil
}

func (Adapter) ValidateSourceFeedConfig(ctx context.Context, connector ingestion_connectors.Connector, feed *ingestion_connectors.SourceFeed) error {
	_ = ctx
	_ = connector
	if feed == nil {
		return fmt.Errorf("http_url: nil feed")
	}
	u := strings.TrimSpace(feed.ExternalRef)
	if u == "" {
		return fmt.Errorf("http_url: external_ref must be a URL")
	}
	parsed, err := url.Parse(u)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("http_url: external_ref must be a valid absolute URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("http_url: scheme must be http or https")
	}
	return nil
}

func (Adapter) ListAvailableFeeds(ctx context.Context, connector ingestion_connectors.Connector) ([]ingestion_connectors.AvailableFeedRef, error) {
	_ = ctx
	_ = connector
	return nil, ingestion_connectors.ErrListAvailableFeedsNotSupported
}

func (Adapter) SyncFeed(ctx context.Context, feed ingestion_connectors.SourceFeed, connector ingestion_connectors.Connector) error {
	_ = ctx
	_ = feed
	_ = connector
	return fmt.Errorf("http_url adapter: use Service.SyncHTTPURL or queued connector:source_sync")
}

func (Adapter) MapArtifactMetadata(ctx context.Context, artifactType string, rawPayload []byte) (map[string]any, error) {
	_ = ctx
	m := map[string]any{
		"connector":     "http_url",
		"artifact_type": artifactType,
	}
	if len(rawPayload) > 0 {
		m["raw_bytes"] = len(rawPayload)
	}
	// Keep metadata JSON small and non-sensitive; the actual content is stored in connector-specific fields.
	var stub map[string]any
	if err := json.Unmarshal(rawPayload, &stub); err == nil {
		m["parsed_json"] = true
	}
	return m, nil
}
