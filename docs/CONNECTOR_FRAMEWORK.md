# Connector framework

**Canonical:** [connector-framework.md](connector-framework.md), [INGESTION_AND_CONNECTORS.md](INGESTION_AND_CONNECTORS.md), [INGESTION_API.md](INGESTION_API.md).

**Adapter contract:** `apps/api/internal/ingestion_connectors/adapter.go` — `ConnectorAdapter` interface and `Registry`.

**Per-family specs:** `docs/*-connector-family.md`, [TELEGRAM_CONNECTOR_V1.md](TELEGRAM_CONNECTOR_V1.md).

When adding or changing connector behavior, update ingestion docs and the relevant family document.
