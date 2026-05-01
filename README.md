# Organizational Memory & Knowledge Operations Platform

If your checkout has an **outer** folder that only contains `Knowledge Layer Local/` and a compose wrapper, start here — this directory is the **canonical monorepo root** for code and docs.

A governed platform that turns fragmented company knowledge from chats, meetings, documents, and operational systems into structured, traceable, permission-aware organizational memory.

This repository is **not** a generic AI chatbot over company data, a document dump, or a collaboration replacement. It is organizational memory infrastructure: controlled ingestion, governed knowledge objects, permission-aware retrieval, and AI-assisted synthesis with auditability.

Unlike generic AI assistants over company docs, Knowledge Layer enforces **permission-aware retrieval**, **governed publication**, and **full audit trails by design** — not by post-hoc filter — so a user only ever sees what their role permits, every published fact carries provenance, and every access leaves a record an auditor can read.

## Who it is for

- **Evaluators** deciding whether to adopt or fork a governed memory stack
- **Contributors** extending connectors, jobs, or the control plane
- **Developers** integrating APIs and permission-safe retrieval
- **Operators** deploying and hardening self-hosted instances
- **Admins** configuring roles, scenarios, sources, and governance queues
- **End users** asking, searching, and exploring memory within policy

## Key concepts

**Connectors** are plugin types; **source feeds** are your governed instances (domain, sensitivity, sync). **Canonical entities** are durable knowledge objects—not just files. **Roles**, **scenarios**, and **jobs** structure access and automation; **presets** and **setup sessions** accelerate first run. **Governance** (review, approval) gates high-risk outputs. See [docs/GLOSSARY.md](docs/GLOSSARY.md) and [docs/PRODUCT_CONCEPTS.md](docs/PRODUCT_CONCEPTS.md).

## Main capabilities

- Connector-based **ingestion** with raw preservation and normalization paths
- **Permission-aware** retrieval and Ask/Search (no “retrieve everything then filter”)
- **Knowledge jobs** engine for digests, sync helpers, and scheduled work
- **Control plane** for roles, scenarios, jobs, presets, and onboarding
- **Auditability** and operational hooks for production use
- **MCP endpoint** (v0.5.1+, opt-in) — Claude Desktop / Cursor / IDE plugins consume KL tools after OIDC auth via the built-in OAuth 2.1 proxy. Every tool call routes through `AccessEvaluator`. See [docs/operations/mcp.md](docs/operations/mcp.md).
- **L1 in-process cache** (v0.4.0+, opt-in) for hot reads — `/domains`, `/users/:id/effective-access`, `/search`, `/knowledge-jobs/engine-metadata`. Principal-scoped keys, event-driven invalidation. See [docs/PRODUCTION_HARDENING.md §12](docs/PRODUCTION_HARDENING.md).
- **`kltools` CLI** (v0.4.0+) shipping inside the API image — `summarize`, `reindex`, `schema-info`. See [docs/operations/kltools.md](docs/operations/kltools.md).
- **Generic OpenAPI v3 connector** (v0.6.0+) — add REST-API source feeds via configuration instead of per-vendor Go code. See [docs/adr/0016-openapi-v3-generic-connector.md](docs/adr/0016-openapi-v3-generic-connector.md).

## High-level architecture

Modular monolith **API** (`apps/api`) + **Next.js** web (`apps/web`) + **workers** (job/connector processes). Postgres holds canonical data; Redis backs queues. Deep dive: [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md), [docs/BACKEND_ARCHITECTURE.md](docs/BACKEND_ARCHITECTURE.md).

## Current status

The codebase is **actively evolving**: core flows, control-plane UI, and deployment packs land incrementally. Treat production paths as **follow the deployment and hardening docs** rather than “batteries included SaaS.” See [docs/OSS_V1_SCOPE.md](docs/OSS_V1_SCOPE.md) for a single-page OSS contract, then [docs/DEPLOY_CHECKLIST.md](docs/DEPLOY_CHECKLIST.md) and [docs/PRODUCTION_HARDENING.md](docs/PRODUCTION_HARDENING.md).

## Open source and deployment model

> **Self-hosted, single-tenant.** This project is open source ([LICENSE](LICENSE), [NOTICE](NOTICE)) and designed for **one organization per instance** — own database, own API, own web app. It is **NOT a multi-tenant SaaS** where unrelated companies share one deployment. Read [docs/OSS_V1_SCOPE.md](docs/OSS_V1_SCOPE.md) (release contract) and [docs/LIMITATIONS.md](docs/LIMITATIONS.md) (known stubs and optional modules) before evaluating.

API stability follows [docs/API_STABILITY.md](docs/API_STABILITY.md): v0.x allows breaking changes between minor releases; v1.0 will introduce the semver contract and `/v1/...` versioned endpoints.

- Production setup: [docs/SELF_HOSTED.md](docs/SELF_HOSTED.md)
- Day-2 operations: [docs/OPERATIONS.md](docs/OPERATIONS.md)
- Contributing: [CONTRIBUTING.md](CONTRIBUTING.md) (documentation must stay aligned with code — see [docs/DOCS_MAINTENANCE_POLICY.md](docs/DOCS_MAINTENANCE_POLICY.md))
- Security reports: [SECURITY.md](SECURITY.md)
- Community: [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)
- Releases and tagging: [docs/RELEASING.md](docs/RELEASING.md)
- API stability: [docs/API_STABILITY.md](docs/API_STABILITY.md)
- Optional modules (feature-flag-gated): see "Optional modules" in [docs/OSS_V1_SCOPE.md](docs/OSS_V1_SCOPE.md)
- Pre-push sanity (tracked secrets): [`scripts/repo-sanity-check.sh`](scripts/repo-sanity-check.sh)

## Repository layout

See [docs/REPO_STRUCTURE.md](docs/REPO_STRUCTURE.md). Top-level:

- `apps/web` — Next.js UI: **canonical product routes** at the root (`/search`, `/ask`, `/governance`, `/entities`, …). **Administration and builders** use canonical URLs under **`/control-plane/*`** (legacy `/admin/*` redirects there; some CP paths rewrite to dash implementations — see [docs/ADMIN_UI_CONSOLIDATION_PLAN.md](docs/ADMIN_UI_CONSOLIDATION_PLAN.md)). Deprecated **`/app/*`** mirrors redirect to root equivalents (see [docs/INFORMATION_ARCHITECTURE_V1.md](docs/INFORMATION_ARCHITECTURE_V1.md)). Known stubs: [docs/LIMITATIONS.md](docs/LIMITATIONS.md).
- `apps/api` — Go (Fiber) modular monolith API
- `apps/workers` — notes for the background worker (binary: `apps/api/cmd/jobworker`)
- `packages/shared` — shared TypeScript constants/types
- `docs/` — product and architecture documentation, `docs/adr/` (index: [docs/README.md](docs/README.md); maintenance: [docs/DOCS_MAINTENANCE_POLICY.md](docs/DOCS_MAINTENANCE_POLICY.md))

## Prerequisites

- Docker (for Postgres + Redis)
- Go 1.22+
- Node 20+ (for web and `packages/shared`)

## Quick start (local)

1. Copy environment template:

   ```bash
   cp .env.example .env
   ```

2. Start databases:

   ```bash
   make db-up
   ```

3. Run the API (applies migrations on startup):

   ```bash
   cd apps/api && go run ./cmd/api
   ```

4. Run the web app (separate terminal):

   ```bash
   cd apps/web && npm install && npm run dev
   ```

   Optional env (defaults: API `http://localhost:8080`, dev principal header on):

   - `NEXT_PUBLIC_API_URL`
   - `NEXT_PUBLIC_USE_DEV_HEADER` — set `true` for local pilot (`X-Principal-User-ID`); `false` for session login against `AUTH_MODE=session`
   - `NEXT_PUBLIC_PRINCIPAL_USER_ID` — seed admin UUID when dev header is on

5. (Optional) Run background workers when `REDIS_URL` is set:

   ```bash
   cd apps/api && go run ./cmd/jobworker
   cd apps/api && go run ./cmd/connectorworker
   ```

   `jobworker` handles `knowledge:job_run`. `connectorworker` consumes the **ingestion artifact queue** (`ingestion:process_artifact`) and runs **typed normalizers** for supported raw artifact kinds (chat, docs/files, calendar/meeting families — see [docs/LIMITATIONS.md](docs/LIMITATIONS.md), [docs/CONNECTOR_CAPABILITY_MATRIX.md](docs/CONNECTOR_CAPABILITY_MATRIX.md), and [docs/connector-framework.md](docs/connector-framework.md)); other artifact types no-op until a normalizer exists. For worker layout and task names, see [docs/backend-architecture.md](docs/backend-architecture.md).

The API listens on `http://localhost:8080` by default when running via Go (`API_PORT`). Health check: `GET /health`.

If you run the full stack via this repo’s `docker compose`, the **published host ports** default to low-collision values (see `docker-compose.yml`): API **`http://localhost:18080`**, web **`http://localhost:13000`**, Postgres **`localhost:25432`**, Redis **`localhost:16379`**, OpenSearch **`localhost:19200`**.

## Local golden path (what to try first)

After API + web are running:

1. **Bootstrap** if needed — open `http://localhost:3000/bootstrap` when the instance has no domain yet (or use API env auto-bootstrap).
   - If you’re using `docker compose` defaults from this repo, use `http://localhost:13000/bootstrap` instead.
2. **Home** — `http://localhost:3000/` — feed and honest “what works locally” summary.
   - Compose defaults: `http://localhost:13000/`
3. **Search** — `http://localhost:3000/search` — open an entity from results (OpenSearch from compose must be running).
   - Compose defaults: `http://localhost:13000/search`
4. **Ask** — `http://localhost:3000/ask` — cited answers over permitted content.
   - Compose defaults: `http://localhost:13000/ask`

Operators use **Control plane** in the sidebar (`/control-plane/governance` entry) for sources, jobs, presets, and setup. Supported local runtime now includes queued normalization for chat/docs/meeting families plus runnable jobs for `weekly_digest`, `decision_extraction`, `planning_summary`, `stale_scan`, and `support_trends_extraction`. Remaining partial behavior is listed in [docs/LIMITATIONS.md](docs/LIMITATIONS.md).

## Authentication (local vs production)

**Local pilot (default `.env.example`):** set `NEXT_PUBLIC_USE_DEV_HEADER=true` and send:

```http
X-Principal-User-ID: <uuid>
```

Seed data (after migrations) includes users such as `30000000-0000-0000-0000-000000000001` (admin) — see migration `000008_dev_seed`.

**Production-style:** API `AUTH_MODE=session`, `SESSION_SECRET` set, web `NEXT_PUBLIC_USE_DEV_HEADER=false`, sign in at `/login` after bootstrap or invitation. See [docs/SELF_HOSTED.md](docs/SELF_HOSTED.md).

## Makefile

| Target      | Description                                      |
|------------|--------------------------------------------------|
| `make db-up`   | `docker compose up -d postgres redis`        |
| `make dev`     | Prints commands to run API / web / worker    |
| `make test`    | Go tests in `apps/api` + build `cmd/jobworker` + `cmd/connectorworker` |
| `make lint`    | `go vet` + web ESLint + shared typecheck    |
| `make typecheck` | Next.js + `packages/shared` TypeScript    |

## Integration / E2E digest test

With a migrated database that includes seed data:

```bash
# If you use the repo's docker-compose Postgres publish mapping, use the host port from docker-compose.yml
# (defaults to 25432 to avoid collisions with a local Postgres on :5432).
export DATABASE_URL=postgres://knowledge:knowledge@localhost:25432/knowledge?sslmode=disable
export E2E_DB=1
cd apps/api && go test ./internal/integration/... -count=1 -v
```

Coverage includes entity access scope, digest flow, **governance publish gate**, and **search relation expansion (no cross-domain leak)**.

## CI

GitHub Actions workflow `.github/workflows/ci.yml` runs Go tests (with Postgres service), then web lint and typecheck.

## Documentation

**Hub:** [docs/README.md](docs/README.md) — audience paths (evaluators, operators, contributors, product learners) and full index.

| Topic | Doc |
|-------|-----|
| Product concepts | [docs/PRODUCT_CONCEPTS.md](docs/PRODUCT_CONCEPTS.md) |
| Glossary | [docs/GLOSSARY.md](docs/GLOSSARY.md) |
| External dev quickstart (canonical URLs) | [docs/EXTERNAL_DEV_QUICKSTART.md](docs/EXTERNAL_DEV_QUICKSTART.md) |
| Known limitations / stubs | [docs/LIMITATIONS.md](docs/LIMITATIONS.md) |
| Connector sync vs normalization | [docs/CONNECTOR_CAPABILITY_MATRIX.md](docs/CONNECTOR_CAPABILITY_MATRIX.md) |
| Getting started | [docs/GETTING_STARTED.md](docs/GETTING_STARTED.md) |
| Architecture (open source overview) | [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) |
| Control plane | [docs/CONTROL_PLANE_OVERVIEW.md](docs/CONTROL_PLANE_OVERVIEW.md) |
| User guide | [docs/USER_GUIDE.md](docs/USER_GUIDE.md) |
| Deployment | [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md), [docs/SELF_HOSTED.md](docs/SELF_HOSTED.md) |
| Contributing | [CONTRIBUTING.md](CONTRIBUTING.md), [docs/CONTRIBUTING.md](docs/CONTRIBUTING.md) |
| Doc maintenance | [docs/DOCS_MAINTENANCE_POLICY.md](docs/DOCS_MAINTENANCE_POLICY.md) |

**In-app help:** set `NEXT_PUBLIC_DOCS_BASE_URL` in `.env` to your repo’s GitHub `.../blob/<branch>/docs` path so UI callouts link to these files (see [.env.example](.env.example)).

Additional references: [docs/ARCHITECTURE_HOST.md](docs/ARCHITECTURE_HOST.md), [docs/IMPLEMENTATION_PLAN.md](docs/IMPLEMENTATION_PLAN.md), [docs/USER_GUIDE_V1.md](docs/USER_GUIDE_V1.md), [docs/ADMIN_GUIDE_V1.md](docs/ADMIN_GUIDE_V1.md), [AGENTS.md](AGENTS.md).
