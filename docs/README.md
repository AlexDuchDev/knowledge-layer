# Documentation index

Maintained entry point for product, architecture, and operations documentation.

## Find your path (by audience)

| Audience | Start here | Then |
|----------|------------|------|
| **Evaluators** (is this real / understandable?) | [OSS_V1_SCOPE.md](OSS_V1_SCOPE.md) | [../README.md](../README.md) → [PRODUCT_CONCEPTS.md](PRODUCT_CONCEPTS.md) → [ARCHITECTURE.md](ARCHITECTURE.md) or [GETTING_STARTED.md](GETTING_STARTED.md) |
| **Operators / admins** (run & deploy) | [GETTING_STARTED.md](GETTING_STARTED.md) | [DEPLOYMENT.md](DEPLOYMENT.md) → [SETUP_EXAMPLES.md](SETUP_EXAMPLES.md) → [CONTROL_PLANE_OVERVIEW.md](CONTROL_PLANE_OVERVIEW.md) → [DEPLOY_CHECKLIST.md](DEPLOY_CHECKLIST.md) |
| **Contributors / developers** | [CONTRIBUTING.md](CONTRIBUTING.md) | [OSS_V1_SCOPE.md](OSS_V1_SCOPE.md) → [EXTERNAL_DEV_QUICKSTART.md](EXTERNAL_DEV_QUICKSTART.md) → [LIMITATIONS.md](LIMITATIONS.md) → [CONNECTOR_CAPABILITY_MATRIX.md](CONNECTOR_CAPABILITY_MATRIX.md) → [BACKEND_ARCHITECTURE.md](BACKEND_ARCHITECTURE.md) → [PERMISSION_SYSTEM.md](PERMISSION_SYSTEM.md) → [CONNECTOR_FRAMEWORK.md](CONNECTOR_FRAMEWORK.md) → [KNOWLEDGE_JOBS_ENGINE.md](KNOWLEDGE_JOBS_ENGINE.md) |
| **Product learners** (using the UI) | [USER_GUIDE.md](USER_GUIDE.md) → [USER_GUIDE_V1.md](USER_GUIDE_V1.md#local-golden-path-evaluators) | [CONTROL_PLANE_OVERVIEW.md](CONTROL_PLANE_OVERVIEW.md) → [GLOSSARY.md](GLOSSARY.md) → [LIMITATIONS.md](LIMITATIONS.md) → screen-specific docs (e.g. [CONTROL_PLANE_UI_IA.md](CONTROL_PLANE_UI_IA.md), [USER_FACING_PRODUCT_SURFACE.md](USER_FACING_PRODUCT_SURFACE.md)) |

**Examples:** [EXAMPLES.md](EXAMPLES.md), [SETUP_EXAMPLES.md](SETUP_EXAMPLES.md) · **Design:** [DESIGN_SYSTEM_AND_PAGE_TEMPLATES.md](DESIGN_SYSTEM_AND_PAGE_TEMPLATES.md) · **DigitalOcean:** [DO_DEPLOYMENT.md](DO_DEPLOYMENT.md), [DO_INFRA_TOPOLOGY.md](DO_INFRA_TOPOLOGY.md)

## Documentation maintenance (start here for changes)

| Doc | Purpose |
|-----|---------|
| [DOCS_MAINTENANCE_POLICY.md](DOCS_MAINTENANCE_POLICY.md) | When and how to update docs; mandatory rules |
| [DOCS_IMPACT_MAP.md](DOCS_IMPACT_MAP.md) | Code paths → documentation files |
| [DOCS_UPDATE_CHECKLIST.md](DOCS_UPDATE_CHECKLIST.md) | Per-task / PR checklist |
| [templates/TASK_AND_PR_DOC_IMPACT.md](templates/TASK_AND_PR_DOC_IMPACT.md) | Copy-paste completion report |

Contributor guide: [CONTRIBUTING.md](CONTRIBUTING.md) (extended); root [CONTRIBUTING.md](../CONTRIBUTING.md) (short).

Agent expectations: [AGENTS.md](../AGENTS.md) (includes documentation contract).

## Canonical product and domain

- [OSS_V1_SCOPE.md](OSS_V1_SCOPE.md) — **single-page OSS release contract** (links into limitations, IA, AI privacy).
- [PRODUCT.md](PRODUCT.md), [PRD-v1.md](PRD-v1.md)
- [DOMAIN_MODEL.md](DOMAIN_MODEL.md) (operational index), [DOMAIN_MODEL_CONTRACT.md](DOMAIN_MODEL_CONTRACT.md) (long-form contract)
- [ACCESS_MODEL.md](ACCESS_MODEL.md)
- [permission-system.md](permission-system.md), [permission-flow.md](permission-flow.md)

## Architecture and structure

- [ARCHITECTURE.md](ARCHITECTURE.md), [ARCHITECTURE_HOST.md](ARCHITECTURE_HOST.md)
- [MODULE_BOUNDARIES.md](MODULE_BOUNDARIES.md), [backend-architecture.md](backend-architecture.md)
- [REPO_STRUCTURE.md](REPO_STRUCTURE.md), [apps:api.md](apps:api.md), [apps:web.md](apps:web.md)
- [adr/](adr/) — architecture decision records (includes [ADR-0011](adr/0011-internal-modules-pilot-scope.md) — `internal/modules` pilot scope)

## Ingestion, jobs, retrieval, AI

- [CONNECTOR_CAPABILITY_MATRIX.md](CONNECTOR_CAPABILITY_MATRIX.md) — sync vs raw artifact types vs worker normalization (single table).
- [INGESTION_AND_CONNECTORS.md](INGESTION_AND_CONNECTORS.md), [INGESTION_API.md](INGESTION_API.md)
- [SECOND_BRAIN_CLIENT_GAP_MATRIX.md](SECOND_BRAIN_CLIENT_GAP_MATRIX.md) — mapping a “Second Brain”-style client brief to KL capabilities (positioning, not a client spec).
- [SECOND_BRAIN_OVERLAY_SIZING.md](SECOND_BRAIN_OVERLAY_SIZING.md) — effort sizing for bots, pre-meeting, Meet capture, extraction product path, OKR metrics; pre-kickoff scope checklists.
- [EXTRACTED_MEETING_TASKS.md](EXTRACTED_MEETING_TASKS.md) — draft/confirm lifecycle for meeting-sourced tasks.
- [FIREFLIES_SECURITY.md](FIREFLIES_SECURITY.md) — token, retention, and access boundaries for transcript providers.
- [KNOWLEDGE_JOBS.md](KNOWLEDGE_JOBS.md), [knowledge-jobs-engine.md](knowledge-jobs-engine.md)
- [AI_RETRIEVAL_GOVERNANCE.md](AI_RETRIEVAL_GOVERNANCE.md), [retrieval-ai-foundation.md](retrieval-ai-foundation.md)

## API and admin

- [API_SURFACE_V1.md](API_SURFACE_V1.md)
- [ADMIN_GUIDE_V1.md](ADMIN_GUIDE_V1.md), [ADMIN_UI_V1.md](ADMIN_UI_V1.md)
- [USER_GUIDE_V1.md](USER_GUIDE_V1.md)

## UX and information architecture

- [INFORMATION_ARCHITECTURE_V1.md](INFORMATION_ARCHITECTURE_V1.md), [INFORMATION_ARCHITECTURE_PRODUCT_V1.md](INFORMATION_ARCHITECTURE_PRODUCT_V1.md)
- [ADMIN_UI_CONSOLIDATION_PLAN.md](ADMIN_UI_CONSOLIDATION_PLAN.md) — canonical `/control-plane` admin URLs and migration
- [control-plane-ui-ia.md](control-plane-ui-ia.md), [user-facing-product-surface.md](user-facing-product-surface.md)
- [SEARCH_AND_QA_UX.md](SEARCH_AND_QA_UX.md)

## Deployment and operations

- [RUNBOOK_STAGING_PROD.md](RUNBOOK_STAGING_PROD.md) — **ordered golden path** (links into the docs below).
- [SELF_HOSTED.md](SELF_HOSTED.md), [OPERATIONS.md](OPERATIONS.md)
- [DEPLOY_CHECKLIST.md](DEPLOY_CHECKLIST.md), [PRODUCTION_HARDENING.md](PRODUCTION_HARDENING.md), [PRODUCTION_GO_LIVE_CHECKLIST.md](PRODUCTION_GO_LIVE_CHECKLIST.md), [INFRA_PRODUCTION_REFERENCE.md](INFRA_PRODUCTION_REFERENCE.md), [PRODUCTION_CUTOVER_QUICKREF.md](PRODUCTION_CUTOVER_QUICKREF.md), [RELEASING.md](RELEASING.md), [SLO_AND_ALERTING_TEMPLATE.md](SLO_AND_ALERTING_TEMPLATE.md)
- [STAGING_SMOKE_TEST.md](STAGING_SMOKE_TEST.md), [CONFIG_ENV.md](CONFIG_ENV.md)
- [OPENSEARCH_PROD_VS_DEV.md](OPENSEARCH_PROD_VS_DEV.md) — security split for OpenSearch.

## Other

- [LIMITATIONS.md](LIMITATIONS.md) — stubs and degraded behavior (set expectations for OSS and ops).
- [EXTERNAL_DEV_QUICKSTART.md](EXTERNAL_DEV_QUICKSTART.md) — canonical URLs after local bring-up.
- Historical or deep-dive docs remain in this directory; prefer this index and the impact map when choosing what to update.

For a minimal legacy stub, see [docs.md](docs.md).
