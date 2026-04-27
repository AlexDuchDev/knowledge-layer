# Module: ingestion_connectors

**Purpose:** HTTP and app wiring for connectors, source feeds, sync, and raw/normalized artifact routes.

**Flows:** Register connector, create feed, activate, trigger sync, inspect artifacts.

**Integrates with:** `internal/ingestion_connectors` service and `ConnectorAdapter` registry.

**Anti-pattern:** Do not bypass adapter validation for activation ([`adapter.go`](../../ingestion_connectors/adapter.go)).

**Docs:** [docs/CONNECTOR_FRAMEWORK.md](../../../../../docs/CONNECTOR_FRAMEWORK.md).
