# Documentation impact map

Path-oriented guide: **when you change code under these paths**, review/update these docs (and in-product copy where noted).

Use with [DOCS_MAINTENANCE_POLICY.md](DOCS_MAINTENANCE_POLICY.md) and [DOCS_UPDATE_CHECKLIST.md](DOCS_UPDATE_CHECKLIST.md).

---

## Backend — modular monolith (`apps/api`)

### Identity and access

| Code paths (primary) | Documentation to review/update |
|----------------------|--------------------------------|
| `apps/api/internal/identity_access/` | [ACCESS_MODEL.md](ACCESS_MODEL.md), [permission-system.md](permission-system.md), [permission-flow.md](permission-flow.md), [DOMAIN_MODEL.md](DOMAIN_MODEL.md) (grants, principals, domains) |
| `apps/api/internal/modules/identity_access/` | Same as above |
| `apps/api/internal/auth/` (session, cookies) | [ACCESS_MODEL.md](ACCESS_MODEL.md), [SELF_HOSTED.md](SELF_HOSTED.md), [PRODUCTION_HARDENING.md](PRODUCTION_HARDENING.md) |
| `apps/api/internal/httpserver/middleware.go`, route registration for `/users`, `/domains`, `/access/*`, login | [ACCESS_MODEL.md](ACCESS_MODEL.md), [API_SURFACE_V1.md](API_SURFACE_V1.md) |
| `apps/api/internal/platform/permissions/` | [permission-system.md](permission-system.md), [ACCESS_MODEL.md](ACCESS_MODEL.md) |

### Knowledge core

| Code paths (primary) | Documentation to review/update |
|----------------------|--------------------------------|
| `apps/api/internal/knowledge_core/` | [DOMAIN_MODEL.md](DOMAIN_MODEL.md), [API_SURFACE_V1.md](API_SURFACE_V1.md) |
| `apps/api/internal/modules/knowledge_core/` | Same + [MODULE_BOUNDARIES.md](MODULE_BOUNDARIES.md) |

### Ingestion and connectors

| Code paths (primary) | Documentation to review/update |
|----------------------|--------------------------------|
| `apps/api/internal/ingestion_connectors/` | [INGESTION_AND_CONNECTORS.md](INGESTION_AND_CONNECTORS.md), [INGESTION_API.md](INGESTION_API.md), [CONNECTOR_CAPABILITY_MATRIX.md](CONNECTOR_CAPABILITY_MATRIX.md), [connector-framework.md](connector-framework.md), [chat-connector-family.md](chat-connector-family.md), [meeting-transcript-connector-family.md](meeting-transcript-connector-family.md), [LIMITATIONS.md](LIMITATIONS.md) |
| `apps/api/internal/db/migrations/*extracted*` / `*second_brain*` / meeting task tables | [EXTRACTED_MEETING_TASKS.md](EXTRACTED_MEETING_TASKS.md), [SECOND_BRAIN_CLIENT_GAP_MATRIX.md](SECOND_BRAIN_CLIENT_GAP_MATRIX.md), [SECOND_BRAIN_OVERLAY_SIZING.md](SECOND_BRAIN_OVERLAY_SIZING.md), [SECOND_BRAIN_BOTS.md](SECOND_BRAIN_BOTS.md), [LIMITATIONS.md](LIMITATIONS.md), [API_SURFACE_V1.md](API_SURFACE_V1.md) §14.1 |
| `apps/api/internal/extracted_meeting_tasks/`, `apps/api/internal/secondbrain/`, `apps/api/internal/httpserver/secondbrain_webhooks.go` | [API_SURFACE_V1.md](API_SURFACE_V1.md) §14.1, [SECOND_BRAIN_BOTS.md](SECOND_BRAIN_BOTS.md), [CONFIG_ENV.md](CONFIG_ENV.md) |
| `apps/web/src/app/(dash)/meeting-tasks/` | [EXTRACTED_MEETING_TASKS.md](EXTRACTED_MEETING_TASKS.md) (operator UX) |
| `apps/api/internal/blobstore/` | [CONFIG_ENV.md](CONFIG_ENV.md), [SELF_HOSTED.md](SELF_HOSTED.md), [connector-framework.md](connector-framework.md) |
| `apps/api/internal/ingestion_connectors/adapters/*` | Family docs: [chat-connector-family.md](chat-connector-family.md), [docs-wiki-connector-family.md](docs-wiki-connector-family.md), [email-connector-family.md](email-connector-family.md), [meeting-transcript-connector-family.md](meeting-transcript-connector-family.md), [microsoft-365-connector-family.md](microsoft-365-connector-family.md), [crm-support-connector-family.md](crm-support-connector-family.md), [work-management-connector-family.md](work-management-connector-family.md), [TELEGRAM_CONNECTOR_V1.md](TELEGRAM_CONNECTOR_V1.md), etc. |
| `apps/api/internal/modules/ingestion_connectors/` | [INGESTION_AND_CONNECTORS.md](INGESTION_AND_CONNECTORS.md), [API_SURFACE_V1.md](API_SURFACE_V1.md) |
| `apps/api/cmd/connectorworker/` | [INGESTION_AND_CONNECTORS.md](INGESTION_AND_CONNECTORS.md), [connector-framework.md](connector-framework.md), [DEPLOY_CHECKLIST.md](DEPLOY_CHECKLIST.md), [apps:workers.md](apps:workers.md) (artifact task → `audit_events`) |

### Knowledge jobs

| Code paths (primary) | Documentation to review/update |
|----------------------|--------------------------------|
| `apps/api/internal/knowledge_jobs/` | [KNOWLEDGE_JOBS.md](KNOWLEDGE_JOBS.md), [knowledge-jobs-engine.md](knowledge-jobs-engine.md), [job-builder.md](job-builder.md) |
| `apps/api/internal/modules/knowledge_jobs/` | Same + [API_SURFACE_V1.md](API_SURFACE_V1.md) |
| `apps/api/internal/platform/queue/` | [KNOWLEDGE_JOBS.md](KNOWLEDGE_JOBS.md), [DEPLOY_CHECKLIST.md](DEPLOY_CHECKLIST.md) |
| `apps/api/cmd/jobworker/` | [KNOWLEDGE_JOBS.md](KNOWLEDGE_JOBS.md), [DEPLOY_CHECKLIST.md](DEPLOY_CHECKLIST.md), [apps:workers.md](apps:workers.md) |

### Retrieval and intelligence (search, ask, QA)

| Code paths (primary) | Documentation to review/update |
|----------------------|--------------------------------|
| `apps/api/internal/retrieval_intelligence/` | [AI_RETRIEVAL_GOVERNANCE.md](AI_RETRIEVAL_GOVERNANCE.md), [retrieval-ai-foundation.md](retrieval-ai-foundation.md), [SEARCH_AND_QA_UX.md](SEARCH_AND_QA_UX.md) |
| `apps/api/internal/modules/retrieval_intelligence/` | Same + [API_SURFACE_V1.md](API_SURFACE_V1.md) |
| `apps/api/internal/search/`, `apps/api/internal/retrieval/`, `apps/api/internal/embeddings/` | [AI_RETRIEVAL_GOVERNANCE.md](AI_RETRIEVAL_GOVERNANCE.md), [API_SURFACE_V1.md](API_SURFACE_V1.md), [permission-system.md](permission-system.md) (scoped retrieval) |
| `apps/api/internal/qa/` | [AI_RETRIEVAL_GOVERNANCE.md](AI_RETRIEVAL_GOVERNANCE.md), [SEARCH_AND_QA_UX.md](SEARCH_AND_QA_UX.md) |

### Governance and review

| Code paths (primary) | Documentation to review/update |
|----------------------|--------------------------------|
| `apps/api/internal/governance/` | [AI_RETRIEVAL_GOVERNANCE.md](AI_RETRIEVAL_GOVERNANCE.md), [ADMIN_GUIDE_V1.md](ADMIN_GUIDE_V1.md), [USER_GUIDE_V1.md](USER_GUIDE_V1.md) where product copy depends on behavior |
| `apps/api/internal/modules/governance/` | Same |
| `apps/api/internal/review/` | [AI_RETRIEVAL_GOVERNANCE.md](AI_RETRIEVAL_GOVERNANCE.md), [ADMIN_GUIDE_V1.md](ADMIN_GUIDE_V1.md) |
| HTTP routes under `/governance/*` in `apps/api/internal/httpserver/` | [API_SURFACE_V1.md](API_SURFACE_V1.md), [ADMIN_GUIDE_V1.md](ADMIN_GUIDE_V1.md) |

### Audit and operations

| Code paths (primary) | Documentation to review/update |
|----------------------|--------------------------------|
| `apps/api/internal/audit/` | [OPERATIONS.md](OPERATIONS.md), governance/admin docs if event types are user-visible |
| `apps/api/internal/modules/audit_ops/` | Same |
| `apps/api/internal/ai/privacy/vault_*.go` | [PRODUCTION_HARDENING.md](PRODUCTION_HARDENING.md), [OPERATIONS.md](OPERATIONS.md) (vault.* audit events) |

### L1 cache (v0.4.0+)

| Code paths (primary) | Documentation to review/update |
|----------------------|--------------------------------|
| `apps/api/internal/cache/` | [PRODUCTION_HARDENING.md](PRODUCTION_HARDENING.md) §12, [CONFIG_ENV.md](CONFIG_ENV.md), [`.env.example`](../.env.example) |
| Hot-read handlers (`/domains`, `/users/:id/effective-access`, `/search`, `/knowledge-jobs/engine-metadata`) in `routes_register.go` | [PRODUCTION_HARDENING.md](PRODUCTION_HARDENING.md) §12 (staleness window for `/effective-access`) |
| Invalidation hooks (entity publish, role grant, policy update, feed config patch) | [PRODUCTION_HARDENING.md](PRODUCTION_HARDENING.md) §12 |

### OAuth proxy + MCP (v0.5.0 / v0.5.1)

| Code paths (primary) | Documentation to review/update |
|----------------------|--------------------------------|
| `apps/api/internal/oauth_proxy/` | [adr/0015-oauth-proxy-and-mcp-bridge.md](adr/0015-oauth-proxy-and-mcp-bridge.md), [PRODUCTION_HARDENING.md](PRODUCTION_HARDENING.md) §13, [SECRET_ROTATION.md](SECRET_ROTATION.md) §9–10, [CONFIG_ENV.md](CONFIG_ENV.md), [operations/mcp.md](operations/mcp.md) |
| `apps/api/internal/mcp/` | [adr/0015-oauth-proxy-and-mcp-bridge.md](adr/0015-oauth-proxy-and-mcp-bridge.md), [PRODUCTION_HARDENING.md](PRODUCTION_HARDENING.md) §14, [operations/mcp.md](operations/mcp.md), [CONFIG_ENV.md](CONFIG_ENV.md) |
| `apps/api/internal/db/migrations/000043_oauth_clients.*` | [UPGRADE_AND_ROLLBACK.md](UPGRADE_AND_ROLLBACK.md) §9 |

### entity_summarize knowledge job + kltools CLI (v0.4.0)

| Code paths (primary) | Documentation to review/update |
|----------------------|--------------------------------|
| `apps/api/internal/knowledge_jobs/entity_summarize.go` | [KNOWLEDGE_JOBS.md](KNOWLEDGE_JOBS.md), [operations/kltools.md](operations/kltools.md) |
| `apps/api/internal/ai/prompts/templates/entity_summarize.v1.json` | If contract changes (length, structure), bump to .v2 and update [operations/kltools.md](operations/kltools.md) |
| `apps/api/internal/db/migrations/000042_entity_summarize_projection.*` | [UPGRADE_AND_ROLLBACK.md](UPGRADE_AND_ROLLBACK.md) §9 |
| `apps/api/cmd/kltools/` | [operations/kltools.md](operations/kltools.md), [Dockerfile.api](../Dockerfile.api) (binary copy line) |

### OpenAPI v3 generic connector (v0.6.0)

| Code paths (primary) | Documentation to review/update |
|----------------------|--------------------------------|
| `apps/api/internal/ingestion_connectors/adapters/openapi_v3/` | [adr/0016-openapi-v3-generic-connector.md](adr/0016-openapi-v3-generic-connector.md), [CONNECTOR_CAPABILITY_MATRIX.md](CONNECTOR_CAPABILITY_MATRIX.md), [CONFIG_ENV.md](CONFIG_ENV.md), [`.env.example`](../.env.example) |
| `apps/api/internal/db/migrations/000044_openapi_v3_connector.*` | [UPGRADE_AND_ROLLBACK.md](UPGRADE_AND_ROLLBACK.md) §9, [CONNECTOR_CAPABILITY_MATRIX.md](CONNECTOR_CAPABILITY_MATRIX.md) |

### Manual upload connector (v0.7.0)

| Code paths (primary) | Documentation to review/update |
|----------------------|--------------------------------|
| `apps/api/internal/ingestion_connectors/manual.go`, `manual_extract.go`, `manual_youtube.go` | [operations/manual-upload.md](operations/manual-upload.md), [CONNECTOR_CAPABILITY_MATRIX.md](CONNECTOR_CAPABILITY_MATRIX.md) |
| `apps/api/internal/ingestion_connectors/adapters/manual/` | Same |
| `apps/api/internal/httpserver/manual_routes.go` | [operations/manual-upload.md](operations/manual-upload.md), [API_SURFACE_V1.md](API_SURFACE_V1.md) for new routes table |
| `apps/api/internal/db/migrations/000045_manual_connector.*` | [UPGRADE_AND_ROLLBACK.md](UPGRADE_AND_ROLLBACK.md) §9, [CONNECTOR_CAPABILITY_MATRIX.md](CONNECTOR_CAPABILITY_MATRIX.md) |
| `apps/web/src/app/control-plane/sources/collections/`, `apps/web/src/components/manual-upload/` | [operations/manual-upload.md](operations/manual-upload.md), [INFORMATION_ARCHITECTURE_V1.md](INFORMATION_ARCHITECTURE_V1.md) for the new CP surface |
| `apps/api/cmd/api/main.go` BodyLimit | [operations/manual-upload.md](operations/manual-upload.md) (50 MiB upload cap rationale) |

### Cross-cutting HTTP and config

| Code paths (primary) | Documentation to review/update |
|----------------------|--------------------------------|
| `apps/api/internal/httpserver/routes_register.go`, `routes_*.go` | [API_SURFACE_V1.md](API_SURFACE_V1.md), [ACCESS_MODEL.md](ACCESS_MODEL.md) for new public routes |
| `apps/api/internal/config/` (incl. `hardening.go`, `config.go`, `bootstrap_env.go`) | [CONFIG_ENV.md](CONFIG_ENV.md), [SELF_HOSTED.md](SELF_HOSTED.md), [ACCESS_MODEL.md](ACCESS_MODEL.md), [PRODUCTION_HARDENING.md](PRODUCTION_HARDENING.md), [DEPLOY_CHECKLIST.md](DEPLOY_CHECKLIST.md), [`.env.example`](../.env.example) |
| `apps/api/internal/instancebootstrap/`, `apps/api/cmd/api/main.go` (startup bootstrap) | [CONFIG_ENV.md](CONFIG_ENV.md), [SELF_HOSTED.md](SELF_HOSTED.md), [ACCESS_MODEL.md](ACCESS_MODEL.md) |
| `apps/api/internal/db/migrations/` | [DOMAIN_MODEL.md](DOMAIN_MODEL.md), [DOMAIN_MODEL_CONTRACT.md](DOMAIN_MODEL_CONTRACT.md) (taxonomy / lifecycle narrative), [INITIAL_SCHEMA_OUTLINE.md](INITIAL_SCHEMA_OUTLINE.md) if still used; [RELEASE_READINESS_AUDIT.md](RELEASE_READINESS_AUDIT.md) if migration story affects operators |
| `apps/api/internal/onboarding/`, `apps/api/internal/presetcatalog/` | [onboarding-setup-flow.md](onboarding-setup-flow.md), [preset-catalog.md](preset-catalog.md), [USER_GUIDE_V1.md](USER_GUIDE_V1.md) |

### Builders (roles, scenarios, jobs)

| Code paths (primary) | Documentation to review/update |
|----------------------|--------------------------------|
| `apps/api/internal/role_builder/` | [role-builder.md](role-builder.md), [API_SURFACE_V1.md](API_SURFACE_V1.md), [AI_RETRIEVAL_GOVERNANCE.md](AI_RETRIEVAL_GOVERNANCE.md), [ACCESS_MODEL.md](ACCESS_MODEL.md) (scenario binding checks on Ask) |
| `apps/api/internal/scenario_builder/` | [scenario-builder.md](scenario-builder.md), [API_SURFACE_V1.md](API_SURFACE_V1.md) |
| `apps/api/internal/knowledge_jobs/` (builder-facing APIs) | [job-builder.md](job-builder.md) |

---

## Frontend — Next.js (`apps/web`)

| App / route area under `apps/web/src/app/` | Documentation to review/update |
|--------------------------------------------|--------------------------------|
| `control-plane/*` (setup, sources, feeds, presets) | [control-plane-ui-ia.md](control-plane-ui-ia.md), [INFORMATION_ARCHITECTURE_V1.md](INFORMATION_ARCHITECTURE_V1.md), [SOURCE_FEED_SETUP_FLOW.md](SOURCE_FEED_SETUP_FLOW.md), [onboarding-setup-flow.md](onboarding-setup-flow.md), [preset-catalog.md](preset-catalog.md), [USER_GUIDE_V1.md](USER_GUIDE_V1.md) |
| Builders: roles / scenarios / jobs (control-plane or app routes) | [role-builder.md](role-builder.md), [scenario-builder.md](scenario-builder.md), [job-builder.md](job-builder.md), [USER_GUIDE_V1.md](USER_GUIDE_V1.md) |
| `governance/*`, `control-plane/governance/*`, `app/governance/*` | [ADMIN_GUIDE_V1.md](ADMIN_GUIDE_V1.md), [user-facing-product-surface.md](user-facing-product-surface.md), [INFORMATION_ARCHITECTURE_V1.md](INFORMATION_ARCHITECTURE_V1.md) |
| Search, ask, explorer, project memory surfaces | [SEARCH_AND_QA_UX.md](SEARCH_AND_QA_UX.md), [AI_RETRIEVAL_GOVERNANCE.md](AI_RETRIEVAL_GOVERNANCE.md), [INFORMATION_ARCHITECTURE_PRODUCT_V1.md](INFORMATION_ARCHITECTURE_PRODUCT_V1.md) |
| Digests, decisions, project views | [user-facing-product-surface.md](user-facing-product-surface.md), [INFORMATION_ARCHITECTURE_PRODUCT_V1.md](INFORMATION_ARCHITECTURE_PRODUCT_V1.md), [KNOWLEDGE_JOBS.md](KNOWLEDGE_JOBS.md) if digest behavior changes |
| `(dash)/` landing / navigation | [INFORMATION_ARCHITECTURE_V1.md](INFORMATION_ARCHITECTURE_V1.md), [ADMIN_UI_V1.md](ADMIN_UI_V1.md) |
| `apps/web/next.config.ts`, `apps/web/src/middleware.ts` (redirects / rewrites; `/app/*` → dash; `/admin/*` → `/control-plane/*`; CP → dash rewrites) | [INFORMATION_ARCHITECTURE_V1.md](INFORMATION_ARCHITECTURE_V1.md), [ADMIN_UI_CONSOLIDATION_PLAN.md](ADMIN_UI_CONSOLIDATION_PLAN.md), [EXTERNAL_DEV_QUICKSTART.md](EXTERNAL_DEV_QUICKSTART.md) |

### In-product guidance (copy)

| Code pattern | Action |
|--------------|--------|
| `apps/web/src/lib/docConcepts.ts`, `docsLinks.ts`, `components/guidance/*` | Keep slugs aligned with [GLOSSARY.md](GLOSSARY.md) and the doc file each slug points at; set `NEXT_PUBLIC_DOCS_BASE_URL` per [CONFIG_ENV.md](CONFIG_ENV.md) / [`.env.example`](../.env.example) |
| `apps/web/src/**/*.tsx` — empty states, `title`, `description`, tooltips, wizard steps | Update the UX/admin/user docs above for the matching surface; keep tone consistent with [USER_GUIDE_V1.md](USER_GUIDE_V1.md) / [ADMIN_GUIDE_V1.md](ADMIN_GUIDE_V1.md) |

---

## Deployment and runtime

| Code / config paths | Documentation to review/update |
|---------------------|--------------------------------|
| `docker-compose.yml`, `Dockerfile`, `Dockerfile.web` | [DEPLOY_CHECKLIST.md](DEPLOY_CHECKLIST.md), [SELF_HOSTED.md](SELF_HOSTED.md), [STAGING_SMOKE_TEST.md](STAGING_SMOKE_TEST.md) |
| `.github/workflows/*.yml` | [DEPLOY_CHECKLIST.md](DEPLOY_CHECKLIST.md), [REPO_STRUCTURE.md](REPO_STRUCTURE.md) if contributor-facing |
| `.env.example` | [CONFIG_ENV.md](CONFIG_ENV.md), [PRODUCTION_HARDENING.md](PRODUCTION_HARDENING.md), [DEPLOY_CHECKLIST.md](DEPLOY_CHECKLIST.md) |
| `apps/api/cmd/api/main.go` (startup, migrate, listen) | [DEPLOY_CHECKLIST.md](DEPLOY_CHECKLIST.md), [OPERATIONS.md](OPERATIONS.md) |
| Health/ops routes (`apps/api/internal/httpserver/routes_health.go`) | [ACCESS_MODEL.md](ACCESS_MODEL.md), [PRODUCTION_HARDENING.md](PRODUCTION_HARDENING.md), [PRODUCTION_GO_LIVE_CHECKLIST.md](PRODUCTION_GO_LIVE_CHECKLIST.md) |
| Cross-cutting stubs / capability matrix (operators) | [LIMITATIONS.md](LIMITATIONS.md), [SELF_HOSTED.md](SELF_HOSTED.md), [RELEASE_READINESS_AUDIT.md](RELEASE_READINESS_AUDIT.md); periodic readiness refresh may cross-link [ADMIN_UI_CONSOLIDATION_PLAN.md](ADMIN_UI_CONSOLIDATION_PLAN.md) / [INFORMATION_ARCHITECTURE_V1.md](INFORMATION_ARCHITECTURE_V1.md) when web shells or admin URLs shift |

---

## Shared concepts and examples

| Change type | Documentation to review/update |
|-------------|--------------------------------|
| New or renamed domain term | [DOMAIN_MODEL.md](DOMAIN_MODEL.md), [USER_GUIDE_V1.md](USER_GUIDE_V1.md); add [GLOSSARY.md](GLOSSARY.md) if many terms |
| User journey or persona | [USER_GUIDE_V1.md](USER_GUIDE_V1.md), [ADMIN_GUIDE_V1.md](ADMIN_GUIDE_V1.md) |
| Architecture or module seams | [ARCHITECTURE.md](ARCHITECTURE.md), [MODULE_BOUNDARIES.md](MODULE_BOUNDARIES.md), [adr/0011-internal-modules-pilot-scope.md](adr/0011-internal-modules-pilot-scope.md), [backend-architecture.md](backend-architecture.md), [apps:api.md](apps:api.md) |
| Durable decision | New ADR in [adr/](adr/) |
| API examples / curl | [API_SURFACE_V1.md](API_SURFACE_V1.md), [INGESTION_API.md](INGESTION_API.md) as appropriate |

---

## Quick reference: required backend modules (from policy)

| Module area | Top-level code | Primary docs |
|-------------|----------------|--------------|
| identity_access | `identity_access/`, `modules/identity_access/` | ACCESS_MODEL, permission-*, DOMAIN_MODEL |
| knowledge_core | `knowledge_core/`, `modules/knowledge_core/` | DOMAIN_MODEL, API_SURFACE_V1 |
| ingestion_connectors | `ingestion_connectors/`, `modules/ingestion_connectors/` | INGESTION_*, family docs, SOURCE_FEED_SETUP_FLOW |
| knowledge_jobs | `knowledge_jobs/`, `modules/knowledge_jobs/`, `cmd/jobworker` | KNOWLEDGE_JOBS, knowledge-jobs-engine, job-builder |
| retrieval_intelligence | `retrieval_intelligence/`, search, retrieval, embeddings, qa | AI_RETRIEVAL_GOVERNANCE, SEARCH_AND_QA_UX, retrieval-ai-foundation |
| governance | `governance/`, `review/`, `modules/governance/` | AI_RETRIEVAL_GOVERNANCE, ADMIN_GUIDE_V1 |
| audit_ops | `audit/`, `modules/audit_ops/` | OPERATIONS, RELEASE_READINESS_AUDIT if needed |
