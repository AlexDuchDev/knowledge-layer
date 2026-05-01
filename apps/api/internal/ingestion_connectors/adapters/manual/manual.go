// Package manual is the connector adapter for manually-uploaded content.
// All ingestion logic and shared types live in the parent ingestion_connectors
// package (see manual.go, manual_extract.go, manual_youtube.go); this package
// only carries the ConnectorAdapter contract implementation.
package manual

import (
	"context"
	"fmt"
	"strings"

	"github.com/knowledgelayer/api/internal/ingestion_connectors"
)

// Adapter implements ingestion_connectors.ConnectorAdapter for the manual
// connector. SyncFeed is intentionally a no-op — manual artifacts arrive via
// the /api/manual/collections/:id/* upload routes, not by polling.
type Adapter struct{}

func (Adapter) ConnectorType() string { return ingestion_connectors.ConnectorTypeManual }

func (Adapter) ValidateConnectorConfig(ctx context.Context, connector ingestion_connectors.Connector) error {
	_ = ctx
	_ = connector
	return nil
}

func (Adapter) ValidateSourceFeedConfig(ctx context.Context, connector ingestion_connectors.Connector, feed *ingestion_connectors.SourceFeed) error {
	_ = ctx
	_ = connector
	if feed == nil {
		return fmt.Errorf("manual: nil feed")
	}
	cfg, err := ingestion_connectors.ParseManualCollectionConfig(feed.ConnectorConfigJSON)
	if err != nil {
		return err
	}
	if strings.TrimSpace(cfg.Label) == "" {
		return fmt.Errorf("manual: collection.label is required in connector_config_json")
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
	// Manual artifacts arrive through HTTP upload handlers, not sync. Returning
	// nil is safe in the rare case a generic sync trigger fires for a manual
	// feed (e.g. an operator clicks "Sync now" on the source-feed list).
	return nil
}

func (Adapter) MapArtifactMetadata(ctx context.Context, artifactType string, rawPayload []byte) (map[string]any, error) {
	_ = ctx
	m := map[string]any{
		"connector":     ingestion_connectors.ConnectorTypeManual,
		"artifact_type": artifactType,
	}
	if len(rawPayload) > 0 {
		m["raw_bytes"] = len(rawPayload)
	}
	return m, nil
}
