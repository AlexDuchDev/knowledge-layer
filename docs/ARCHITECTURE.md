# Architecture

This page is the **navigation hub** for system architecture. Deep content lives in linked documents (single source of truth—no duplicate prose).

## Read this first

- **This document** — architecture navigation hub (readable on GitHub).
- **[ARCHITECTURE_HOST.md](ARCHITECTURE_HOST.md)** — overview suitable for quick orientation.
- **[MODULE_BOUNDARIES.md](MODULE_BOUNDARIES.md)** — modular monolith seams.
- **[backend-architecture.md](backend-architecture.md)** — API layout, packages, workers.
- **[apps:api.md](apps:api.md)** — API app map.
- **[apps:web.md](apps:web.md)** — web app map.
- **[adr/](adr/)** — architecture decision records.

## By concern


| Concern                | Document                                                                                                                                                     |
| ---------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Permissions            | [PERMISSION_SYSTEM.md](PERMISSION_SYSTEM.md)                                                                                                                 |
| Connectors & ingestion | [CONNECTOR_FRAMEWORK.md](CONNECTOR_FRAMEWORK.md), [INGESTION_AND_CONNECTORS.md](INGESTION_AND_CONNECTORS.md)                                                 |
| Jobs                   | [KNOWLEDGE_JOBS_ENGINE.md](KNOWLEDGE_JOBS_ENGINE.md), [KNOWLEDGE_JOBS.md](KNOWLEDGE_JOBS.md)                                                                 |
| Retrieval & AI         | [AI_RETRIEVAL_GOVERNANCE.md](AI_RETRIEVAL_GOVERNANCE.md), [retrieval-ai-foundation.md](retrieval-ai-foundation.md), [AI_PRIVACY_FLOW.md](AI_PRIVACY_FLOW.md) |
| Control plane UI       | [CONTROL_PLANE_UI_IA.md](CONTROL_PLANE_UI_IA.md)                                                                                                             |
| Product surface        | [USER_FACING_PRODUCT_SURFACE.md](USER_FACING_PRODUCT_SURFACE.md)                                                                                             |
| Design                 | [DESIGN_SYSTEM_AND_PAGE_TEMPLATES.md](DESIGN_SYSTEM_AND_PAGE_TEMPLATES.md)                                                                                   |
| Staging / production   | [RUNBOOK_STAGING_PROD.md](RUNBOOK_STAGING_PROD.md), [OPENSEARCH_PROD_VS_DEV.md](OPENSEARCH_PROD_VS_DEV.md), [STAGING_SMOKE_TEST.md](STAGING_SMOKE_TEST.md)   |


## Maintenance

When architecture **behavior** changes, update the linked canonical doc and, if needed, ADRs. See [DOCS_IMPACT_MAP.md](DOCS_IMPACT_MAP.md).