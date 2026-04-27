// Package telegram implements the Telegram connector adapter (see docs/connector-framework.md).
package telegram

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/knowledgelayer/api/internal/ingestion_connectors"
)

// Adapter is v1 ingestion-only Telegram support: explicitly configured chats per source feed (no crawling).
type Adapter struct{}

func (Adapter) ConnectorType() string { return "telegram" }

func (Adapter) ValidateConnectorConfig(ctx context.Context, connector ingestion_connectors.Connector) error {
	_ = ctx
	_ = connector
	return nil
}

func (Adapter) ValidateSourceFeedConfig(ctx context.Context, connector ingestion_connectors.Connector, feed *ingestion_connectors.SourceFeed) error {
	_ = ctx
	_ = connector
	if feed == nil {
		return fmt.Errorf("telegram: nil feed")
	}
	cfg, err := ingestion_connectors.ParseTelegramFeedConfig(feed.ConnectorConfigJSON)
	if err != nil {
		return fmt.Errorf("telegram: config: %w", err)
	}
	// Activation path: require explicit allowlist and primary external_ref (draft feeds may omit until configured).
	if feed.Status == "active" {
		return ingestion_connectors.ValidateTelegramV1ForActivation(feed, cfg)
	}
	if cfg.BotToken == "" {
		return fmt.Errorf("telegram: bot_token required before sync")
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
	// Persistence and run rows are handled by ingestion_connectors.Service.SyncTelegram / worker; adapter stays a contract anchor.
	return fmt.Errorf("telegram adapter: use Service.SyncTelegram or queued connector:source_sync")
}

func (Adapter) MapArtifactMetadata(ctx context.Context, artifactType string, rawPayload []byte) (map[string]any, error) {
	_ = ctx
	m := map[string]any{
		"connector":     "telegram",
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
