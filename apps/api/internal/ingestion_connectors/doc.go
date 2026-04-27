// Package ingestion_connectors implements the connector framework: connector and source_feed
// persistence, sync orchestration, and a typed adapter registry (ConnectorAdapter).
//
// Adapters are registered by connector type and must validate feed config before activation.
// See docs/CONNECTOR_FRAMEWORK.md and adapter.go for the contract.
package ingestion_connectors
