package openapi_v3

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/knowledgelayer/api/internal/ingestion_connectors"
)

// Adapter implements ingestion_connectors.ConnectorAdapter for the
// generic openapi_v3 connector type. SyncFeed is intentionally a stub —
// the live sync is driven by Service.SyncOpenAPIV3 (or the connectorworker
// queued task), which has access to the persistence helpers (raw_artifacts,
// PersistNormalizedRecord). Keeping the adapter thin avoids importing the
// service layer here.
type Adapter struct{}

func (Adapter) ConnectorType() string { return "openapi_v3" }

func (Adapter) ValidateConnectorConfig(ctx context.Context, connector ingestion_connectors.Connector) error {
	_ = ctx
	_ = connector
	return nil
}

func (Adapter) ValidateSourceFeedConfig(ctx context.Context, connector ingestion_connectors.Connector, feed *ingestion_connectors.SourceFeed) error {
	_ = connector
	if feed == nil {
		return fmt.Errorf("openapi_v3: nil feed")
	}
	cfg, err := decodeFeedConfig(feed.ConnectorConfigJSON)
	if err != nil {
		return err
	}
	if err := cfg.Validate(); err != nil {
		return err
	}
	// Spec fetch is the network-touching part of validation. Skip on tests
	// that pass a feed without ExternalRef hint pointing at a fake server,
	// but always try when openapi_url is set — Validate() requires it.
	if _, err := FetchAndValidateSpec(ctx, cfg.OpenAPIURL, cfg.ListPath); err != nil {
		return err
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
	// SyncFeed delegates to the service layer (Service.SyncOpenAPIV3 +
	// queued connector:source_sync) which holds the DB pool and the
	// PersistNormalizedRecord helper. This keeps the adapter package
	// dependency-free apart from the connector contract types.
	return fmt.Errorf("openapi_v3 adapter: use Service.SyncOpenAPIV3 or queued connector:source_sync")
}

func (Adapter) MapArtifactMetadata(ctx context.Context, artifactType string, rawPayload []byte) (map[string]any, error) {
	_ = ctx
	m := map[string]any{
		"connector":     "openapi_v3",
		"artifact_type": artifactType,
	}
	if len(rawPayload) > 0 {
		m["raw_bytes"] = len(rawPayload)
		var stub map[string]any
		if err := json.Unmarshal(rawPayload, &stub); err == nil {
			m["parsed_json"] = true
		}
	}
	return m, nil
}

// decodeFeedConfig parses source_feeds.connector_config_json into the
// FeedConfig shape this adapter expects. Empty config → zero-value error
// (Validate will catch it).
func decodeFeedConfig(raw json.RawMessage) (*FeedConfig, error) {
	var cfg FeedConfig
	if len(raw) == 0 {
		return &cfg, nil
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("openapi_v3: parse connector_config_json: %w", err)
	}
	return &cfg, nil
}
