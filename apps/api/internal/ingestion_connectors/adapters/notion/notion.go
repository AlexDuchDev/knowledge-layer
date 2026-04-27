// Package notion implements the Notion connector adapter (see docs/docs-wiki-connector-family.md).
package notion

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/knowledgelayer/api/internal/ingestion_connectors"
	"github.com/knowledgelayer/api/internal/ingestion_connectors/families/docs_wiki"
)

// Adapter is v1 Notion integration: one page or database per source feed.
type Adapter struct{}

func (Adapter) ConnectorType() string { return "notion" }

func (Adapter) ValidateConnectorConfig(ctx context.Context, connector ingestion_connectors.Connector) error {
	_ = ctx
	_ = connector
	return nil
}

func (Adapter) ValidateSourceFeedConfig(ctx context.Context, connector ingestion_connectors.Connector, feed *ingestion_connectors.SourceFeed) error {
	_ = ctx
	_ = connector
	if feed == nil {
		return fmt.Errorf("notion: nil feed")
	}
	scopeCfg, err := docs_wiki.ParseNotionConfig(feed.ConnectorConfigJSON)
	if err != nil {
		return fmt.Errorf("notion: config: %w", err)
	}
	token, _ := docs_wiki.RequireNotionToken(feed.ConnectorConfigJSON)
	if feed.Status == "active" {
		return ingestion_connectors.ValidateNotionV1ForActivation(feed, token, scopeCfg)
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
	return fmt.Errorf("notion adapter: use Service.SyncNotion or queued connector:source_sync")
}

func (Adapter) MapArtifactMetadata(ctx context.Context, artifactType string, rawPayload []byte) (map[string]any, error) {
	_ = ctx
	m := map[string]any{
		"connector":     "notion",
		"artifact_type": artifactType,
	}
	if len(rawPayload) > 0 {
		var stub map[string]any
		if err := json.Unmarshal(rawPayload, &stub); err == nil {
			m["parsed"] = stub
		}
	}
	return m, nil
}
