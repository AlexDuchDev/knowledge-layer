// Package mattermost implements the Mattermost connector adapter (v1: PAT + channel posts).
package mattermost

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/knowledgelayer/api/internal/ingestion_connectors"
)

// Adapter is v1 Mattermost support: one channel per source feed (posts API).
type Adapter struct{}

func (Adapter) ConnectorType() string { return "mattermost" }

func (Adapter) ValidateConnectorConfig(ctx context.Context, connector ingestion_connectors.Connector) error {
	_ = ctx
	_ = connector
	return nil
}

func (Adapter) ValidateSourceFeedConfig(ctx context.Context, connector ingestion_connectors.Connector, feed *ingestion_connectors.SourceFeed) error {
	_ = ctx
	_ = connector
	if feed == nil {
		return fmt.Errorf("mattermost: nil feed")
	}
	cfg, err := ingestion_connectors.ParseMattermostFeedConfig(feed.ConnectorConfigJSON)
	if err != nil {
		return fmt.Errorf("mattermost: config: %w", err)
	}
	if feed.Status == "active" {
		return ingestion_connectors.ValidateMattermostV1ForActivation(feed, cfg)
	}
	if cfg.Token == "" || cfg.BaseURL == "" {
		return fmt.Errorf("mattermost: mattermost_base_url and mattermost_token required before sync")
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
	return fmt.Errorf("mattermost adapter: use Service.SyncMattermost or queued connector:source_sync")
}

func (Adapter) MapArtifactMetadata(ctx context.Context, artifactType string, rawPayload []byte) (map[string]any, error) {
	_ = ctx
	m := map[string]any{
		"connector":     "mattermost",
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
