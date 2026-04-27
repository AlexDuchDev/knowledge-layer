package filesystem

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/knowledgelayer/api/internal/ingestion_connectors"
)

type Adapter struct{}

func (Adapter) ConnectorType() string { return "filesystem" }

func (Adapter) ValidateConnectorConfig(ctx context.Context, connector ingestion_connectors.Connector) error {
	_ = ctx
	_ = connector
	return nil
}

func (Adapter) ValidateSourceFeedConfig(ctx context.Context, connector ingestion_connectors.Connector, feed *ingestion_connectors.SourceFeed) error {
	_ = ctx
	_ = connector
	if feed == nil {
		return fmt.Errorf("filesystem: nil feed")
	}
	rel := strings.TrimSpace(feed.ExternalRef)
	if rel == "" {
		return fmt.Errorf("filesystem: external_ref must be a relative file path under /data")
	}
	if filepath.IsAbs(rel) || strings.Contains(rel, "..") {
		return fmt.Errorf("filesystem: external_ref must be a safe relative path")
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
	return fmt.Errorf("filesystem adapter: use Service.SyncFilesystem or queued connector:source_sync")
}

func (Adapter) MapArtifactMetadata(ctx context.Context, artifactType string, rawPayload []byte) (map[string]any, error) {
	_ = ctx
	return map[string]any{
		"connector":     "filesystem",
		"artifact_type": artifactType,
		"raw_bytes":     len(rawPayload),
	}, nil
}
