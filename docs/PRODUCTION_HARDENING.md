# Production hardening

How the API and workers behave in **staging** and **production** (`APP_ENV=staging` or `APP_ENV=production`). For local compose, keep `APP_ENV=local`. See also [DEPLOY_CHECKLIST.md](DEPLOY_CHECKLIST.md) and [ACCESS_MODEL.md](ACCESS_MODEL.md).

## 1. Auth and session requirements

| Rule | Enforcement |
|------|-------------|
| `AUTH_MODE` must be **`session`** | Startup: [`ValidateAPI`](../apps/api/internal/config/hardening.go) rejects `development_header` and any other value in staging/production. |
| `SESSION_SECRET` required | Must yield ≥16 bytes via [`SessionSecretBytes`](../apps/api/internal/config/config.go) (raw or hex). Missing/short secret prevents startup in staging/production. |
| No `X-Principal-User-ID` impersonation | [`principalMiddleware`](../apps/api/internal/httpserver/middleware.go) enables the dev header only when `AUTH_MODE=development_header` **and** `IsLocalDev()` (not staging/production). |
| Session cookies | [`IssueSessionCookie`](../apps/api/internal/auth/session.go) sets `Secure` when [`SessionCookieSecure`](../apps/api/internal/config/hardening.go) is true (default **true** in staging/production). Set `SESSION_COOKIE_SECURE=false` only for HTTP-only lab environments. |

## 2. Required secrets and unsafe defaults to avoid

**API (staging/production)** — startup fails unless:

- `DATABASE_URL` is **set in the environment** (empty env var is rejected even if a default DSN exists in code).
- `REDIS_URL` is **set in the environment** (required for API and queue-backed behavior in non-local).
- `CORS_ALLOW_ORIGINS` is **set** (do not rely on the localhost default).
- `SESSION_SECRET` valid when using session auth (mandatory in these profiles).
- `OPENSEARCH_URL` uses **`https://`** when non-empty, unless `OPENSEARCH_ALLOW_INSECURE_HTTP=1` is explicitly set for a private network.

**API (production only)** — additionally:

- `OPS_AUTH_TOKEN` must be set in the environment with length ≥ **16** (trimmed) so `/ops/health` and `/metrics` are not anonymously reachable with full or stub detail.
- `APP_PUBLIC_URL` must start with **`https://`**.
- `SESSION_COOKIE_SECURE` must not be disabled (`false`/`0`/`no` rejected).
- **AI Privacy Vault**: `AI_PRIVACY_VAULT_KEY` is **required** (32 raw bytes or base64 of 32 bytes for AES-256) and `AI_PRIVACY_DEV_PLAINTEXT_STORE=1` is **forbidden**. The vault encrypts placeholder cleartext used by sanitize/rehydrate flows; running production with plaintext-at-rest defeats the privacy boundary. Both API and workers fail-start without these (`ValidateAPI` / `ValidateWorker`).

**Workers** — `DATABASE_URL`, `REDIS_URL`, and OpenSearch HTTPS rule when `APP_ENV` is staging/production ([`ValidateWorker`](../apps/api/internal/config/hardening.go)).

**Do not**:

- Expose the API on the public internet with `APP_ENV=local` or `AUTH_MODE=development_header`.
- Commit real `SESSION_SECRET`, `OPS_AUTH_TOKEN`, or DB passwords.
- Run production against the compose OpenSearch image with `plugins.security.disabled=true` (local only).

## 3. Ops endpoint exposure rules

| Endpoint | Local / dev (`APP_ENV` not staging/production) | Staging | Production |
|----------|-----------------------------------------------|---------|------------|
| `GET /health` | Public `{"status":"ok"}` | Same | Same |
| `GET /ops/health` | Full JSON including raw DB/OpenSearch error strings | If `OPS_AUTH_TOKEN` **unset**: **200** redacted booleans only (no raw errors). If token **set**: **401** without bearer; bearer → full detail. | `OPS_AUTH_TOKEN` **required** at startup → **401** without bearer; bearer → full detail. |
| `GET /metrics` | Prometheus text/OpenMetrics (Go + process + `http_requests_total` by method/route template) without auth | **401** unless `OPS_AUTH_TOKEN` is non-empty **and** bearer matches (staging with no token → always **401**). | Same (token always present). |
| `GET /ops/failed-runs` | Requires authenticated principal + identity admin gate | Unchanged | Unchanged |

Implementations: [`routes_health.go`](../apps/api/internal/httpserver/routes_health.go) (metrics + HTTP counter), [`routes.go`](../apps/api/internal/httpserver/routes.go) (`PrometheusHTTPRequestsMiddleware`). `/metrics` uses a **dedicated Prometheus registry** so it does not collide with the global default registry used elsewhere in the process.

## 4. OpenSearch security expectations

- Prefer **TLS** to the cluster (`https://`). Plain `http://` is rejected at startup for staging/production unless `OPENSEARCH_ALLOW_INSECURE_HTTP=1`.
- Run OpenSearch with **authentication and TLS** in real environments; compose’s single-node insecure profile is for developer machines only.
- Workers and API share the same URL validation so misconfiguration fails closed before processing jobs.

## 5. Staging vs production differences

**Shared (`IsNonLocal()`):** `AUTH_MODE=session`, explicit `DATABASE_URL`, `REDIS_URL`, `CORS_ALLOW_ORIGINS`, `SESSION_SECRET`, OpenSearch HTTPS rule.

**Production-only (`IsProduction()`):** `OPS_AUTH_TOKEN` (min 16 chars), `APP_PUBLIC_URL` must be `https://`, `SESSION_COOKIE_SECURE` cannot be turned off, `AI_PRIVACY_VAULT_KEY` required (≥32 bytes) and `AI_PRIVACY_DEV_PLAINTEXT_STORE=1` forbidden.

| Concern | Staging | Production |
|---------|---------|------------|
| Code gates | As above | Plus token, HTTPS public URL, secure cookies |
| `/ops/health` anonymous | Redacted **if** `OPS_AUTH_TOKEN` unset | Always **401** without bearer (token required) |
| Data | Non-production datasets, may tolerate softer SLOs | Production data, stricter change control |
| Secrets | Rotate independently; may share shape with prod | Unique secrets; rotation policy |
| TLS | HTTPS expected; may use internal CAs | Public CA or approved internal PKI |
| Observability | Protect bearer token; metrics need token to scrape | Same |

## 6. Final production go-live checks

Use the full operator list in [PRODUCTION_GO_LIVE_CHECKLIST.md](PRODUCTION_GO_LIVE_CHECKLIST.md). Summary:

- [ ] `APP_ENV=production`, `AUTH_MODE=session`, strong `SESSION_SECRET` (store in secret manager).
- [ ] `DATABASE_URL`, `REDIS_URL`, `CORS_ALLOW_ORIGINS`, `APP_PUBLIC_URL` (`https://`) set explicitly.
- [ ] `OPENSEARCH_URL` is `https://` or `OPENSEARCH_ALLOW_INSECURE_HTTP=1` is consciously set for private mesh only.
- [ ] `OPS_AUTH_TOKEN` ≥ 16 characters; document who may call `/ops/health` and `/metrics` with bearer auth.
- [ ] `SESSION_COOKIE_SECURE` not disabled; HTTPS termination in front of the API.
- [ ] `AI_PRIVACY_VAULT_KEY` set (32-byte AES-256 key) and `AI_PRIVACY_DEV_PLAINTEXT_STORE` unset; vault audit-events (`vault.placeholder_stored`, `vault.placeholder_decrypted`, `vault.rehydration_applied`) appear in `audit_events` after first Ask.
- [ ] Web app: `NEXT_PUBLIC_USE_DEV_HEADER=false`; users authenticate via session login.
- [ ] Run smoke adapted for session auth plus ops/metrics bearer checks.
- [ ] Review [RELEASE_READINESS_AUDIT.md](RELEASE_READINESS_AUDIT.md) for remaining product gaps (blob store, Prometheus handler vs stub, etc.).

## 12. L1 cache (v0.4.0+, optional)

`CACHE_L1_ENABLED=true` enables an in-process BigCache for hot read paths (`/domains`, `/users/:id/effective-access`, `/search` keyword, `/knowledge-jobs/engine-metadata`). The cache is principal-scoped — every key embeds the requesting user — and is invalidated on `entity.published`, `role.granted`, `policy.updated`, `feed.config_patched`. **Authorization decisions are not cached**: every authz call still runs through `AccessEvaluator.Evaluate`. The cache only stores already-decided responses for one TTL window (default 60s).

The `/users/:id/effective-access` endpoint is the most subtle: an operator who just revoked a role for a user may see the old "allowed" UI for up to TTL seconds. The user themselves is not at risk — the next privileged operation runs `AccessEvaluator` synchronously and is denied immediately. To minimise the window, lower `CACHE_L1_TTL_SECONDS`. To eliminate it, leave `CACHE_L1_ENABLED=false`.

## 13. OAuth 2.1 proxy (v0.5.0+, optional)

`OAUTH_PROXY_ENABLED=true` mounts `/.well-known/oauth-authorization-server` and `/oauth/{authorize,token,register,callback}` to front an operator-supplied OIDC issuer. The proxy is a prerequisite for the v0.5.1 MCP endpoint. Per [ADR-0015](adr/0015-oauth-proxy-and-mcp-bridge.md) the proxy preserves the single-tenant invariant (no tenant claim in tokens) and the AccessEvaluator-mandatory contract (every issued bearer maps to a real `users.id`; admin bypass tokens are rejected).

| Rule | Enforcement |
|------|-------------|
| `OAUTH_SECRET_KEY` ≥ 32 bytes | [`ValidateAPI`](../apps/api/internal/config/hardening.go) rejects shorter keys at startup. Signs OAuth state HMAC + JWT bearers. |
| `OIDC_ISSUER_URL` reachable | OIDC discovery runs at startup. Production: failure aborts the proxy. Local/staging: logs and continues without OAuth. |
| `OIDC_CLIENT_ID` set | Required even when the IDP allows public clients — KL acts as a confidential client at the IDP. |
| `OIDC_CLIENT_SECRET` set in production | `ValidateAPI` rejects empty `OIDC_CLIENT_SECRET` when `APP_ENV=production`. Public/PKCE-only flows are an MCP-side concern; our proxy is confidential at the IDP. |
| `OIDC_SUB_COLUMN` is `email` or `external_idp_subject` | Anything else rejected. Default `email`. Operators with opaque OIDC subjects should add the column and set this to `external_idp_subject`. |
| `redirect_uris` allow-list per dynamic-registered client | `Server.callback` rejects callbacks whose `redirect_uri` doesn't match an `oauth_clients.redirect_uris` entry. No path-prefix matching. |

**Single-instance assumption.** Auth codes live 2 minutes in `sync.Map`. Multi-pod deployments need sticky LB on `/oauth/*` so the callback hits the same pod that issued the code. ADR-0014 documents the rationale.

**Token rotation.** JWTs live 1 h. To force-revoke every in-flight bearer, rotate `OAUTH_SECRET_KEY` ([SECRET_ROTATION.md §9](SECRET_ROTATION.md)). Per-client revocation: `UPDATE oauth_clients SET revoked_at=now() WHERE client_id=...`.

**Audit observability.** `oauth.client.registered` and `oauth.token.issued` log to **stderr** in v0.5.x — promotion to `audit_events` is a v0.5.x point release. For now, parse container logs.

## 14. MCP endpoint (v0.5.1+, optional)

`MCP_ENABLED=true` mounts `/mcp` (Model Context Protocol streamable HTTP). External AI clients (Claude Desktop, Cursor) speak MCP after authenticating through the OAuth proxy.

| Rule | Enforcement |
|------|-------------|
| `MCP_ENABLED=true` requires `OAUTH_PROXY_ENABLED=true` | `ValidateAPI` rejects the combination. The /mcp endpoint cannot validate bearers without the proxy. |
| Every tool wrapped in `withAccessGuard` | Static-grep contract test [`TestNew_allToolsAccessGuarded`](../apps/api/internal/mcp/server_test.go) iterates the registry; CI fails if a new tool bypasses. No admin bypass. |
| Bearer absent → 401 with `WWW-Authenticate: Bearer realm="mcp"` | [`bearerMiddleware`](../apps/api/internal/mcp/route.go). |
| Bearer present but `sub` doesn't resolve to a `users` row → 401 (not 403) | `TokenBridge.Resolve` rejects with explicit "no KL user matches sub" error. The IDP authenticated *someone* but they have no Knowledge Layer identity. |
| Per-tool action / resource_type passed to AccessEvaluator | `kl_search`, `kl_ask_global`, `kl_get_entity` all run as `(action="view", resource_type="entity")`. Future tools must declare both. |

**Tool surface.** v0.5.1 ships read-only tools. Mutation tools (publish, patch) come later as the access-guard pattern proves itself in production.

See [`docs/operations/mcp.md`](operations/mcp.md) for the full operator guide.
