// Package slack implements the Slack connector adapter (see docs/chat-connector-family.md).
package slack

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/knowledgelayer/api/internal/ingestion_connectors"
	"github.com/knowledgelayer/api/internal/ingestion_connectors/families/chat"
)

// Adapter is v1 bot-token Slack support: one channel per source feed (conversations.history + replies).
type Adapter struct{}

func (Adapter) ConnectorType() string { return "slack" }

func (Adapter) ValidateConnectorConfig(ctx context.Context, connector ingestion_connectors.Connector) error {
	_ = ctx
	_ = connector
	return nil
}

func (Adapter) ValidateSourceFeedConfig(ctx context.Context, connector ingestion_connectors.Connector, feed *ingestion_connectors.SourceFeed) error {
	_ = ctx
	_ = connector
	if feed == nil {
		return fmt.Errorf("slack: nil feed")
	}
	cfg, err := ingestion_connectors.ParseSlackFeedConfig(feed.ConnectorConfigJSON)
	if err != nil {
		return fmt.Errorf("slack: config: %w", err)
	}
	if feed.Status == "active" {
		return ingestion_connectors.ValidateSlackV1ForActivation(feed, cfg)
	}
	if cfg.BotToken == "" {
		return fmt.Errorf("slack: bot_token required before sync")
	}
	if err := chat.RequireFeedKindForSlack(feed.ConnectorConfigJSON); err != nil {
		return fmt.Errorf("slack: %w", err)
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
	return fmt.Errorf("slack adapter: use Service.SyncSlack or queued connector:source_sync")
}

func (Adapter) MapArtifactMetadata(ctx context.Context, artifactType string, rawPayload []byte) (map[string]any, error) {
	_ = ctx
	m := map[string]any{
		"connector":     "slack",
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
