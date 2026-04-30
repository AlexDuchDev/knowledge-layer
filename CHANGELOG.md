# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.6.0] — 2026-04-30

Adds a generic `openapi_v3` connector type that lets operators add REST-API source feeds via configuration instead of per-vendor Go code. Adopts the spirit of Hugr's data-source model (one configurable HTTP source) while keeping Knowledge Layer's REST/connector-first surface. See [ADR-0016](docs/adr/0016-openapi-v3-generic-connector.md).

### Added

- **`openapi_v3` connector type** at id `20000000-0000-0000-0000-000000000014` (migration 000044). Single new connector serves any REST API matching v0.6.0 constraints — bearer auth, offset/limit pagination, JSONPath strict-mode mapping, 5MB spec cap.
- **`apps/api/internal/ingestion_connectors/adapters/openapi_v3/`** — five-file package:
  - `config.go` — `FeedConfig` struct + activation `Validate()` covering 11 misconfiguration classes.
  - `jsonpath.go` — strict-mode wrapper around `PaesslerAG/jsonpath`. Rejects `?(...)` filter expressions at activation; missing-field reads return empty (soft) so individual bad records don't abort sync.
  - `spec_validate.go` — `FetchAndValidateSpec(ctx, openAPIURL, listPath)` parses via `getkin/kin-openapi`, enforces 5MB cap + 10s timeout + `IsExternalRefsAllowed=false` (SSRF defense), asserts `list_path` exists as a `GET` operation.
  - `sync.go` — `Run(ctx, baseURL, cfg, httpClient, onItem)` paginates via offset/limit, maps each item through the operator's JSONPath table, hashes for dedup. Caller owns persistence (Service.PersistNormalizedRecord) so this package stays decoupled from the rest of ingestion.
  - `adapter.go` — implements `ConnectorAdapter`. `ValidateSourceFeedConfig` runs both schema + spec checks at activation; `SyncFeed` delegates to the service layer.
- **Adversarial test suite** (`openapi_v3_test.go`) — 21 sub-tests including JSONPath-strict-mode rejection of three filter shapes, 11 config-misuse cases, oversized-spec rejection, valid-spec happy path, missing-field soft empty.
- **Closed `record_type` enum** matching `chunks/extract.go`'s 14 supported types — typo from operator gets a clear validation error rather than a silent no-op.
- **[ADR-0016 — OpenAPI v3 generic connector](docs/adr/0016-openapi-v3-generic-connector.md)** codifies the v0.6.0 scope cuts: pagination strategies other than offset/limit, auth schemes other than bearer, JSONPath filter expressions, external `$refs`, and record-type extensions are all out of scope.

### Changed

- `app.NewDeps` registers the new adapter alongside the existing 19 — connector total now 20.
- New deps: `github.com/getkin/kin-openapi/openapi3` (spec parser) + `github.com/PaesslerAG/jsonpath` (JSONPath engine). Both pure Go.
- `.env.example` documents the per-feed config shape so operators can start from a known-good template.

### Operations

- **Per-feed token storage** matches existing connector pattern (notion, jira, etc.). Encryption-at-rest for `connector_config_json` secrets is a pre-existing gap not unique to this connector — separate ADR will tackle it when an operator demands it.
- **Source-feed UI** does not yet surface the new connector — adding it to `apps/web/src/app/(dash)/source-feeds/page.tsx` is a v0.6.x follow-up. Operators can create feeds via direct `POST /source-feeds` with `connector_id=20000000-0000-0000-0000-000000000014` until the picker lands.
- **Future widening (v0.7+):** cursor + link-header pagination, OAuth2 client_credentials outbound, auto-suggest item_mapping from spec response schemas, GraphQL connector. Each gets its own ADR.

## [0.5.1] — 2026-04-30

Adds the `/mcp` endpoint that consumes the v0.5.0 OAuth proxy. MCP clients (Claude Desktop, Cursor, IDE plugins) authenticate against the operator's OIDC, exchange an auth code for a JWT bearer at `POST /oauth/token`, then call MCP tools at `POST /mcp`. **Every tool call routes through `AccessEvaluator.Evaluate` exactly like REST** — there is no admin bypass.

### Added

- **MCP server** ([`apps/api/internal/mcp/`](apps/api/internal/mcp/)) — built on [`mark3labs/mcp-go`](https://github.com/mark3labs/mcp-go) v0.50.0 with streamable HTTP transport. Three initial tools, each access-guarded:
  - `kl_search` — keyword search across permitted entities. Wraps `Retrieval.SearchScoped`. Action `view`, resource_type `entity`.
  - `kl_ask_global` — synthesized Q&A through the privacy gateway. Wraps `Retrieval.AskGlobal`. Same action/resource_type.
  - `kl_get_entity` — single-entity fetch by UUID. Wraps `Entities.Get`.
- **Mandatory access-guard** ([`access_guard.go`](apps/api/internal/mcp/access_guard.go)) — `withAccessGuard(eval, action, resourceType, fn)` wraps every tool. Calls `AccessEvaluator.Evaluate` first; deny short-circuits before the inner handler runs. Missing principal in context is a hard reject (defense in depth — the bearer middleware is supposed to populate it, but a bug there must not silently allow the call).
- **Bearer middleware** ([`route.go`](apps/api/internal/mcp/route.go)) — Fiber middleware runs before `/mcp`. Reads `Authorization: Bearer X`, calls `oauth_proxy.Server.VerifyBearer` (added in v0.5.0), stashes the principal UUID via `mcp.WithPrincipal(ctx, ...)`. 401s with `WWW-Authenticate: Bearer realm="mcp", error="invalid_token"` on bad/missing tokens.
- **Static-contract test** ([`server_test.go`](apps/api/internal/mcp/server_test.go)) — `TestNew_allToolsAccessGuarded` iterates the registered tool set and asserts every handler short-circuits on a deny decision. Prevents the future bug where someone adds a tool that bypasses `withAccessGuard`. Plus 4 unit tests covering allow/deny/missing-principal/eval-error paths.

### Changed

- `app.NewDeps` builds an `*mcpserver.MCPServer` when `MCP_ENABLED=true` and the OAuth proxy is also up.
- `httpserver.Mount` mounts `/mcp` after the OAuth proxy. Bearer middleware sits inline; the MCP handler is wrapped via Fiber's `adaptor.HTTPHandler` because `mcp-go` exposes a stdlib `http.Handler`.
- `config.ValidateAPI` rejects `MCP_ENABLED=true` without `OAUTH_PROXY_ENABLED=true`. The /mcp endpoint cannot validate bearers without the proxy; failing at startup beats a deployed-but-broken endpoint.
- New env: `MCP_ENABLED` (default `false`). Documented in [`docs/CONFIG_ENV.md`](docs/CONFIG_ENV.md) "MCP endpoint" section.
- ADR-0015 promoted to "OAuth + MCP" with v0.5.1 implementation notes appended.

### Operations

- **Audit emission for `mcp.tool.invoked` is deferred to a follow-up patch.** v0.5.1 logs to stderr; the underlying audit_events row is wired in v0.5.x once operators have a reason to query MCP traffic via `/audit-events`.
- **Token revocation cascade.** A revoked OAuth client (`oauth_clients.revoked_at = now()`) blocks new `/oauth/token` exchanges; bearers issued before revocation expire naturally within 1h. Force-revoke bearers across the fleet by rotating `OAUTH_SECRET_KEY` (acceptable trade-off — documented in ADR-0015).
- **Single-instance assumption persists.** ADR-0014 stays in force; auth codes still live 2 minutes in `sync.Map`. Multi-pod operators continue to need sticky LB.

## [0.5.0] — 2026-04-30

Adds an OAuth 2.1 authorization-server proxy fronting an operator-supplied OIDC issuer. Off by default. The proxy is the prerequisite for the v0.5.1 MCP endpoint, which will let Claude Desktop / Cursor authenticate against the operator's IDP (Keycloak, Auth0, Okta, Dex) and consume Knowledge Layer tools with `AccessEvaluator`-gated bearers. v0.5.0 ships the proxy alone so OIDC integration, secret-key rotation, and audit posture bake for one release cycle before any MCP traffic hits.

### Added

- **OAuth 2.1 proxy** ([`apps/api/internal/oauth_proxy/`](apps/api/internal/oauth_proxy/)) — five HTTP endpoints, stateless except for short-lived in-process auth codes:
  - `GET /.well-known/oauth-authorization-server` — RFC 8414 metadata.
  - `GET /oauth/authorize` — entry point; signs a state Payload (HMAC-SHA256 over the MCP client's redirect_uri, PKCE challenge, and nonce) and 302s to the IDP.
  - `GET /oauth/callback` — verifies state HMAC, exchanges IDP code via `coreos/go-oidc/v3`, maps OIDC `sub` to a KL principal via `TokenBridge`, mints a one-time auth code, redirects MCP client back to its registered URI.
  - `POST /oauth/token` — auth-code grant with PKCE-S256 verification; returns a 1-hour JWT (HS256, signed with `OAUTH_SECRET_KEY`).
  - `POST /oauth/register` — RFC 7591 dynamic-client registration. New `oauth_clients` table (migration 000043) holds `client_id`, bcrypt-hashed secret, redirect_uri allow-list, and `revoked_at` for operator revocation.
- **`oauth_proxy.Server.VerifyBearer(jwt)`** — single chokepoint v0.5.1 MCP middleware will call to map a bearer back to a `users.id` UUID. Symmetric HS256 by design (no KMS round-trip needed; same secret on every API replica per ADR-0014's single-instance stance).
- **State HMAC + 6-test suite** ([`state.go`](apps/api/internal/oauth_proxy/state.go), [`state_test.go`](apps/api/internal/oauth_proxy/state_test.go)) — covers happy-path round-trip, tampered body, wrong key (rotation), expired state, short-key rejection, malformed input.
- **[ADR-0015 — OAuth proxy and MCP bridge](docs/adr/0015-oauth-proxy-and-mcp-bridge.md)** — codifies single-tenant stance preserved, AccessEvaluator-mandatory contract, no admin-bypass tokens, alternatives considered.

### Changed

- `app.NewDeps` builds an optional `*oauth_proxy.Server` when `OAUTH_PROXY_ENABLED=true`. `Mount` registers the routes before everything else so RFC 8414's anonymous metadata endpoint stays public.
- `config.ValidateAPI` rejects `OAUTH_PROXY_ENABLED=true` without `OAUTH_SECRET_KEY` ≥ 32 bytes, `OIDC_ISSUER_URL`, `OIDC_CLIENT_ID`, or (in production) `OIDC_CLIENT_SECRET`.
- New env: `OAUTH_PROXY_ENABLED`, `OAUTH_SECRET_KEY`, `OIDC_ISSUER_URL`, `OIDC_CLIENT_ID`, `OIDC_CLIENT_SECRET`, `OIDC_SUB_COLUMN`, `OAUTH_PROXY_ISSUER`, `OAUTH_PROXY_CALLBACK`. Documented in [`docs/CONFIG_ENV.md`](docs/CONFIG_ENV.md) "OAuth 2.1 proxy" section.

### Operations

- **Audit emission for `oauth.client.registered` and `oauth.token.issued`** is currently **stderr-only**. Wiring through the audit_events table is deferred to v0.5.1 when MCP traffic gives operators a reason to query these events via `/audit-events`.
- **No refresh tokens.** Clients re-authorize hourly; silent IDP redirect when the IDP session is fresh keeps UX cost low. Revisit if operators complain.
- **Single-instance assumption.** Auth codes live 2 minutes in `sync.Map`. Multi-pod deployments need sticky LB or a Redis backend (out of scope per ADR-0014).
- **Token revocation.** Operators set `oauth_clients.revoked_at = now()` to block a client. In-flight bearers for that client expire naturally (1h max). To force-revoke individual bearers, rotate `OAUTH_SECRET_KEY` (invalidates ALL in-flight tokens — documented trade-off).

## [0.4.0] — 2026-04-29

Adopts three Hugr-inspired patterns into Knowledge Layer: an L1 cache for hot reads, an `entity_summarize` knowledge job that auto-fills synthesized summaries on the search projection, and a `kltools` operator CLI that ships inside the existing API image. Migration `000042` adds `synthesized_summary` + `synthesized_at` columns to `entity_search_projection`. No backward-incompatible API changes; the cache and the new job are off by default.

### Added

- **L1 in-process cache** ([`apps/api/internal/cache/`](apps/api/internal/cache/)) — BigCache via [eko/gocache/v4](https://github.com/eko/gocache) wrapping `Get`/`Set`/`Delete`/`DeletePrefix`. Wired into hot reads: `GET /domains`, `GET /users/:id/effective-access` (simple-variant only — query-param-rich variants pass through), `GET /search` (keyword filters), `GET /knowledge-jobs/engine-metadata`. Every cache key is **principal-scoped**: `cache.DomainsKey(principal)`, `cache.EffectiveAccessKey(principal, target)`, `cache.SearchKeywordKey(principal, query, filters)`. Cross-principal contamination is the safety-critical regression to avoid; the test suite (`cache_test.go`, 7 tests) asserts every key embeds the principal. Cache `X-Cache: HIT|MISS` header surfaced for diagnosability.
- **Cache invalidator** ([`apps/api/internal/cache/invalidation.go`](apps/api/internal/cache/invalidation.go)) — drops cached entries on state-changing events. Hooks fire after `entity.published` (direct + review approval), `domain_grant.upserted` / `.updated` / `.deleted` (via `Invalidator.RoleGranted`), `role.assignment.created`. Invalidator refuses empty-prefix calls so a coding bug can't wipe the entire cache.
- **`entity_summarize` knowledge job** ([`apps/api/internal/knowledge_jobs/entity_summarize.go`](apps/api/internal/knowledge_jobs/entity_summarize.go)) — sixth implemented job_type. Reads entities lacking `synthesized_summary`, routes each through `privacy.PrivacyGateway.InvokeOpenAI` with `PromptTemplateID="entity_summarize.v1"`, UPSERTs `synthesized_summary` + `synthesized_at` on `entity_search_projection`. Cost discipline: per-run cap (default 100, hard cap 500), body excerpt truncated to ~1200 runes. Fails clearly when no LLM provider is configured rather than silently producing empty summaries.
- **Prompt template** [`entity_summarize.v1.json`](apps/api/internal/ai/prompts/templates/entity_summarize.v1.json) — concise prose, language-matched, no preamble.
- **`kltools` CLI** ([`apps/api/cmd/kltools/`](apps/api/cmd/kltools/)) — operator binary with three subcommands: `summarize` (backfill via the entity_summarize job, dry-run by default), `reindex` (rebuild chunks for a specific entity or drain pending normalized_records, dry-run by default), `schema-info` (read-only pipeline-stage counts + connector inventory + implemented job types). Ships inside `ghcr.io/.../knowledge-layer-api:v0.4.0` as `/app/knowledge-tools`. Operator usage: `docker compose exec api /app/knowledge-tools schema-info`. DB pool capped at 4 connections so the CLI never starves the running API. Pattern after Hugr's `hugr-tools`.
- **5-touch-point job-type registration** for `entity_summarize`: [`processor_capabilities.go`](apps/api/internal/knowledge_jobs/processor_capabilities.go) (slice + capability switch), [`orchestrator.go`](apps/api/internal/knowledge_jobs/orchestrator.go) (executeProcessor case + new `summarizer` field on the orchestrator). `resolveSources` and `buildInputRefs` switches deliberately not extended — the job reads entities directly, not source feeds.

### Changed

- `entity_search_projection` gains `synthesized_summary TEXT` and `synthesized_at TIMESTAMPTZ` columns (migration 000042). Partial index `idx_entity_search_projection_pending_summary` lets the backfill paginate efficiently. OpenSearch indexed text widening to include `synthesized_summary` is **not yet wired** — that's a v0.4.x follow-up; the column is populated in v0.4.0 so the data is ready when the indexer is updated.
- `JobService` constructor now takes an `*EntitySummarizer`. `app.NewDeps` builds it after `PrivacyGateway` so the orchestrator never holds a nil summarizer when the entity_summarize case is enabled.
- `Dockerfile.api` builds and ships a fourth binary (`/app/knowledge-tools`) alongside `knowledge-api`, `knowledge-jobworker`, `knowledge-connectorworker`. No new image; no new release-CI job. Web image unchanged.
- `Makefile` test target builds `cmd/kltools` to keep the binary green in CI.
- `.env.example` documents `CACHE_L1_ENABLED`, `CACHE_L1_TTL_SECONDS`, `CACHE_L1_MAX_MB` (all default off / conservative).

### Operations

- **Cache is off by default**. Enable on the API container only — workers don't need it. The cache is principal-scoped and invalidates on entity publish, role grant, policy update, feed config patch.
- **`/users/:id/effective-access` cache hit can be up to TTL seconds stale.** This is UI affordance staleness only — `AccessEvaluator.Evaluate` runs synchronously on every authorization decision, so a freshly-revoked role is enforced immediately on protected operations. To eliminate the UI window, leave `CACHE_L1_ENABLED=false`. Documented in [`docs/PRODUCTION_HARDENING.md` §12](docs/PRODUCTION_HARDENING.md).
- **`kltools` write subcommands require `--yes`**. Dry-run by default; the message explains exactly what would change. Inspired by Hugr's `hugr-tools` parallel-LLM pattern but stricter on operator confirmation since the binary ships in the same image as the running API.

## [0.3.0] — 2026-04-29

Closes the chunking gap: before this release, the entire ingestion pipeline (`raw_artifacts → normalized_records → entities → chunks → embeddings`) only chunked content that became an entity, and **only Google Drive documents had a normalized-record-to-entity mapper**. All other record types — chat messages (Slack / Telegram / Mattermost / Teams), meeting transcripts (Fireflies / Google Meet), email (Gmail / M365), work items (Linear / Jira / Asana / Trello), support conversations (Zendesk / Intercom / HubSpot), Confluence / Notion docs, calendar events — sat in `normalized_records` indefinitely, never chunked, never embedded, invisible to retrieval. A live audit on a v0.2.x stack showed 16 normalized records resulting in **0 entities and 0 chunks**.

v0.3.0 decouples chunks from entities so any record_type can be chunked directly from its normalized-record payload. After this release, `/health`-equivalent live verification on a fresh ingestion stack shows `8 normalized records → 8 chunks → 8 chunks_rebuilt_at stamps` within ~8 seconds of the connectorworker starting.

### Added

- **Polymorphic chunk source** (migration 000041, [`internal/chunks/service.go`](apps/api/internal/chunks/service.go)). The `chunks` table now carries `normalized_record_id UUID NULL` alongside the existing `entity_id UUID NULL`, with a `chunks_exactly_one_source_chk` constraint guaranteeing every row has exactly one parent. `source_type` discriminates: `entity_body` (legacy) vs `normalized_record` (new). New per-record-type extraction registry in [`internal/chunks/extract.go`](apps/api/internal/chunks/extract.go) maps 14+ record types (chat, docs, meeting, calendar, email, work_item, support_ticket, etc.) to record-relevant text; unknown record_types return cleanly without inserting noise.
- **Periodic backfill loop** in [`cmd/connectorworker/main.go`](apps/api/cmd/connectorworker/main.go). Runs `chunks.RebuildPendingNormalizedRecords` every 30 s (and once on startup). Drains up to 100 normalized_records with `chunks_rebuilt_at IS NULL` per tick; logs `chunk backfill processed=N failures=M`. Decouples the chunks-rebuild contract from the 24+ scattered `INSERT INTO normalized_records ...` call sites in connector adapters — new connectors land without remembering to fire any hook.
- **Synchronous fast path** via `Service.PersistNormalizedRecord(...)` in [`internal/ingestion_connectors/artifact_worker.go`](apps/api/internal/ingestion_connectors/artifact_worker.go). When an adapter routes through this helper (currently the typed-artifact-worker path), the chunks-rebuild + embedding-enqueue fires synchronously inside the same goroutine. The wider connector-adapter set falls back to the periodic backfill at 30 s lag.
- **Embedding retrieval UNION query** in [`internal/embeddings/service.go`](apps/api/internal/embeddings/service.go). `Candidate` gains `NormalizedRecordID uuid.UUID`; the SQL behind `SemanticNear` now UNIONs entity-rooted and normalized-record-rooted chunks with appropriate domain access (`source_feeds.domain_id` for the latter) and synthesized governance defaults (`pre_approved` / `mirrored_authority` / `published` / `fresh`) for raw-source candidates.
- **Tests**: `TestRebuildNormalizedRecordChunks_extractsTextAndPersists` exercises the full DB-shape (FK to `normalized_records`, NULL `entity_id`, `source_type='normalized_record'`, text round-trip) against a real Postgres. `TestExtractTextFromNormalizedRecord_perTypeRegistry` covers 8 record types and the unknown-type fall-through. Adding a new connector record_type without a chunk extractor will fail the latter loudly.

### Changed

- **`embeddings.Candidate.EntityID` is now optional.** Existing callers that always read `c.EntityID.String()` will get the zero UUID for normalized-record candidates. v0.3.0 conservatively *filters* `nil`-EntityID candidates out of the Ask path in [`internal/retrieval/service.go`](apps/api/internal/retrieval/service.go) (both `semantic_only` and `hybrid` modes) so /ask responses don't break — they just don't yet cite normalized-record fragments. Pure-embedding callers (`SemanticNear`) DO see them. v0.3.1 wires the synthesized citations through Ask.
- `chunks.Service.OnNormalizedRecordPersisted` stamps `normalized_records.chunks_rebuilt_at = now()` after a successful rebuild so the backfill loop never reprocesses the same row.

### Migration notes

- Migration 000041 is **non-destructive**. Pre-existing entity-rooted chunks keep their shape (`entity_id` non-null, `normalized_record_id` NULL, `source_type='entity_body'`). The CHECK constraint applies to new rows only.
- After upgrade, the connectorworker's first tick (≤30 s) backfills any normalized_records inserted before the upgrade. Live-verified: 8 records → 8 chunks in 8 s on a quad-core dev box.
- Rollback (000041 down) drops the `normalized_record_id` column, which cascades to embeddings — operators rolling back must regenerate embeddings.

### Known follow-ups (v0.3.1 candidates)

- **Ask citations for normalized-record fragments** — synthesize a virtual citation (channel + timestamp + author_ref) per record_type so /ask can quote chat / meeting fragments without an entity.
- **OpenSearch indexing for normalized_records** — `/search` keyword-only retrieval today still indexes only entities. Mirroring the chunks change at the OpenSearch layer is the next pass.
- **24+ inline `INSERT INTO normalized_records` call sites** still bypass the synchronous hook. They land via the periodic backfill within 30 s. Refactoring them to route through `Service.PersistNormalizedRecord` is a mechanical chore that closes the lag window.

## [0.2.3] — 2026-04-28

Single-fix patch closing the last open Dependabot alert from the v0.2.2 security pass. Brings the repo's open advisory count to zero. Backward-compatible with v0.2.2.

### Security

- **`go.opentelemetry.io/otel` v1.39.0 → v1.41.0** (transitive bump including `otel/metric` and `otel/trace`) — fixes CVE-2026-29181 (high, multi-value `baggage` header extraction causes excessive allocations / remote DoS amplification). Practical exploit risk in this codebase is low — the OTEL pipeline is operator-internal and `/metrics` is bearer-gated outside `APP_ENV=local` per [PRODUCTION_HARDENING.md](docs/PRODUCTION_HARDENING.md) — but staying on patched releases is the contract.

## [0.2.2] — 2026-04-28

Re-cut of [0.2.1] after the Web image build hung for 2 hours on `linux/arm64` QEMU emulation during `npm ci`. v0.2.2 ships every change intended for v0.2.1 (Dependabot security updates + release-notes fix) plus the build-system change that unblocked the Web publish. The v0.2.1 git tag exists but never produced a stable Web image or a GitHub Release; the API image at `ghcr.io/.../knowledge-layer-api:v0.2.1` is the partial output of that cancelled run and operators should pull `:v0.2.2` or `:latest` instead.

### Changed

- **Web image is now `linux/amd64` only.** [`.github/workflows/release.yml`](.github/workflows/release.yml) no longer asks buildx to also produce `linux/arm64` for the Web image. The Next.js `npm ci` step under arm64 QEMU emulation hangs (one of the postinstall scripts loops indefinitely), even though the same step takes <1 minute on amd64. The API image stays multi-arch — Go cross-compiles natively without emulation, so amd64 + arm64 API images remain available. Operators on arm64 hosts (Graviton, Apple Silicon) who need the Web image should build from source per [`CONTRIBUTING.md`](CONTRIBUTING.md).

The remainder of this entry is identical to the [0.2.1] section below; consolidated here for the canonical release notes.

### Security

Closes 14 open Dependabot alerts (4 critical / 3 high / 6 moderate / 1 low) in the lockfiles by bumping vulnerable direct + indirect dependencies. None of the advisories were known to be exploitable in this codebase's call graph, but staying ahead of advisories is the contract.

- **`golang.org/x/crypto` → v0.46.0** — fixes CVE-2024-45337 (critical, ServerConfig.PublicKeyCallback authorization bypass), CVE-2025-22869 (high, ssh DoS), CVE-2025-47914 + CVE-2025-58181 (medium, agent panic + ssh memory).
- **`google.golang.org/grpc` → v1.79.3** — fixes CVE-2026-33186 (critical, authorization bypass via missing leading slash in `:path`).
- **`github.com/jackc/pgx/v5` → v5.9.2** — fixes CVE-2026-33816 (critical, memory-safety) and GHSA-j88v-2chj-qfwx (low, SQL-injection via dollar-quoted-string placeholder confusion).
- **`github.com/gofiber/fiber/v2` → v2.52.12** — fixes CVE-2025-66630 (critical, predictable UUIDv4 fallback), CVE-2025-54801 (high, BodyParser crash on unvalidated large slice index), CVE-2026-25882 (medium, Route Parameter Overflow DoS).
- **`golang.org/x/oauth2` → v0.34.0** — fixes CVE-2025-22868 (high, improper validation of syntactic correctness).
- **`golang.org/x/net` → v0.48.0** — fixes CVE-2025-22870 + CVE-2025-22872 (medium, IPv6 zone-id proxy bypass + XSS).
- **`postcss` → ^8.5.12** in `apps/web` (top-level + `overrides` to force the nested `next/node_modules/postcss`) — fixes GHSA-qx2v-qp2m-jg93 (medium, XSS via unescaped `</style>` in CSS Stringify).

### Changed (toolchain side-effects)

- **Go toolchain → 1.25.0** in `apps/api/go.mod` (forced by the upgraded modules above). [`Dockerfile.api`](Dockerfile.api) now uses `golang:1.25-alpine`. [`.github/workflows/ci.yml`](.github/workflows/ci.yml) reads the Go version from `apps/api/go.mod` via `go-version-file` instead of hard-coding `1.22.x`, so future toolchain bumps are picked up automatically by both CI gates.

### Fixed

- **Release CI release-notes extraction** — [`.github/workflows/release.yml`](.github/workflows/release.yml) was matching `## [vX.Y.Z]` against CHANGELOG headings written as `## [X.Y.Z]`, so the v0.2.0 release page fell through to the "no section matched" fallback. Strip the leading `v` from `GITHUB_REF_NAME` before grepping so future releases pick up the right CHANGELOG section automatically.

## [0.2.1] — 2026-04-28 (abandoned — Web arm64 build hang)

Tag pushed at commit `10d55af` but never produced a complete release: the `linux/arm64` Web image build hung in `npm ci` for two hours under QEMU emulation. Workflow [run 25038554573](https://github.com/AlexDuchDev/knowledge-layer/actions/runs/25038554573) was cancelled. **v0.2.2 supersedes** with the platforms fix and ships every intended change above.

The v0.2.1 git tag is preserved for audit trail. **Do not** pull `ghcr.io/<owner>/knowledge-layer-web:v0.2.1` — it does not exist. The orphan `ghcr.io/<owner>/knowledge-layer-api:v0.2.1` image (a partial side-effect of the cancelled run) carries the v0.2.2 security bumps but is not paired with a Web image; use `:v0.2.2` or `:latest` instead.

## [0.2.0] — 2026-04-27

> First **publicly published artifacts** on GHCR. The earlier `[0.1.0]` entry below documents an internal documentation/tooling milestone from 2026-04-21 — no images were published for that tag. v0.2.0 is therefore the first release operators outside the original dev environment can pull and run end-to-end. Four `-rcN` candidates surfaced and fixed install-time bugs (GHCR uppercase rejection, ExternalRef drop in source-feed validation, INSERT placeholder count mismatch in CreateSourceFeed) before this stable cut.

### Release summary

This release establishes Knowledge Layer as a **production-ready, single-tenant, self-hosted organizational memory platform** under the Apache 2.0 licence. It is the first tag intended for external operators — every code path that ships now carries the access, audit, and privacy guarantees the project was designed around, and every operator-facing surface (deployment, upgrade, rollback, secret rotation, alerting, dashboards, runbooks) ships alongside it.

**What's IN scope** — the full release contract lives in [docs/OSS_V1_SCOPE.md](docs/OSS_V1_SCOPE.md). At a glance: connector-based ingestion with raw preservation, permission-aware retrieval and Q&A, the knowledge-jobs engine, the native Control Plane (governance queues, scenarios, presets, setup wizard, roles, jobs runs, effective-access debug view), and the AI Privacy Vault that gates every outbound LLM call.

**What's OUT of scope** — multi-tenant deployment is explicitly off the table per [ADR-0014](docs/adr/0014-single-tenant-deployment-stance.md): one organization per instance. Some surfaces (Second Brain, GraphRAG) are feature-flag-gated optional modules — see *Optional modules* in [OSS_V1_SCOPE.md](docs/OSS_V1_SCOPE.md). Decision-board UI, Project Memory, additional webhook adapters, and a Second-Brain plugin contract are deferred to v1.1 / v1.2 / v2 per [LIMITATIONS.md](docs/LIMITATIONS.md).

**Highlights since the prior tag**:
- **Two production-quality webhook adapters** (Slack, Mattermost) with a shared `WebhookHandler` contract, full HMAC + replay protection on Slack, constant-time-compare token auth on Mattermost, and 15 unit tests between them — no live workspace required.
- **Five wired knowledge-job processors** (`weekly_digest`, `decision_extraction`, `planning_summary`, `stale_scan`, `support_trends_extraction`) — unsupported job types now error instead of silently completing.
- **Native Control Plane builders** for six governance queues (Reviews, Approvals, Stale, Failed jobs, Failed syncs, Policy exceptions) plus Scenarios, Presets, Setup wizard, Roles catalog, Job runs list, and Effective-access debug view.
- **Centralised publish gate** (`EntityRepo.Publish`) — one chokepoint for entity → published, with audit + version snapshot + projection update in a single transaction; PATCH on a published entity is now blocked.
- **Centralised LLM prompt registry** (`internal/ai/prompts/`) with 5 versioned templates and `prompt_template_id` threaded through the privacy gateway so every answer's prompt version is in `answer_traces`.
- **Full release CI** (`.github/workflows/release.yml`): tag-triggered, multi-arch (amd64 + arm64) GHCR publish for API + web images, with `:latest` only on stable tags.
- **Operator-facing infrastructure docs**: Kubernetes manifest set ([deploy/k8s/](deploy/k8s/)), Grafana dashboard exports ([deploy/grafana/](deploy/grafana/)), [docs/SECRET_ROTATION.md](docs/SECRET_ROTATION.md), [docs/ALERTING_PLAYBOOK.md](docs/ALERTING_PLAYBOOK.md), [docs/UPGRADE_AND_ROLLBACK.md](docs/UPGRADE_AND_ROLLBACK.md), bare-metal systemd runbook in [docs/SELF_HOSTED.md](docs/SELF_HOSTED.md).

**Where to start**:
- New evaluators → [README.md](README.md) quick-start.
- New operators → [docs/SELF_HOSTED.md](docs/SELF_HOSTED.md), then [docs/PRODUCTION_HARDENING.md](docs/PRODUCTION_HARDENING.md) and [docs/ALERTING_PLAYBOOK.md](docs/ALERTING_PLAYBOOK.md).
- Existing operators → [docs/UPGRADE_AND_ROLLBACK.md](docs/UPGRADE_AND_ROLLBACK.md) before deploying.
- Contributors → [CONTRIBUTING.md](CONTRIBUTING.md), [GOVERNANCE.md](GOVERNANCE.md), [SUPPORT.md](SUPPORT.md).

Detailed phase-by-phase changes follow.

### Fixed — RC-cycle hardening (2026-04-27)

Three install-time defects surfaced by RC validation (`docs/RC_VALIDATION.md`) and fixed pre-stable. None of these reached an external operator — that is the point of the rc loop.

- **CI: lowercase GHCR owner.** `.github/workflows/release.yml` was rejecting every multi-arch push with `invalid tag ghcr.io/<Owner>/...: repository name must be lowercase`. Compose the GHCR path at use-time from `${GITHUB_REPOSITORY_OWNER,,}`; reduce `env.{API,WEB}_IMAGE_NAME` to image-suffix only. Surfaced in v0.1.0-rc1.
- **API: `POST /source-feeds` candidate dropped `ExternalRef`.** Adapter-level config-save validation (Phase 4.2.2) built the candidate `*SourceFeed` by hand and silently dropped `ExternalRef` (and several other input fields). Effect: every filesystem / telegram / linear / asana / confluence feed creation rejected with a misleading error. Fix: introduce `CreateSourceFeedInput.CandidateSourceFeed()` that mirrors all 13 input fields; locked in with `TestCandidateSourceFeed_mirrorsAllInputFields`. Surfaced in v0.1.0-rc2.
- **API: `CreateSourceFeed` INSERT placeholder count off-by-one.** `INSERT INTO source_feeds (... 17 columns ...) VALUES ($1..$16,'draft','idle')` had 18 expressions for 17 columns; PG rejected with SQLSTATE 42601 and the row never inserted. Fix: `$1..$15` + 2 literals = 17. Locked in with new DB-backed `TestCreateSourceFeed_insertsRow` regression test that runs in the verify job's Postgres service. Surfaced in v0.1.0-rc3.

Each rc surfaced exactly one defect on the documented no-credentials evaluator path (filesystem connector → first source feed). v0.1.0-rc4 ran the full validation cleanly: POST `/source-feeds` returns 201 and writes `source_feed.created` to `audit_events`.

### Added — Follow-up pass (2026-04-25)

- **Prompt registry — full extraction** (Phase 4.1.1 follow-up): three more inline prompts moved into the central `internal/ai/prompts/templates/` registry — `ai_summarize.v1.json` (POST /ai/summarize), `ai_draft_suggestions.v1.json` (POST /ai/draft-suggestions), and `graphrag_entity_extract.v1.json` (graphrag/extract entity extraction). Each call site now uses `prompts.Get(id)` and threads `PromptTemplateID` into the privacy gateway so audit/answer-trace records the template version. Legacy `internal/graphrag/prompts/` package deleted (single-file embed loader superseded by the central registry). Test count for the registry doubles from 4 to 8 — every embedded template gets a load + key-phrase smoke check so a rename of a `.json` file fails CI loudly.
- **Mattermost webhook handler** (Phase 2.2.3 follow-up): second adapter implementing the `WebhookHandler` contract, exercising the Outgoing-Webhooks shared-token auth model (form-encoded body, constant-time token compare, 401 on mismatch). Per-feed `outgoing_webhook_token` lives in `connector_config_json` alongside the existing PAT. Sentinel errors map through the central route's status table (401 / 503 / 400) — same observability semantics as Slack. **7 unit tests** in [`adapters/mattermost/webhook_test.go`](apps/api/internal/ingestion_connectors/adapters/mattermost/webhook_test.go): valid token → artifact, bad token rejected, empty token rejected, missing config → 503, malformed form body → 400, external_ref fallback (post_id missing → channel_id:timestamp), constant-time compare smoke. No live Mattermost server required. LIMITATIONS.md and `.env.example` updated to document both adapters' config shapes.

### Added — Phase 4.2.1 + 4.3 closeout (2026-04-25)

- **Centralized publish-gate** (Phase 4.2.1 — closes audit risk #8): new [`EntityRepo.Publish(ctx, entityID, principal)`](apps/api/internal/knowledge_core/entity_repo.go) is the **single canonical path** for moving an entity to `lifecycle_state="published"`. Atomically updates the entity row (lifecycle + approval_status + approved_at + approved_by), inserts an entity_versions snapshot ("publish"), and updates entity_search_projection — all in one transaction with idempotent re-publish handling (`PublishResult{WasPublished, WasIdempotent}`). Returns `ErrAlreadyPublished` semantics via the `WasIdempotent` flag rather than an error.
- **PATCH guard**: `EntityRepo.Patch` now rejects `lifecycle_state="published"` with `ErrPatchPublishForbidden`. `PATCH /entities/:id` maps it to a 400 directing the caller to the new publish endpoint. Eliminates the previously unguarded path where any caller with the `edit` action could effectively publish via PATCH.
- **New route**: `POST /entities/:id/publish` — guarded by `AccessEvaluator.Evaluate(action="publish")`. Calls `EntityRepo.Publish`, fires `Search.ReindexEntity`, emits `entity.published` audit event with reason="direct". Returns the updated entity plus `was_published`/`was_idempotent` flags.
- **Refactored `POST /review-tasks/:id/approve`**: previously did inline `UPDATE entities SET lifecycle_state='published'`. Now calls `EntityRepo.Publish` and emits `entity.published` audit with reason="review_approval". The version snapshot + projection update + audit emission now happen via the same path as direct publish — operators get one chokepoint to monitor.
- **Grafana dashboard exports** (Phase 4.3): new [`deploy/grafana/`](deploy/grafana/) directory with `knowledge-layer-overview.json` (14 panels: API request rate + 5xx rate by route, duration by job_type and adapter_kind, Asynq queue depths + failed counts, Postgres pool stats + acquire rate, Go runtime). Dashboard uses `$datasource` and `$job` variables so it imports cleanly across instances. README documents three import paths (UI, API, kube-prometheus-stack sidecar) and links to the alerting playbook for matching alert rules.

### Added — Phase 4.1.1 prompt registry (2026-04-25)

- **Versioned LLM prompt registry** at [`apps/api/internal/ai/prompts/`](apps/api/internal/ai/prompts/) — central package with embed-loaded JSON templates under `templates/<id>.json`. Each template carries a stable id (`ask_global_qa.v1`), a description that documents the version-bump rule ("bump to .v2 when output shape, citation rules, or evidence handling change — never repurpose"), a system_prompt, and an optional user_template with `{{name}}` substitution. Loader is sync.Once-cached; unknown placeholders are left as literals so a typo in `Render(params)` is caught in tests instead of silently dropping content.
- **Reference templates extracted from the inline `buildSystemPrompt`** in `qa/synthesize.go`: [`ask_global_qa.v1.json`](apps/api/internal/ai/prompts/templates/ask_global_qa.v1.json) and [`ask_global_qa_best_trusted.v1.json`](apps/api/internal/ai/prompts/templates/ask_global_qa_best_trusted.v1.json). The Q&A synthesis path now reads from the registry and threads the template id through `privacy.InvokeInput.PromptTemplateID` → `privacy_trace.prompt_template_id`, so `answer_traces.privacy_json` records exactly which template produced each answer.
- **Audit-trail field**: `InvokeInput.PromptTemplateID` (new, optional) and `prompt_template_id` in the gateway's `privacy_trace` JSON. Empty value = legacy inline prompt (acceptable during the gradual extraction); non-empty = traceable to a registry entry.
- **Tests**: 5-case unit suite for the registry — load, variant load, unknown id, placeholder leak detection, ID enumeration. Integration tests across qa / retrieval_intelligence remain green.
- Pattern documented in [`apps/api/internal/README.md`](apps/api/internal/README.md) "where to add new code" — adding an AI flow now means dropping a `.json` template + referencing via `prompts.Get(id)`.

### Added — Phase 4 hardening, second pass (2026-04-25)

- **[ADR-0014: Single-tenant deployment stance](docs/adr/0014-single-tenant-deployment-stance.md)** — accepts the long-implicit single-tenant contract as a stable architectural decision. Codifies "no `tenant_id`, no tenant routing, no shared-cluster isolation primitives", documents the three rejected alternatives (soft-tenancy via domain reuse, `tenant_id` on every table, schema-per-tenant), and specifies the revisit gate (any future multi-tenant ADR must include a worked schema-migration plan + per-step access-pipeline review). LIMITATIONS.md gets a new row pointing to it.
- **Connector validation at config-save** (Phase 4.2.2): new `Service.ValidateSourceFeed(ctx, *SourceFeed)` runs adapter-level validation against a candidate feed and is now called inline from `POST /source-feeds` (before insert) and `PATCH /source-feeds/:id` (against an overlay of the existing feed + new config — no DB write if validation fails). New `POST /source-feeds/validate` endpoint runs pure dry-run validation for the source-feed wizard to surface inline errors before commit. Bad configs are rejected at save time rather than at the first sync attempt.
- **AI gateway lint-rule** (Phase 4.1.2): new [`apps/api/.golangci.yml`](apps/api/.golangci.yml) with a depguard rule that blocks unsanctioned imports of `internal/llm`. Allowed importers (current legitimate set: ai/privacy, qa, embeddings per ADR-0013, graphrag/extract legacy, app, httpserver, cmd/connectorworker, cmd/api) are explicit; any new package importing `llm` triggers the rule. Plus opinionated checks (errcheck, ineffassign, misspell, unused). Not yet a CI gate — ships as documentation enforcement; CONTRIBUTING.md instructs contributors to run `golangci-lint run ./...` locally before PR. Aligns with the existing "Things to avoid" guidance in `apps/api/internal/README.md`.

### Added — Phase 3.4 + Phase 4 first-pass (2026-04-25)

- **Kubernetes deployment example** at [`deploy/k8s/`](deploy/k8s/) — minimum-viable manifest set: `namespace.yaml`, `configmap.yaml`, `secret.example.yaml` (template — not for commit), API + jobworker + connectorworker + web Deployments with rolling updates, hardened security context (runAsNonRoot, dropped capabilities, readOnlyRootFilesystem where applicable), liveness + readiness + startup probes hitting `/health`, and an example `ingress.example.yaml` with the full route prefix list. README walks operators through `kubectl apply` order, secret-creation alternatives, port-forward verification, and a production checklist that maps to PRODUCTION_HARDENING.md. Closes the "DigitalOcean is the only deployment example" gap from the system audit.
- **Release CI workflow** at [`.github/workflows/release.yml`](.github/workflows/release.yml) — fires on `v*` tag push, re-runs lint/typecheck/tests/build on the tagged SHA before publishing (a tag pointing at an old commit can't ship a broken image), then builds and pushes multi-arch (amd64 + arm64) `knowledge-layer-api` and `knowledge-layer-web` images to GHCR with `vX.Y.Z` and conditional `:latest` tags (stable releases only — pre-releases like `v0.X.Y-rc1` skip `:latest`). Creates a GitHub Release with the matching CHANGELOG section as the body, marks `-rcN` tags as pre-release. [`docs/RELEASING.md`](docs/RELEASING.md) updated with the automated tag-to-release flow.
- **[`docs/SECRET_ROTATION.md`](docs/SECRET_ROTATION.md)** — every secret Knowledge Layer holds, with rotation cadence, dual-token windows for `OPS_AUTH_TOKEN`, the three-path strategy for `AI_PRIVACY_VAULT_KEY` rotation (drain + rotate vs in-place re-encryption vs forbidden plaintext fallback), per-feed connector secrets via PATCH, mandatory verification checklist after every rotation.
- **[`docs/ALERTING_PLAYBOOK.md`](docs/ALERTING_PLAYBOOK.md)** — the seven alerts you almost certainly want, full Prometheus rule snippets for API liveness + 5xx rate, worker stalled vs queue backlog vs failure rate, job p95 + failure rate, connector sync failure rate, Postgres pool saturation/exhaustion, kube-prometheus-stack `PrometheusRule` + `ServiceMonitor` skeletons, audit-event monitoring as a separate compliance channel.

### Added — Phase 3 release readiness (2026-04-25)

- **Code-level READMEs**: new [`apps/api/internal/README.md`](apps/api/internal/README.md) is the canonical module map (every package, layering rules, where-to-add-X table). New [`packages/shared/README.md`](packages/shared/README.md) documents the contract surface. Expanded [`apps/api/README.md`](apps/api/README.md) and [`apps/web/README.md`](apps/web/README.md) with quick-start, layout, env, conventions, boundaries, and per-area docs links — external developers can navigate without grepping.
- **Issue templates**: [`.github/ISSUE_TEMPLATE/bug_report.md`](.github/ISSUE_TEMPLATE/bug_report.md), [`feature_request.md`](.github/ISSUE_TEMPLATE/feature_request.md), [`docs_improvement.md`](.github/ISSUE_TEMPLATE/docs_improvement.md), plus a `config.yml` that disables blank issues and surfaces the SECURITY.md link. Feature template carries a five-question governance checklist (access / audit / privacy / provenance / failure-mode) so triage gets the right inputs up-front.
- **Root [`CONTRIBUTING.md`](CONTRIBUTING.md) expansion**: TL;DR clone-to-PR script, ordered "before you start" reading list, local-development table, verification commands (`make lint typecheck test`), PR conventions (one logical change, doc-impact mandatory, ADRs for behavioural changes), filing-issues guidance, optional-modules contract, semver pointer.
- **[`docs/UPGRADE_AND_ROLLBACK.md`](docs/UPGRADE_AND_ROLLBACK.md)**: zero-downtime rolling upgrade for multi-pod deployments + single-pod path for compose/bare-metal; compatibility matrix (HTTP API, schema, queue payloads, audit events); three rollback strategies (code-only / backward-compatible migration / destructive); Postgres-vs-Redis consistency hazards; mandatory + recommended post-upgrade verification checklist; known upgrade gotchas (Phase 1.1.2 vault-key requirement, dirty-migration recovery, Phase 2.1.5 setup URL change).
- **Bare-metal runbook in [`docs/SELF_HOSTED.md`](docs/SELF_HOSTED.md)**: ten-step Linux/systemd deployment — system user + dirs, Postgres + pgvector + pgcrypto, build path, env file with production-hardening rules, three systemd unit files (API + jobworker + connectorworker) with hardening flags, journald log retention, web-app unit with reverse-proxy guidance, scrape endpoints incl. worker `/ops/health`, off-host backup cron, upgrade short-form referencing the new runbook.

### Added — Phase 2 increments (2026-04-25, fifth pass)

- **Slack webhook pilot** (Phase 2.2.3 — OSS, no live workspace required): new `WebhookHandler` contract on `ConnectorAdapter` (`HandleWebhook(ctx, WebhookRequest) (*WebhookResult, error)`), Slack adapter implements full Events API verification ([adapters/slack/webhook.go](apps/api/internal/ingestion_connectors/adapters/slack/webhook.go)) — HMAC-SHA256 over `"v0:{ts}:{body}"`, 5-minute replay window, URL-verification challenge handshake, dedup via Slack `event_id`. Per-feed `signing_secret` lives in `connector_config_json`. New HTTP route `POST /connectors/webhook/:adapter_kind/:source_feed_id` with 1 MiB body cap, body-copy before adapter invocation, sentinel-error → HTTP-status mapping (401 bad sig / stale ts / missing headers, 503 missing secret, 400 malformed). New `Service.IngestWebhookResult` persists artifacts under a `transport=webhook` ingestion run with same dedup as polled syncs and inline-normalises via `ProcessQueuedRawArtifact`. **8 unit tests** in [webhook_test.go](apps/api/internal/ingestion_connectors/adapters/slack/webhook_test.go) cover URL verification, event_callback → artifact, bad signature, stale timestamp replay, missing headers, missing secret, case-insensitive header lookup, and unknown-but-signed envelope acceptance — `nowFn` is overridden so signature/replay assertions are deterministic. Operators can curl-replay these fixtures against a local instance with the documented test signing secret.
- **Native CP Setup wizard** (Phase 2.1.5): `/control-plane/setup` (hub: templates + recent sessions, "New session" CTA) and `/control-plane/setup/session/[id]` (5-step wizard: pick template → toggle connector families → assign initial admin → preview → launch). Each step commits independently; preview surfaces validation issues; launch is gated on a clean preview. Backed by `/api/onboarding/*`. New shared clients [SetupHubClient](apps/web/src/components/control-plane/SetupHubClient.tsx) and [SetupSessionWizardClient](apps/web/src/components/control-plane/SetupSessionWizardClient.tsx). Rewrite for `/control-plane/setup/session/:id` removed from `next.config.ts`; legacy `apps/web/src/app/(dash)/admin/setup/` deleted. Existing `SetupControlPlaneClient.tsx` had broken `/onboarding/...` API paths (missing `/api` prefix); fixed in-place via `sed` so the unmounted sub-pages (templates / launch-preview / launch-result / session/new) work correctly when reached directly.

### Added — Phase 2 increments (2026-04-25, fourth pass)

- **Native CP Governance queues** (Phase 2.1.x — six pages promoted from `CpScaffold` stubs to working operator views): Reviews (`/control-plane/governance/reviews`), Approvals (`/control-plane/governance/approvals`), Stale (`/control-plane/governance/stale`), Failed jobs (`/control-plane/governance/failed-jobs`), Failed syncs (`/control-plane/governance/failed-syncs`), Policy exceptions (`/control-plane/governance/policy-exceptions`). Backed by existing APIs: `/review-tasks`, `/governance/approval-queue`, `/governance/stale-content`, `/ops/failed-runs`, `/governance/policy-exceptions`. New shared module [GovernanceQueueClients](apps/web/src/components/control-plane/GovernanceQueueClients.tsx) provides 6 thin client components with a common `QueueShell` for load/refresh/error state and tree-shakeable per-page imports. CP governance hub page now lists the six queues as cards instead of just cross-links.
- **Bulk-action affordances** intentionally NOT wired in this iteration — queues are read-only triage views; mutations (approve/dismiss/retry) remain on per-item entity workflow tools and the existing mutation API endpoints.

### Added — Phase 2 increments (2026-04-25, third pass)

- **Native CP Scenarios builder** (Phase 2.1.2): `/control-plane/scenarios` (catalog, list + presets, create-from-preset form) and `/control-plane/scenarios/[id]` (detail with definition + preview JSON, builder section outline) now render natively. Removed the `next.config.ts` rewrite for `/control-plane/scenarios` and the `middleware.ts` rewrite for `/control-plane/scenarios/[id]`; deleted `apps/web/src/app/(dash)/admin/scenarios/`. Inbound 308 redirects from `/admin/scenarios` retained.
- **Native CP Presets catalog** (Phase 2.1.4): `/control-plane/presets` (filterable list with type/category axis filters) and `/control-plane/presets/[id]` (entry, categories, related presets, preview, instantiate form) now render natively. Removed both `next.config.ts` rewrite and `middleware.ts` matcher for `/control-plane/presets/*`; deleted `apps/web/src/app/(dash)/admin/presets/`. The `middleware.ts` matcher list is now down to a single entry (`/control-plane/jobs/:path*`).

### Fixed

- **`/source-feeds` production build**: wrapped `useSearchParams()` in a `Suspense` boundary so Next.js 15 prerender succeeds (`SourceFeedsPageInner` + outer `SourceFeedsPage`). Same pattern as `search`, `invite`, `login`, and `entities/[id]` already use. Unblocks `npm run build` end-to-end.

### Added — Phase 2 increments (2026-04-24, second pass)

- **Explore from here** (Phase 2.3.1): bounded one-hop entity-level traversal of the GraphRAG co-mention graph with permission filtering. New API `GET /entities/:id/graph-explore?max_nodes=…` (returns `{neighbours, denied_count, truncated}`); new web page `/entities/[id]/explore`; "Explore from here" link added to entity detail header. New Neo4j repo method `RelatedEntitiesByCoMention`. Backed by the canView pipeline from Phase 1.1.1, so denied neighbours never reach the response. Returns 503 when `NEO4J_URL` is unset.
- **Native CP Roles catalog** (Phase 2.1.1): `/control-plane/roles` now renders a CP-native catalog UI directly (lists roles + presets, on-select shows detail / access preview / assignments JSON panes). Removed the `next.config.ts` rewrite to legacy `/admin/roles` and deleted `apps/web/src/app/(dash)/admin/roles/page.tsx`. The `/admin/roles → /control-plane/roles` 308 redirect remains for inbound legacy URLs.

### Added — Phase 2 increments (2026-04-24)

- **Worker `/ops/health` endpoints** for `cmd/jobworker` (`:9001` default) and `cmd/connectorworker` (`:9002` default). Returns DB/Redis liveness, per-Asynq-queue depth, and last-processed timestamp per task type — the canonical "is this worker stuck or just idle" signal. Public `/health` always returns liveness; `/ops/health` is bearer-gated outside `APP_ENV=local` using the same `OPS_AUTH_TOKEN` as the API. New package: [apps/api/internal/workerhealth](apps/api/internal/workerhealth) with unit tests for the per-task tracker.
- **`/metrics` enrichment** with new collectors registered in the shared [internal/platform/metrics](apps/api/internal/platform/metrics) package: `knowledge_job_run_duration_seconds{job_type,status}` (recorded inline by `knowledge_jobs.runOrchestrator`), `connector_sync_duration_seconds{adapter_kind,status}` (recorded by `ingestion_connectors/app.SyncOrchestrator.RunSync`), on-scrape `postgres_pool_*` gauges from `pgxpool.Stat()`, and `asynq_queue_*{queue}` gauges via Asynq Inspector. The `httpserver` package now reuses the same registry instead of maintaining a private one.
- **Effective-access UI** (Phase 2.1.6) at `/control-plane/users/[id]/access`: new `GET /users/:id/effective-access` API endpoint surfaces the 9-step `AccessEvaluator` trace (normally `json:"-"`) so operators can debug "why can't user X view entity Y?" Auth: identity admin or self.
- **Job runs list** (Phase 2.1.3) at `/control-plane/jobs/runs`: new `GET /knowledge-jobs/runs?status&job_type&limit` endpoint with `JobRunListing` projection joining `job_runs` ↔ `knowledge_jobs` for the table view (no N+1 fetch).

### Changed — Phase 2 increments

- `cmd/jobworker` and `cmd/connectorworker` instrument all task handlers with `workerhealth.Tracker.Wrap(...)` so `/ops/health` carries a per-task-type "last successful completion" timestamp.
- `internal/app/deps.go` registers `metrics.RegisterPoolStats(pool)` and `metrics.RegisterQueueDepth(inspector, queues)` once at startup; `/metrics` exposes them on every scrape.
- `internal/secondbrain/prebrief.go`: removed unreachable `return nil` after the `for {}` polling loop (pre-existing `go vet` warning that blocked `make lint`).

### Added — Phase 1 alignment (2026-04-24)

- **AI Privacy Vault audit-events**: `vault.placeholder_stored`, `vault.placeholder_decrypted`, `vault.rehydration_applied` written to `audit_events` for every encrypt/decrypt/rehydrate (`apps/api/internal/ai/privacy/vault_store.go`, `rehydrate.go`).
- **GraphRAG permission filter**: `graphExpandContextPieces` now filters expanded chunks through the same `canView` callback as seed entities — closing the access-before-retrieval leak where co-mention chunks could reach LLM context for denied entities ([apps/api/internal/retrieval_intelligence/service.go](apps/api/internal/retrieval_intelligence/service.go), regression test in `graph_expansion_permission_test.go`).
- **API stability policy** ([docs/API_STABILITY.md](docs/API_STABILITY.md)) — declares v0.x breaking-change tolerance and v1.0 semver/`/v1/...` contract.
- **OSS_V1_SCOPE.md "Optional modules"** section documenting Second Brain and GraphRAG as feature-flag-gated.
- **Honest LIMITATIONS.md entries** for Decision UI, Project Memory, Setup wizard, CP native builders, Effective-access UI, connector webhooks.

### Changed — Phase 1 alignment

- **Production fail-closed for AI Privacy Vault**: `ValidateAPI` and `ValidateWorker` now require `AI_PRIVACY_VAULT_KEY` (≥32 bytes) and forbid `AI_PRIVACY_DEV_PLAINTEXT_STORE=1` when `APP_ENV=production` (`apps/api/internal/config/hardening.go`). Composition root in `internal/app/deps.go` panics on vault init failure in production.
- **Permission-check pulled inside `retrieval_intelligence` façade**: `AskGlobal` and `AskEntity` no longer accept a `canView` callback parameter. The façade builds it internally via the centralized 9-step `AccessEvaluator`, so a new caller cannot accidentally omit the check (`apps/api/internal/retrieval_intelligence/service.go`).
- **ADR-0012 (Second Brain)** promoted from Provisional to **Accepted** with feature-flag gating (`SECOND_BRAIN_PREBRIEF_TICK`, `TELEGRAM_BOT_TOKEN`).
- **README highlight** clarifies self-hosted single-tenant positioning (NOT a multi-tenant SaaS).
- **PRODUCTION_HARDENING.md** documents vault key requirement and audit-event coverage.
- **Env-table single source of truth**: `DEPLOY_CHECKLIST.md` and `PRODUCTION_GO_LIVE_CHECKLIST.md` now defer to `.env.example` + `CONFIG_ENV.md` rather than maintain parallel tables.

### Removed — Phase 1 alignment

- **Orphan stub task handlers** `TaskAISummarize` and `TaskGovernanceFollowUp` (and their payload structs) — they were registered in `cmd/connectorworker` but never enqueued anywhere; the silent-success handler ate work that the API thought it had dispatched. If AI summarization is reintroduced later, it must come with a real implementation.
- **Empty placeholder packages** `internal/platform_operations`, `internal/workflow_governance`, `internal/events` (each was a single `doc.go` line with no consumers).
- **Stale duplicate doc** `docs/Config and environments.md` — replaced with one-line redirect to canonical `docs/CONFIG_ENV.md`.

### Added

- API startup **auto-bootstrap** when the database has zero domains: enabled by default for `APP_ENV=local` (override with `AUTO_BOOTSTRAP_INSTANCE=0`); staging/production require explicit `AUTO_BOOTSTRAP_INSTANCE=1` plus `BOOTSTRAP_ADMIN_EMAIL` / `BOOTSTRAP_ADMIN_PASSWORD` ([docs/CONFIG_ENV.md](docs/CONFIG_ENV.md), `apps/api/internal/instancebootstrap`).
- [docs/EXTERNAL_DEV_QUICKSTART.md](docs/EXTERNAL_DEV_QUICKSTART.md) and [docs/LIMITATIONS.md](docs/LIMITATIONS.md) for OSS expectations; IA policy for canonical `(dash)/` vs `/app/*` redirects ([docs/INFORMATION_ARCHITECTURE_V1.md](docs/INFORMATION_ARCHITECTURE_V1.md)).
- Merged legacy [docs/Domain model.md](docs/Domain%20model.md) narrative into [docs/DOMAIN_MODEL.md](docs/DOMAIN_MODEL.md); stub redirect file keeps old path alive.

### Changed

- Knowledge job orchestrator: unsupported `job_type` now returns an error instead of completing with no work ([apps/api/internal/knowledge_jobs/orchestrator.go](apps/api/internal/knowledge_jobs/orchestrator.go)).
- [Connector Framework Specification.md](docs/Connector%20Framework%20Specification.md) §3.1 aligned with glossary (connector kind vs source feed instance).

- MemPalace-inspired **governed** patterns (docs only where noted): scope-first retrieval and internal benchmark stance ([docs/AI_RETRIEVAL_GOVERNANCE.md](docs/AI_RETRIEVAL_GOVERNANCE.md)); Search/entity **explorer UX** spec ([docs/SEARCH_AND_QA_UX.md](docs/SEARCH_AND_QA_UX.md), [docs/INFORMATION_ARCHITECTURE_PRODUCT_V1.md](docs/INFORMATION_ARCHITECTURE_PRODUCT_V1.md)); self-hosted capability matrix ([docs/SELF_HOSTED.md](docs/SELF_HOSTED.md), [docs/CONFIG_ENV.md](docs/CONFIG_ENV.md)); transcript/mega-file splitting ([docs/INGESTION_AND_CONNECTORS.md](docs/INGESTION_AND_CONNECTORS.md) §12.7, [docs/KNOWLEDGE_JOBS.md](docs/KNOWLEDGE_JOBS.md) §25.4).
- `GET /entities/:id/related` supports `depth=2` (bounded 2-hop `entity_links` with per-entity `view` checks). Entity detail UI: **Explore from here** rail using `related?limit=12&depth=2` ([docs/API_SURFACE_V1.md](docs/API_SURFACE_V1.md)).

- Apache-2.0 `LICENSE`, `CONTRIBUTING.md`, `SECURITY.md`.
- Self-hosted documentation: `docs/SELF_HOSTED.md`, `docs/OPERATIONS.md`, `docs/ARCHITECTURE_HOST.md`, `docs/CONFIG_ENV.md`.
- User and administrator guides: `docs/USER_GUIDE_V1.md`, `docs/ADMIN_GUIDE_V1.md`.
- Auth: `AUTH_MODE` (`development_header` | `session`), `SESSION_SECRET`, signed cookie `kl_session`.
- `POST /instance/bootstrap`, `GET /instance/status` for first workspace setup.
- `POST /auth/login`, `POST /auth/logout`, `GET /auth/me`.
- Invitations: `POST /invitations`, `GET /invitations/preview`, `POST /invitations/accept`.
- `POST /users/import` (CSV, `invite` or `active` mode, optional `send_invites`).
- `POST /domains`, `PATCH /domains/:id`.
- `GET /settings/instance`, `POST /settings/test-mail`.
- Web routes: `/login`, `/invite`, `/bootstrap`, `/settings`; home administrator checklist and nav links.
- Migration `000014_auth_invitations` (`password_hash`, `user_invitations`).
- CORS credentials and `CORS_ALLOW_ORIGINS`; `NEXT_PUBLIC_USE_DEV_HEADER` for pilot header mode.

## [0.1.0] - 2026-04-21

Documentation and operator tooling for **production cutover** and **open-source hygiene** (no breaking API contract change in this tag).

### Added

- Session-based operator smoke: [`scripts/smoke-session.sh`](scripts/smoke-session.sh) (documented in [docs/STAGING_SMOKE_TEST.md](docs/STAGING_SMOKE_TEST.md)); production cutover quick ref [docs/PRODUCTION_CUTOVER_QUICKREF.md](docs/PRODUCTION_CUTOVER_QUICKREF.md); infra inventory [docs/INFRA_PRODUCTION_REFERENCE.md](docs/INFRA_PRODUCTION_REFERENCE.md).
- Release process [docs/RELEASING.md](docs/RELEASING.md); pre-push heuristic [`scripts/repo-sanity-check.sh`](scripts/repo-sanity-check.sh); [NOTICE](NOTICE), [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md); SLO starter [docs/SLO_AND_ALERTING_TEMPLATE.md](docs/SLO_AND_ALERTING_TEMPLATE.md).
