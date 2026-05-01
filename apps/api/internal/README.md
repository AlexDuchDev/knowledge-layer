# `apps/api/internal/` — module map

This directory is the **modular monolith**: ~45 packages organized by bounded context. The public HTTP surface is mounted from `cmd/api`; background work runs from `cmd/jobworker` and `cmd/connectorworker`. Both binaries import packages from here, never the reverse.

If you're new to the codebase, **read this file first**, then jump to [docs/MODULE_BOUNDARIES.md](../../../docs/MODULE_BOUNDARIES.md) and [docs/backend-architecture.md](../../../docs/backend-architecture.md) for the canonical contract.

## Layering rules (no cycles)

```
httpserver  →  app  →  domain modules  →  platform/  →  shared/
```

- **Transport** (`httpserver/`, `httpcontext/`, `auth/`) — Fiber routes, session middleware, principal extraction.
- **Composition root** (`app/deps.go`) — wires every domain service exactly once, returns a `*Deps` consumed by the binaries.
- **Domain modules** — bounded contexts (one folder = one context); they may depend on `platform/` and other domain modules listed in their import block, never on `httpserver/` or `app/`.
- **Platform** (`platform/`) — cross-cutting infra: queue publisher, permissions resolver, metrics registry.
- **Shared** (`shared/contracts/`) — minimal cross-module type declarations; keep this small.

## Identity, access, and audit

| Module | What it owns |
|---|---|
| [`identity_access/`](identity_access/) | Users, teams, roles, domains, grants, **9-step `AccessEvaluator.Evaluate`**. Entry point for every "may principal X do Y to Z?" question; never re-implement access checks elsewhere. |
| [`auth/`](auth/) | Session cookies, login/logout, password hashing. |
| [`httpcontext/`](httpcontext/) | `RequirePrincipal(c)` — the only sanctioned way to extract the caller's UUID inside a handler. |
| [`audit/`](audit/) | Append-only `audit_events` writer. Use for any sensitive operation (vault decrypt, entity publish, scenario launch, …). |
| [`modules/audit_ops/`](modules/audit_ops/) | Newer hex-architecture slice over the audit table; coexists with `audit/` during the gradual migration. |

## Knowledge core, retrieval, AI

| Module | What it owns |
|---|---|
| [`knowledge_core/`](knowledge_core/) | Canonical entities, versions, links, provenance. The `EntityRepo` is the only writer to the `entities` table. |
| [`chunks/`](chunks/) | Entity → chunks lifecycle (split on persist, embed asynchronously). |
| [`embeddings/`](embeddings/) | Vector storage + permission-filtered nearest-neighbour. |
| [`search/`](search/) | OpenSearch keyword index + trust ranking. Permission-filtered via `platform/permissions`. |
| [`retrieval/`](retrieval/) | Hybrid retrieval — composes `search` + `embeddings` + LLM client. |
| [`retrieval_intelligence/`](retrieval_intelligence/) | Façade for `AskGlobal`, `AskEntity`, `GraphExplore`. Owns `buildCanView` — **all retrieval paths build their own `canView` callback here**, callers never pass one in (Phase 1.1.4 hardening). |
| [`qa/`](qa/) | LLM synthesis from context, wraps every call through the AI privacy gateway. |
| [`graphrag/`](graphrag/) | Optional Neo4j graph store (entity/chunk co-mention). Feature-flagged via `NEO4J_URL`. |
| [`ai/privacy/`](ai/privacy/) | Sanitize → vault → call → rehydrate pipeline. Vault uses AES-256 (`AI_PRIVACY_VAULT_KEY`); production fail-closed (Phase 1.1.2). All vault ops emit `vault.*` audit events (Phase 1.1.3). `InvokeInput.PromptTemplateID` is recorded in the privacy trace so audits can pin which template version produced an answer (Phase 4.1.1). |
| [`ai/prompts/`](ai/prompts/) | Versioned LLM prompt registry — embed-loaded JSON templates under `templates/<id>.json`. Each template has a stable id (`ask_global_qa.v1`) used in audit trails; bump the suffix when behaviour changes, never repurpose an existing version (Phase 4.1.1). |
| [`llm/`](llm/) | OpenAI / OpenRouter client. **Do not call directly from outside `ai/privacy/` and `qa/` — go through the privacy gateway**. Enforced by the `depguard` rule in [`apps/api/.golangci.yml`](../.golangci.yml) (Phase 4.1.2). |
| [`answertrace/`](answertrace/) | Per-Ask trace persistence (LLM calls, chunks retrieved, redaction summary). |

## Ingestion & connectors

| Module | What it owns |
|---|---|
| [`ingestion_connectors/`](ingestion_connectors/) | The connector framework: 18 adapters (Slack, Telegram, Notion, Jira, …), source feeds, raw artifacts, normalised records, sync orchestration, webhook handler contract. Read [`adapter.go`](ingestion_connectors/adapter.go) before adding a connector. |
| [`ingestion_connectors/adapters/`](ingestion_connectors/adapters/) | One sub-package per connector. Slack is the reference for both polling sync and webhook push (Phase 2.2.3). |
| [`ingestion_connectors/families/`](ingestion_connectors/families/) | Per-family normalisation helpers (chat, docs_wiki, meeting, work_mgmt, crm_support, email, microsoft365). |
| [`blobstore/`](blobstore/) | S3-compatible raw payload storage; `Nop` default for dev. |
| [`connectoroauth/`](connectoroauth/) | OAuth2 token refresh for Gmail / Microsoft Graph. |

## Jobs

| Module | What it owns |
|---|---|
| [`knowledge_jobs/`](knowledge_jobs/) | Job definitions, runs, outputs. The orchestrator dispatches to typed processors (`weekly_digest`, `decision_extraction`, `planning_summary`, `stale_scan`, `support_trends_extraction`); other types fail-closed. |
| [`jobqueue/`](jobqueue/) | Run orchestration plumbing distinct from `platform/queue` (which is the Asynq publisher). |

## Governance & review

| Module | What it owns |
|---|---|
| [`governance/`](governance/) | Approval queue, owner assignments, policy exceptions, stale-content scan, answer feedback. |
| [`review/`](review/) | Review tasks for analyst sign-off on AI-generated entities. |

## Control-plane builders

| Module | What it owns |
|---|---|
| [`role_builder/`](role_builder/) | Role definitions and assignments. |
| [`scenario_builder/`](scenario_builder/) | Scenarios + bindings (roles ↔ sources ↔ jobs). |
| [`presetcatalog/`](presetcatalog/) | Curated role / scenario / job templates; instantiation = governed clone. |
| [`onboarding/`](onboarding/) | Setup wizard sessions: pick template → connector toggles → assignment → preview → launch. |

## Product surfaces (read-side)

| Module | What it owns |
|---|---|
| [`home/`](home/) | Home feed (recent activity, follows). |
| [`surfacing/`](surfacing/) | Followed entity surfacing. |
| [`recommendations/`](recommendations/) | "Suggested reads" used by browse and entity detail. |
| [`contenthub/`](contenthub/) | Topic hubs (curated entity collections). |
| [`contentblocks/`](contentblocks/) | Entity body block storage. |
| [`extracted_meeting_tasks/`](extracted_meeting_tasks/) | Meeting → task extraction queue (used by Second Brain). |

## Optional modules (feature-flag-gated)

| Module | What it owns | Enable flag |
|---|---|---|
| [`secondbrain/`](secondbrain/) | Pre-meeting briefs, Telegram/Mattermost outbound delivery. ADR-0012. | `SECOND_BRAIN_PREBRIEF_TICK=1`, `TELEGRAM_BOT_TOKEN`, `MATTERMOST_OUTGOING_WEBHOOK_TOKEN` |
| [`graphrag/`](graphrag/) | Neo4j co-mention graph and "Explore from here" expansion. | `NEO4J_URL` |

## Infrastructure & cross-cutting

| Module | What it owns |
|---|---|
| [`app/`](app/) | Composition root (`deps.go`) — wires every service in `NewDeps(pool, cfg)`. |
| [`config/`](config/) | Env loading + production hardening (`ValidateAPI`, `ValidateWorker`). |
| [`db/`](db/) | Pool, migrations, helpers. SQL migrations live in `db/migrations/`. |
| [`httpserver/`](httpserver/) | Routes, middleware, ops endpoints, metrics export. |
| [`platform/queue/`](platform/queue/) | Redis Asynq publisher and task type constants. |
| [`platform/permissions/`](platform/permissions/) | Wraps `identity_access.AccessEvaluator` for retrieval/search. |
| [`platform/metrics/`](platform/metrics/) | Shared Prometheus registry (Phase 2.2.2): `knowledge_job_run_duration_seconds`, `connector_sync_duration_seconds`, `postgres_pool_*`, `asynq_queue_*`. |
| [`workerhealth/`](workerhealth/) | Tiny HTTP server attached to each worker binary for `/health` + `/ops/health` (queue depth + last-processed timestamps). |
| [`opensearch/`](opensearch/) | OpenSearch client + health probe. |
| [`instancebootstrap/`](instancebootstrap/) | First-run domain creation; `/bootstrap` route + auto-bootstrap from env. |
| [`integration/`](integration/) | E2E integration tests gated by `E2E_DB=1` + `DATABASE_URL`. |
| [`cache/`](cache/) | L1 BigCache + invalidator (v0.4.0). Wraps `eko/gocache/v4`. Off by default; enabled via `CACHE_L1_ENABLED=true`. Principal-scoped keys; event-driven flush. See [package README](cache/README.md). |
| [`oauth_proxy/`](oauth_proxy/) | OAuth 2.1 authorization-server proxy fronting an operator-supplied OIDC issuer (v0.5.0). Off by default. Mounts `/.well-known` + `/oauth/{authorize,token,register,callback}`. See [ADR-0015](../../../docs/adr/0015-oauth-proxy-and-mcp-bridge.md), [package README](oauth_proxy/README.md). |
| [`mcp/`](mcp/) | Model Context Protocol endpoint mounted at `/mcp` (v0.5.1). Tools wrapped through `withAccessGuard` (mandatory contract enforced by `TestNew_allToolsAccessGuarded`). Bearer middleware verifies via `oauth_proxy.Server.VerifyBearer`. See [package README](mcp/README.md). |

## Where to add new code

| If you're adding… | Land it in… |
|---|---|
| A new HTTP route | `httpserver/routes_*.go` (group by surface). Mount in `routes_register.go`. |
| A new connector | A new sub-package under `ingestion_connectors/adapters/`. Implement `ConnectorAdapter`; opt into webhooks by also implementing `WebhookHandler`. Register in `app/deps.go`'s `NewRegistry(...)`. |
| A new knowledge job processor | A new method on `DigestRunner` (or a new orchestrator service) + entry in `knowledge_jobs/processor_capabilities.go` + switch in `orchestrator.go:executeProcessor`. |
| A new AI-using flow | Always go through `ai/privacy.PrivacyGateway.InvokeOpenAI`. Adding a prompt? Drop a JSON file under `internal/ai/prompts/templates/<id>.<version>.json` and reference it via `prompts.Get(id)` — set `InvokeInput.PromptTemplateID` to the id so the trace records it. |
| A new audit event type | `audit.WriteInput{ EventType: "your.event_type", … }`. Register in [docs/OPERATIONS.md](../../../docs/OPERATIONS.md) audit-events table. |
| A new metric | `internal/platform/metrics/metrics.go` — register a collector or call an existing recorder (`ObserveJobRun`, …). Don't create per-package private registries. |
| A new MCP tool | `internal/mcp/server.go` — construct via `newGuardedTool(name, desc, action, resourceType, schema, fn)` then append to the slice in `New(deps)`. **Never** call `srv.AddTool` directly from an unwrapped handler — `TestNew_allToolsAccessGuarded` catches this in CI. |
| A new cached read endpoint | Add a typed key constructor in `internal/cache/keys.go` (must embed principal if per-user). Read-through `d.Cache.Get` / `Set` in the handler; identify the invalidation events that should drop the new key and add them to `internal/cache/invalidation.go` (typed methods only). |
| A new OAuth proxy endpoint | `internal/oauth_proxy/server.go` — register in `Mount`. RFC-required public endpoints add to `PublicPaths()` for the routing allow-list. Don't loosen redirect_uri matching. |
| A new operator CLI subcommand | New file under `apps/api/cmd/kltools/` next to `summarize.go` / `reindex.go` / `schema_info.go`. Dispatch from `main.go`. Write subcommands MUST default to dry-run (require `--yes`). Pool is capped at 4 connections. Update [docs/operations/kltools.md](../../../docs/operations/kltools.md). |

## Things to avoid

- **Calling `llm.NewOpenAIFromEnv()` from anywhere except `ai/privacy/` and `qa/`** — the privacy gateway exists for a reason.
- **Re-implementing access evaluation** — always go through `identity_access.AccessEvaluator` or the `permissions.Resolver` wrapper.
- **Cross-module imports between domain modules** when they don't share a real seam — prefer adding the data to a request DTO in `app/` than coupling two contexts.
- **Adding a placeholder package with only `doc.go`** — Phase 1.2.1 deleted the previous three (`platform_operations`, `workflow_governance`, `events`) for exactly this reason.
- **Logging cleartext from sanitized LLM inputs / vault contents** — the privacy gateway is the only place those values are decrypted.

## Tests

- **Unit tests** live alongside the code (`*_test.go`).
- **Integration tests** in `integration/` need Postgres + `E2E_DB=1`.
- **Smoke** scripts in `scripts/smoke-local.sh` (header auth) and `scripts/smoke-session.sh` (session auth) run after `make db-up` + API.
- CI runs Go tests with a Postgres service container; see `.github/workflows/ci.yml`.
