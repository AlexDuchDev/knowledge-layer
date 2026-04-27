// Package asana implements the Asana connector adapter (work_mgmt family).
package asana

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/knowledgelayer/api/internal/ingestion_connectors"
)

type Adapter struct{}

func (Adapter) ConnectorType() string { return "asana" }

func (Adapter) ValidateConnectorConfig(ctx context.Context, connector ingestion_connectors.Connector) error {
	_ = ctx
	_ = connector
	return nil
}

func (Adapter) ValidateSourceFeedConfig(ctx context.Context, connector ingestion_connectors.Connector, feed *ingestion_connectors.SourceFeed) error {
	_ = ctx
	_ = connector
	if feed == nil {
		return fmt.Errorf("asana: nil feed")
	}
	if feed.Status != "active" {
		return nil
	}
	return ingestion_connectors.ValidateAsanaSourceFeedForActivation(feed)
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
	return fmt.Errorf("asana adapter: use Service.SyncAsana or queued connector:source_sync")
}

func (Adapter) MapArtifactMetadata(ctx context.Context, artifactType string, rawPayload []byte) (map[string]any, error) {
	_ = ctx
	m := map[string]any{"connector": "asana", "artifact_type": artifactType}
	if len(rawPayload) > 0 {
		var stub map[string]any
		if err := json.Unmarshal(rawPayload, &stub); err == nil {
			m["parsed"] = stub
		}
	}
	return m, nil
}
