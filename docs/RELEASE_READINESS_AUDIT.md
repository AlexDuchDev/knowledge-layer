# Release readiness audit — Organizational Memory platform

**Audit date:** 2026-04-12 (evidence); **Doc alignment refresh:** 2026-04-20 (migrations, connector worker scope, Postgres baseline — no full re-audit).  
**Method:** Repository evidence (code, migrations, routes, workers, tests, compose). Docs used only as secondary cross-check.  
**Scope:** Post-remediation snapshot after P0/P1 fixes listed in section 5. **2026-04-12 refresh:** comprehensive system analysis pass; web admin URLs converged on `/control-plane/*` with legacy `/admin/*` redirects and internal rewrites — see [ADMIN_UI_CONSOLIDATION_PLAN.md](ADMIN_UI_CONSOLIDATION_PLAN.md) and [INFORMATION_ARCHITECTURE_V1.md](INFORMATION_ARCHITECTURE_V1.md).

---

## 1. Executive readiness verdict

**Staging-ready; ready for production candidate (repo-side)** — Core API, retrieval governance, jobs, workers, scheduled tick, connector artifact queue handler, optional S3-compatible blobstore, Prometheus `/metrics` (dedicated registry + bearer rules outside local), and frontend build are in place. **Code-level production hardening** includes [`ValidateAPI`](../apps/api/internal/config/hardening.go) (session auth, explicit `DATABASE_URL`/`REDIS_URL`/`CORS`, OpenSearch TLS, **production-only** `OPS_AUTH_TOKEN`, `APP_PUBLIC_URL` https, secure cookies), ops/metrics bearer rules, and dev-header gating (see [PRODUCTION_HARDENING.md](PRODUCTION_HARDENING.md), [PRODUCTION_GO_LIVE_CHECKLIST.md](PRODUCTION_GO_LIVE_CHECKLIST.md)). Remaining gaps are **external/operational**: TLS termination and certs, secret manager wiring, secured OpenSearch in the target cluster, and (if the product promises large raw retention) configuring `BLOBSTORE_BACKEND=s3` and verifying uploads in staging.

---

## 2. What is fully assembled

- **Modular monolith** — [`apps/api/internal/app/deps.go`](apps/api/internal/app/deps.go) wires identity, ingestion, jobs, search, embeddings, retrieval/ask, review, governance, builders, preset catalog, onboarding.
- **Permission-aware search** — [`apps/api/internal/search/service.go`](apps/api/internal/search/service.go): domain grants + `filterHitsByEntityView` → `permissions.Resolver.Evaluate` per hit.
- **Permission-aware semantic retrieval** — [`apps/api/internal/embeddings/service.go`](apps/api/internal/embeddings/service.go): `SemanticNear` + `filterCandidatesByEntityView`.
- **Ask orchestration** — [`apps/api/internal/httpserver/routes_register.go`](apps/api/internal/httpserver/routes_register.go) `POST /ask`, `POST /entities/:id/ask` → [`retrieval_intelligence`](apps/api/internal/retrieval_intelligence/service.go) + [`qa`](apps/api/internal/qa/).
- **Knowledge jobs** — [`knowledge_jobs`](apps/api/internal/knowledge_jobs/): queued runs via [`platform/queue`](apps/api/internal/platform/queue/publisher.go), worker [`cmd/jobworker`](apps/api/cmd/jobworker/main.go); `weekly_digest` in [`orchestrator.go`](apps/api/internal/knowledge_jobs/orchestrator.go) + [`digest.go`](apps/api/internal/knowledge_jobs/digest.go).
- **Connector sync** — [`cmd/connectorworker`](apps/api/cmd/connectorworker/main.go) + ingestion service with adapter registry in `deps.go`.
- **Migrations** — Sequential files through [`000038_second_brain_overlay`](apps/api/internal/db/migrations/000038_second_brain_overlay.up.sql) (`user_chat_links`, `pre_meeting_brief_queue`, `second_brain_product_events`) after [`000037_extracted_meeting_tasks`](apps/api/internal/db/migrations/000037_extracted_meeting_tasks.up.sql). See [EXTRACTED_MEETING_TASKS.md](EXTRACTED_MEETING_TASKS.md) and [SECOND_BRAIN_BOTS.md](SECOND_BRAIN_BOTS.md). Earlier policy/feedback split: [`000031_policy_exceptions_and_feedback`](apps/api/internal/db/migrations/000031_policy_exceptions_and_feedback.up.sql) (fixes duplicate version `000011`).
- **Tests** — `go test ./...` under [`apps/api`](apps/api) passes (integration tests gated on `E2E_DB=1`).
- **Frontend** — [`apps/web`](apps/web): `npm run build` succeeds.
- **Web admin URL policy** — Canonical admin links use `/control-plane/*` ([`navigation.ts`](../apps/web/src/lib/navigation.ts), [`next.config.ts`](../apps/web/next.config.ts), [`middleware.ts`](../apps/web/src/middleware.ts)); full builders still often served via rewrites to `(dash)/` routes until control-plane pages subsume them.
- **Web product UX (local honesty)** — Single primary shell: sidebar groups **Start**, **Governance**, **Control plane** (shortcuts to operator tools), **Advanced** (audit, settings, ops). Home + Search surface the documented golden path and LIMITATIONS-aligned caveats; deprecated `/app/*` is not a second product entry.

---

## 3. What is partially assembled

- **Queues** — For **`APP_ENV=local`**, empty `REDIS_URL` yields a disabled publisher (API runs; queued work cannot enqueue). For **`staging`/`production`**, [`ValidateAPI`](../apps/api/internal/config/hardening.go) requires explicit `REDIS_URL`. Invalid Redis URL fails [`app.NewDeps`](apps/api/internal/app/deps.go).
- **Scheduled job tick** — **Implemented:** [`JobService.ProcessScheduledTick`](../apps/api/internal/knowledge_jobs/scheduled_tick.go) from [`cmd/jobworker`](../apps/api/cmd/jobworker/main.go) on `knowledge:scheduled_tick` (due triggers, idempotency window). See [knowledge-jobs-engine.md](knowledge-jobs-engine.md).
- **Connector worker — artifact queue** — **Implemented:** [`TaskIngestionProcessArtifact`](../apps/api/internal/platform/queue/tasks_connector.go) handled in [`cmd/connectorworker`](../apps/api/cmd/connectorworker/main.go) → [`ProcessQueuedRawArtifact`](../apps/api/internal/ingestion_connectors/artifact_worker.go). **Scope:** async normalizers exist only for `telegram_update`, `slack_message`, `mattermost_post`, `google_drive_file`, `notion_page`, `confluence_page`, `google_calendar_event`, and `fireflies_transcript` (per [`artifact_worker.go`](../apps/api/internal/ingestion_connectors/artifact_worker.go)); all other `artifact_type` values **no-op** when dequeued (documented in [LIMITATIONS.md](LIMITATIONS.md), [CONNECTOR_CAPABILITY_MATRIX.md](CONNECTOR_CAPABILITY_MATRIX.md), [connector-framework.md](connector-framework.md)).
- **Blob storage** — Default remains **Nop** for local; **S3-compatible** store when `BLOBSTORE_BACKEND=s3` and related env vars ([`blobstore/wire.go`](../apps/api/internal/blobstore/wire.go), [CONFIG_ENV.md](CONFIG_ENV.md)).
- **Metrics** — **Implemented:** Prometheus text/OpenMetrics on [`GET /metrics`](../apps/api/internal/httpserver/routes_health.go) via isolated registry (Go + process collectors). **HTTP:** `http_requests_total` with low-cardinality `method` and normalized Fiber `route` labels via [`PrometheusHTTPRequestsMiddleware`](../apps/api/internal/httpserver/routes_health.go) (wired in [`routes.go`](../apps/api/internal/httpserver/routes.go)); smoke asserts presence in [`scripts/smoke-local.sh`](../scripts/smoke-local.sh). Non-local requires `OPS_AUTH_TOKEN` + bearer per hardening docs.
- **Connectors** — Many adapters are thin parse/registry stubs; Telegram/chat paths are stronger (see `ingestion_connectors`).
- **Docker** — [`docker-compose.yml`](docker-compose.yml) includes `api`, `jobworker`, `connectorworker`, `web` (local dev); OpenSearch remains security-disabled for local only.

---

## 4. What is missing or broken (remaining / deferred)

- **Production auth (operator)** — For `APP_ENV=staging|production`, the API **requires** `AUTH_MODE=session`, valid `SESSION_SECRET`, explicit `DATABASE_URL`, **`REDIS_URL`**, and `CORS_ALLOW_ORIGINS`, and blocks dev-header impersonation ([`ValidateAPI`](../apps/api/internal/config/hardening.go), [`middleware.go`](../apps/api/internal/httpserver/middleware.go)). **Production** additionally requires `OPS_AUTH_TOKEN` (≥16 chars), `APP_PUBLIC_URL` with `https://`, and secure session cookies. Operators must still terminate TLS and configure identity in front of the API.
- **Unknown job types** — **Addressed:** [`runOrchestrator.executeProcessor`](apps/api/internal/knowledge_jobs/orchestrator.go) returns an error for unsupported `job_type`. Implemented types today are `weekly_digest`, `decision_extraction`, `planning_summary`, `stale_scan`, and `support_trends_extraction` (see `processor_capabilities.go`). See [knowledge-jobs-engine.md](knowledge-jobs-engine.md) and [LIMITATIONS.md](LIMITATIONS.md).
- **Full kit application** — `POST /domains/:id/apply-setup-kit` documents audit-only apply via JSON contract (see section 6).
- **Existing DBs** — If a database was migrated manually with duplicate `000011` never applied via `golang-migrate`, operators may need a one-time `schema_migrations` / SQL reconciliation after adopting `000031`. New installs should run the full chain through **`000038`** (or current head in `apps/api/internal/db/migrations/`).

---

## 5. Critical blockers (launch / safety)

| Severity | Title | Status after this change |
|----------|--------|---------------------------|
| **P0** | Duplicate migration version `000011` broke `MigrateUp` | **Fixed** — policy/feedback moved to `000031` ([`iofs` duplicate check](https://github.com/golang-migrate/migrate/blob/master/source/iofs/iofs.go)). |
| **P0** | Unauthenticated `GET /users`, `GET /domains`, `POST /access/evaluate` | **Fixed** — [`routes_register.go`](apps/api/internal/httpserver/routes_register.go): users require `requireCanManageIdentity`; domains scoped via [`ListDomainsForUser`](apps/api/internal/identity_access/repo.go); evaluate requires auth + principal rules. |
| **P1** | `perms == nil` bypass in search/embeddings | **Fixed** — explicit errors in [`search/service.go`](apps/api/internal/search/service.go) and [`embeddings/service.go`](apps/api/internal/embeddings/service.go). |
| **P1** | Ignored `NewPublisher` error | **Fixed** — [`NewDeps` returns error](apps/api/internal/app/deps.go); [`cmd/api`](apps/api/cmd/api/main.go), workers fatal on failure. |

---

## 6. Security / governance blockers (remaining)

| Severity | Finding | Evidence | Fix |
|----------|---------|----------|-----|
| **P0** (prod) | Dev header impersonation | Was: open in misconfigured env | **Mitigated in code** — staging/production require `session`; dev header only with `IsLocalDev()` ([`hardening.go`](../apps/api/internal/config/hardening.go), [`middleware.go`](../apps/api/internal/httpserver/middleware.go)). **Infra:** TLS + correct `APP_ENV` still mandatory. |
| **P2** | `/ops/health` leaked dependency errors | [`routes_health.go`](../apps/api/internal/httpserver/routes_health.go) | **Mitigated** — redacted JSON in staging when `OPS_AUTH_TOKEN` unset; bearer required when token set; **production** requires token at startup (anonymous `/ops/health` → 401). `/metrics` requires non-empty token + bearer when not local. |
| **P2** | OpenSearch `plugins.security.disabled=true` | [`docker-compose.yml`](../docker-compose.yml) | **Dev-only** compose; prod must run secured OpenSearch. **Code:** rejects `http://` OpenSearch URL in staging/production unless `OPENSEARCH_ALLOW_INSECURE_HTTP=1`. |

---

## 7. E2E flow gaps

| Flow | Status | Evidence |
|------|--------|----------|
| 1. First-time setup / onboarding | **Partially works** | [`onboarding/service.go`](apps/api/internal/onboarding/service.go) `Launch` instantiates presets; kit route audit-only + [`routes_register.go`](apps/api/internal/httpserver/routes_register.go) JSON contract. |
| 2. Ingestion → job → digest | **Partially works** | Digest + job sources enforced in [`orchestrator.go`](apps/api/internal/knowledge_jobs/orchestrator.go); scheduled tick **enqueues runs** via jobworker; [`digest_flow_test.go`](apps/api/internal/integration/digest_flow_test.go) needs `E2E_DB=1` for full chain. |
| 3. Ask over scoped corpus | **Works** (config-dependent) | [`GET /search`](apps/api/internal/httpserver/routes_register.go) → `SearchScoped`; semantic/hybrid needs `OPENAI_API_KEY`, `OPENROUTER_API_KEY`, or `OPENAI_MOCK=1`. |
| 4. Project memory view | **Partially works** | Entity routes + web pages; governed by domain grants + entity `Evaluate`. |
| 5. Governance review / approval | **Partially works** | [`review`](apps/api/internal/review/), [`governance`](apps/api/internal/governance/); not every transition audited in this pass. |
| 6. Role / scenario / job builder | **Partially works** | Dedicated routes and migrations; non-empty **`scenario_code`** is enforced on **`GET /search`** (query), **`POST /ask`** (JSON), and **`POST /entities/:id/ask`** (optional JSON) via `PrincipalAllowsScenario` ([`routes_register.go`](../apps/api/internal/httpserver/routes_register.go)). Omitted `scenario_code` on entity Ask remains entity `view` + permission-scoped retrieval only ([AI_RETRIEVAL_GOVERNANCE.md](AI_RETRIEVAL_GOVERNANCE.md) §4.1). Allow/deny coverage: [`scenario_search_ask_gate_test.go`](../apps/api/internal/integration/scenario_search_ask_gate_test.go) with `E2E_DB=1`; entity Ask scenario deny: [`entity_ask_test.go`](../apps/api/internal/integration/entity_ask_test.go). |

---

## 8. Deployment blockers

- **Secrets** — `SESSION_SECRET`, SMTP, OAuth as needed; never commit `.env`.
- **Workers** — Run `jobworker` and `connectorworker` when using Redis queues ([`docker-compose.yml`](docker-compose.yml)).
- **Migrations** — API runs [`db.MigrateUp`](apps/api/internal/db/db.go) on start ([`cmd/api/main.go`](apps/api/cmd/api/main.go)). **jobworker** and **connectorworker** also call `MigrateUp` so a cold DB does not require API-first ordering.
- **Docker images** — [`Dockerfile.api`](Dockerfile.api) (api + both workers); [`Dockerfile.web`](Dockerfile.web) for Next.js.
- **Deploy / smoke docs** — [DEPLOY_CHECKLIST.md](DEPLOY_CHECKLIST.md), [STAGING_SMOKE_TEST.md](STAGING_SMOKE_TEST.md), [PRODUCTION_HARDENING.md](PRODUCTION_HARDENING.md), [`scripts/smoke-local.sh`](../scripts/smoke-local.sh).
- **Production config gate** — [`ValidateAPI`](../apps/api/internal/config/hardening.go) runs from [`cmd/api/main.go`](../apps/api/cmd/api/main.go): session + secrets + CORS + DB env + OpenSearch HTTPS rules for `APP_ENV` staging/production. Workers use [`ValidateWorker`](../apps/api/internal/config/hardening.go).

**Runtime verification (this audit):**

| Check | Result |
|-------|--------|
| `go build ./...` (apps/api) | Pass |
| `go test ./...` (apps/api) | Pass |
| `npm run typecheck` (apps/web) | Pass |
| `npm run build` (apps/web) | Pass |
| CI Postgres image | **`pgvector/pgvector:pg16`** in [`.github/workflows/ci.yml`](../.github/workflows/ci.yml) so `CREATE EXTENSION vector` succeeds. |
| `TestMigrateUp_Idempotent` (`DATABASE_URL`) | Runs in CI; proves embedded migrations apply twice without error ([`migrate_smoke_test.go`](../apps/api/internal/db/migrate_smoke_test.go)). |
| `docker compose config` | Validated locally / in Docker workflow. |
| `docker compose build` (api, web, workers) | **Docker** workflow job `compose-validate` ([`.github/workflows/docker.yml`](../.github/workflows/docker.yml)). |
| CI golden-path API smoke | After `go test`, [`.github/workflows/ci.yml`](../.github/workflows/ci.yml) runs `go run ./cmd/api` against service Postgres + [`scripts/smoke-local.sh`](../scripts/smoke-local.sh) (`OPENAI_MOCK=1`). |
| Full `docker compose up` + curl smoke | Documented in [STAGING_SMOKE_TEST.md](STAGING_SMOKE_TEST.md); [`scripts/smoke-local.sh`](../scripts/smoke-local.sh) passed against compose API (`/health`, `/ops/health`, `/domains` with dev header) in a local Docker run (2026-04-11). |
| `go test ./...` (apps/api), `npm run build` (apps/web) | **Re-verified Pass** on 2026-04-12 for this refresh. |

---

## 9. Recommended fix order (ongoing)

1. **Infra:** TLS termination, secret rotation, secured OpenSearch cluster (compose profile is not prod).  
2. **Blob (operator):** when raw/object retention is required in prod, set `BLOBSTORE_BACKEND=s3` and verify bucket policy + connectivity; default Nop remains valid for deployments that do not persist blobs via S3.  
3. ~~Scheduled triggers + artifact queue handler~~ — **Done** (see §3): `ProcessScheduledTick`, `ingestion:process_artifact` with documented type scope.  
4. ~~Fail closed on unknown `job_type`~~ — **Done:** unsupported `job_type` fails the orchestrator; create API rejects unimplemented types (see [knowledge-jobs-engine.md](knowledge-jobs-engine.md)).  
5. ~~Prometheus `/metrics`~~ — **Done:** see [`routes_health.go`](../apps/api/internal/httpserver/routes_health.go) and [PRODUCTION_HARDENING.md](PRODUCTION_HARDENING.md).  
6. ~~**`job_type` matrix + UI honesty**~~ — **Done:** engine-metadata route, presets, jobs UI `processor_implemented`.  
7. **Admin consolidation phase 4:** when control-plane pages fully replace dash builders, remove or thin `(dash)/admin/*` physical routes per [ADMIN_UI_CONSOLIDATION_PLAN.md](ADMIN_UI_CONSOLIDATION_PLAN.md) (**blocked** on CP parity — not a blocker for shipping API + current web).

---

## 9a. Recommended next engineering slice (post–release-hardening)

Focus areas after the **release-hardening** tranche (not P0 for repo-side production candidate):

1. ~~**Optional scenario parity**~~ — **Done:** `POST /entities/:id/ask` accepts optional JSON `scenario_code` with the same `PrincipalAllowsScenario` check as `POST /ask` / `GET /search` ([`routes_register.go`](../apps/api/internal/httpserver/routes_register.go), [`AI_RETRIEVAL_GOVERNANCE.md`](./AI_RETRIEVAL_GOVERNANCE.md) §4.1).  
2. **Ingestion depth** — extend artifact worker outcomes beyond `success`/`error` (e.g. distinguish normalized vs no-op vs failure) where operators need dashboards; add normalizers beyond `telegram_update` when roadmap requires.  
3. **Blob / S3 in CI** — optional MinIO-backed integration tests; operator smoke for real S3 remains documented ([CONFIG_ENV.md](CONFIG_ENV.md), [connector-framework.md](connector-framework.md)).  
4. **Web** — when the UI has a defined scenario context, pass `scenario_code` on Search requests (API already accepts the query param).  
5. **Admin consolidation** — control-plane parity per [ADMIN_UI_CONSOLIDATION_PLAN.md](ADMIN_UI_CONSOLIDATION_PLAN.md) (see §9 item 7).

---

## 10. Final go/no-go recommendation

- **Local / compose (`APP_ENV=local`)**: **go** for development after migrations + smoke ([STAGING_SMOKE_TEST.md](STAGING_SMOKE_TEST.md)).  
- **Hosted staging (`APP_ENV=staging`)**: **go** when env matches [PRODUCTION_HARDENING.md](PRODUCTION_HARDENING.md) (session, secrets, CORS, ops token as needed) and smoke passes with session login.  
- **Public production**: **candidate** when code gates + operator checklist in [PRODUCTION_HARDENING.md](PRODUCTION_HARDENING.md) are satisfied **and** TLS, ingress, secret manager, and secured OpenSearch are in place; **no-go** if `APP_ENV` mis-set or compose-style OpenSearch is used unchanged.

---

## Final artifact (summary)

1. **Readiness verdict:** Staging-ready; **production-candidate** with explicit infra steps ([PRODUCTION_HARDENING.md](PRODUCTION_HARDENING.md)).  
2. **Blocker list:** Duplicate migration and open HTTP surfaces addressed; code enforces session + ops redaction + OpenSearch TLS policy for non-local; blob defaults to Nop until operators enable S3; metrics are Prometheus-backed (`http_requests_total` + defaults) with bearer protection outside local.  
3. **Minimal must-fix before public deploy:** TLS termination, secret manager, secured OpenSearch cluster, `OPS_AUTH_TOKEN`, `NEXT_PUBLIC_USE_DEV_HEADER=false`, follow [PRODUCTION_HARDENING.md](PRODUCTION_HARDENING.md) §6.  
4. **Safe-to-deploy-after-this checklist:** Migrations apply; `ValidateAPI` passes; Redis + workers; session login smoke; `/domains` scoped; search/ask as needed; optional `/metrics` scrape with bearer where required.

---

## Second artifact (historical — early 2026 hardening tranche)

**Status:** The table below records the **completed** P0/P1 backlog from the first production-readiness pass. It is **not** an open task list; current gaps are in §4, §6, §9, and §9a.

### Completed P0 / P1 items (archive)

| ID | Item |
|----|------|
| P0 | Renumber `000011_policy_*` → `000031_policy_exceptions_and_feedback` |
| P0 | Authenticate and gate `GET /users`, `GET /users/:id`, `GET /domains`, `POST /access/evaluate` |
| P1 | Fail-closed `perms == nil` in search + embeddings filters |
| P1 | Propagate `queue.NewPublisher` error via `NewDeps` |
| P1 | Explicit `apply-setup-kit` response contract + ACCESS_MODEL update |
| P1 | Compose + Dockerfiles for api / workers / web |

### Deploy-candidate checklist

- [ ] Fresh DB: migrations through latest version apply without error.  
- [ ] `REDIS_URL` valid if using queues; both workers running.  
- [ ] `AUTH_MODE` and `SESSION_SECRET` match environment.  
- [ ] `CORS_ALLOW_ORIGINS` matches web origin.  
- [ ] `OPENSEARCH_URL` https or empty (or explicit insecure allow for private net).  
- [ ] `OPS_AUTH_TOKEN` for detailed ops scraping in staging/production.  
- [ ] Smoke: session (prod) or dev header (local) → domains ⊆ grants → search → ask.

### Three-step path: fix → staging → deploy

1. **Fix** — Merge migration + HTTP hardening + compose; run `go test ./...`, `npm run build`, migrate clean DB in CI or locally.  
2. **Verify in staging** — Compose or k8s: full stack, session auth if prod-like, synthetic ingest + digest + ask.  
3. **Deploy** — Roll API + workers together; run migrations; monitor job queue and OpenSearch health.
