package domain

// Connector and ingestion ports live in github.com/knowledgelayer/api/internal/ingestion_connectors:
//   - ConnectorAdapter, Registry, optional WebhookHandler (adapter.go)
//   - Service, SourceFeed, Connector, RawArtifact (service + connector_repo, raw_artifact_repo, ingestion_run_repo)
//   - sync_pipeline.go — shared ingestion run lifecycle, raw insert helper, governance merge for metadata
//   - SourceFeedGovernanceService (connector_services.go)
//   - app.SyncOrchestrator (RunSync / RunManualSync / ValidateAdapterConfig)
//
// This package stays free of imports from the concrete Service to preserve module boundary rules.
