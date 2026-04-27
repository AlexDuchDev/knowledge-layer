package app

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/knowledgelayer/api/internal/ingestion_connectors"
	platformmetrics "github.com/knowledgelayer/api/internal/platform/metrics"
)

// SyncOrchestrator coordinates governed source feed sync: load feed → validate → resolve adapter → fetch → raw artifacts → audit hooks.
// HTTP should enqueue work (see platform/queue) when Redis is configured; this type documents the intended pipeline.
type SyncOrchestrator struct{}

// RunManualSync executes an inline sync using the ingestion Service (used by API fallback and connectorworker).
func (SyncOrchestrator) RunManualSync(ctx context.Context, svc *ingestion_connectors.Service, feedID uuid.UUID) (*ingestion_connectors.IngestionRun, error) {
	if svc == nil {
		return nil, fmt.Errorf("sync orchestrator: nil ingestion service")
	}
	return svc.SyncSourceFeed(ctx, feedID)
}

// RunSync is the single entry point name for workers: same behavior as RunManualSync (load feed → validate → adapter sync → raw artifacts).
// Records `connector_sync_duration_seconds{adapter_kind,status}` for /metrics.
func (o SyncOrchestrator) RunSync(ctx context.Context, svc *ingestion_connectors.Service, feedID uuid.UUID) (*ingestion_connectors.IngestionRun, error) {
	adapterKind := "unknown"
	if svc != nil {
		if feed, err := svc.GetSourceFeed(ctx, feedID); err == nil && feed != nil {
			if conn, cErr := svc.GetConnector(ctx, feed.ConnectorID); cErr == nil && conn != nil {
				adapterKind = conn.Type
			}
		}
	}
	start := time.Now()
	run, err := o.RunManualSync(ctx, svc, feedID)
	status := "completed"
	if err != nil {
		status = "failed"
	}
	platformmetrics.ObserveConnectorSync(adapterKind, status, time.Since(start))
	return run, err
}

// ValidateAdapterConfig runs connector adapter validation when a registry entry exists.
func (SyncOrchestrator) ValidateAdapterConfig(ctx context.Context, svc *ingestion_connectors.Service, feed *ingestion_connectors.SourceFeed, conn *ingestion_connectors.Connector) error {
	if svc == nil || svc.Registry == nil || feed == nil || conn == nil {
		return nil
	}
	a, err := svc.Registry.AdapterForConnectorType(conn.Type)
	if err != nil {
		return nil
	}
	if err := a.ValidateConnectorConfig(ctx, *conn); err != nil {
		return err
	}
	return a.ValidateSourceFeedConfig(ctx, *conn, feed)
}
