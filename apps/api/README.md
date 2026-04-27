# Knowledge Layer API (Go)

Fiber **modular monolith**: identity & access, knowledge core, ingestion + connectors, knowledge jobs, retrieval/AI, governance, audit. Three binaries; one Postgres; Redis (queue) and OpenSearch (search) optional in local dev.

## Quick start (local)

```bash
# 1. From the repo root, bring up Postgres + Redis + OpenSearch
make db-up                       # docker compose up -d postgres redis opensearch

# 2. Run the API (applies migrations on first startup)
cd apps/api
go run ./cmd/api                 # listens on :8080 by default

# 3. (Optional) Run the workers in separate terminals
go run ./cmd/jobworker
go run ./cmd/connectorworker

# 4. Smoke
curl -s http://localhost:8080/health
```

The API auto-bootstraps a default domain in `APP_ENV=local`. For staging/production, set `AUTO_BOOTSTRAP_INSTANCE=1` plus `BOOTSTRAP_ADMIN_EMAIL` / `BOOTSTRAP_ADMIN_PASSWORD` (see [docs/SELF_HOSTED.md](../../docs/SELF_HOSTED.md)).

## Binaries

| Command | Role | Default health port |
|---|---|---|
| `go run ./cmd/api` | HTTP API. Applies migrations on startup. | `:8080` (`API_PORT`) |
| `go run ./cmd/jobworker` | Asynq consumer for `knowledge:job_run`, `knowledge:scheduled_tick`, `secondbrain:outbound_delivery`. | `:9001` (`JOBWORKER_HEALTH_PORT`) |
| `go run ./cmd/connectorworker` | Asynq consumer for `connector:source_sync`, `ingestion:process_artifact`, `retrieval:embed_chunk`, `graphrag:extract_entity`. | `:9002` (`CONNECTORWORKER_HEALTH_PORT`) |

## Layout

```
apps/api/
  cmd/                   # binary entrypoints (api, jobworker, connectorworker)
  internal/
    app/                 # composition root (deps.go) — wires every service
    httpserver/          # Fiber routes, middleware, ops endpoints
    config/              # env loading + ValidateAPI / ValidateWorker
    db/migrations/       # SQL migrations (pgvector)
    platform/            # cross-cutting infra (queue, permissions, metrics)
    shared/              # tiny cross-module type declarations
    workerhealth/        # /health + /ops/health for the worker binaries
    <domain modules>/    # identity_access, knowledge_core, retrieval_intelligence, …
  prompts/               # static LLM prompt templates (will move under internal/ai/prompts/ in Phase 4)
```

For the full per-package map see [`internal/README.md`](internal/README.md).

## Make targets (from repo root)

| Target | What it runs |
|---|---|
| `make db-up` | `docker compose up -d postgres redis opensearch` |
| `make test` | `go test ./...` + builds the worker binaries |
| `make lint` | `go vet ./...` + web ESLint + shared TS check |
| `make typecheck` | Web + shared TS check |

## Tests

- **Unit:** alongside the code (`*_test.go`); always passes without env.
- **Integration:** under [`internal/integration/`](internal/integration/). Requires `E2E_DB=1` and a migrated Postgres at `DATABASE_URL` — set them and run `go test ./internal/integration/... -count=1 -v`.
- **Smoke:** scripts at [`../../scripts/smoke-local.sh`](../../scripts/smoke-local.sh) (header auth) and [`../../scripts/smoke-session.sh`](../../scripts/smoke-session.sh) (session auth).

## Flows

1. **Request** → session/auth middleware → route handler → domain service → Postgres / Redis / OpenSearch.
2. **Jobs**: API enqueues `knowledge:job_run` to Redis → `jobworker` runs the orchestrator → typed processor (`weekly_digest`, `decision_extraction`, …).
3. **Connectors**: source feeds + `connectorworker` consume `connector:source_sync` → adapter fetches → raw artifact → normalised record → entity.
4. **Webhooks** (push connectors): `POST /connectors/webhook/:adapter_kind/:source_feed_id` → adapter verifies signature → `Service.IngestWebhookResult` → same dedup as polled syncs.

## Integration points

- **Postgres** — required everywhere (entity store, audit, jobs).
- **Redis** — required in staging/production for queues; optional in local dev (work runs synchronously where supported).
- **OpenSearch** — keyword search; optional in dev (degraded UX without it).
- **Neo4j** — optional GraphRAG module; enable with `NEO4J_URL`.
- **OpenAI / OpenRouter** — required for embeddings + Ask synthesis; routed exclusively through the AI privacy gateway.

## Boundaries

- Do **not** bypass `ConnectorAdapter` for ingestion. New sources land as adapters under `internal/ingestion_connectors/adapters/`.
- Do **not** evaluate permissions in LLM prompts. Use `identity_access.AccessEvaluator` (or `platform/permissions` for retrieval). See [docs/PERMISSION_SYSTEM.md](../../docs/PERMISSION_SYSTEM.md).
- Do **not** call `llm.NewOpenAIFromEnv()` from outside `internal/ai/privacy/` or `internal/qa/` — the privacy gateway is the single sanctioned entrypoint for LLM completions.

## Docs

- Per-package map: [`internal/README.md`](internal/README.md)
- Backend architecture: [docs/BACKEND_ARCHITECTURE.md](../../docs/BACKEND_ARCHITECTURE.md), [docs/backend-architecture.md](../../docs/backend-architecture.md)
- Module boundaries: [docs/MODULE_BOUNDARIES.md](../../docs/MODULE_BOUNDARIES.md)
- Production hardening: [docs/PRODUCTION_HARDENING.md](../../docs/PRODUCTION_HARDENING.md)
- API stability policy: [docs/API_STABILITY.md](../../docs/API_STABILITY.md)
