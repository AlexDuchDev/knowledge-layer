# Deployment checklist

Operational steps to run the Knowledge Layer API, workers, and optional web UI. Align env with [`.env.example`](../.env.example). **Staging/production** hardening: [PRODUCTION_HARDENING.md](PRODUCTION_HARDENING.md).

## 1. Required services

| Service | Why |
|---------|-----|
| **PostgreSQL 16 + pgvector** | Primary store; migrations run `CREATE EXTENSION vector` ([`000016_architecture_foundation.up.sql`](../apps/api/internal/db/migrations/000016_architecture_foundation.up.sql)). Use `pgvector/pgvector:pg16` or equivalent. |
| **Redis** | Asynq queue for knowledge jobs and connector tasks; invalid `REDIS_URL` fails [`app.NewDeps`](../apps/api/internal/app/deps.go) at startup. |
| **OpenSearch** (recommended) | Search index / hybrid retrieval; API waits for healthy OpenSearch in [`docker-compose.yml`](../docker-compose.yml). Empty `OPENSEARCH_URL` disables the client (keyword-only paths may still work). |
| **Web** (optional) | Next.js app from [`Dockerfile.web`](../Dockerfile.web); set `NEXT_PUBLIC_API_URL` to the API base URL. |

## 2. Environment variables and secrets

Consumed by [`config.Load`](../apps/api/internal/config/config.go), [`ValidateAPI` / `ValidateWorker`](../apps/api/internal/config/hardening.go), and CORS in [`cmd/api/main.go`](../apps/api/cmd/api/main.go).

### Local / development (`APP_ENV` not `staging` or `production`)

| Variable | Typical | Notes |
|----------|---------|--------|
| `APP_ENV` | `local` | Enables `development_header` and detailed `/ops/health`. |
| `AUTH_MODE` | `development_header` | `X-Principal-User-ID` allowed only in this profile. |
| `DATABASE_URL` | Your Postgres DSN | Default in code only for convenience; set explicitly in shared envs. |
| `REDIS_URL` | Redis URL | Workers need a reachable Redis. |
| `OPENSEARCH_URL` | `http://...` OK | Local compose uses HTTP; no TLS check. |
| `CORS_ALLOW_ORIGINS` | e.g. `http://localhost:3000` | Optional in local; code defaults to localhost if unset. |
| `OPS_AUTH_TOKEN` | Empty | Optional; if set, same bearer rules as production for `/ops/health` detail. |
| `SESSION_SECRET` | Optional for local | Required when using `AUTH_MODE=session` locally for login flows. |

### Staging / production (enforced at startup)

| Variable | Staging | Production | Notes |
|----------|---------|------------|--------|
| `APP_ENV` | `staging` | `production` | Drives [`ValidateAPI` / `ValidateWorker`](../apps/api/internal/config/hardening.go). |
| `AUTH_MODE` | Yes | Yes | Must be **`session`** only. |
| `SESSION_SECRET` | Yes | Yes | Min effective 16 bytes (see `SessionSecretBytes`). |
| `DATABASE_URL` | Yes | Yes | Must appear in **process environment** (not only implicit default). |
| `REDIS_URL` | **Yes** | **Yes** | Must be non-empty in **process environment** for API and workers (fail-closed). |
| `CORS_ALLOW_ORIGINS` | Yes (API) | Yes (API) | Must be set (comma-separated origins). |
| `APP_PUBLIC_URL` | Set | **`https://…` required** | Production startup **fails** if not prefixed with `https://` (case-insensitive). |
| `OPENSEARCH_URL` | If used | If used | **`https://`** unless `OPENSEARCH_ALLOW_INSECURE_HTTP=1` (private networks only). |
| `OPS_AUTH_TOKEN` | Optional | **Required** (min **16** chars) | Production: required so `/ops/health` and `/metrics` are not anonymously reachable with full/stub detail. Staging: optional; see below. |
| `SESSION_COOKIE_SECURE` | Optional | **Must stay secure** | Defaults **true** when unset in non-local. Production **fails** if set to `false`/`0`/`no`. |
| `AI_PRIVACY_VAULT_KEY` | Recommended (≥32 bytes) | **Required** (32 raw bytes or base64-of-32) | AES-256 key for placeholder vault; production also rejects `AI_PRIVACY_DEV_PLAINTEXT_STORE=1`. |

> **Single source of truth for env variables:** [.env.example](../.env.example) and [CONFIG_ENV.md](CONFIG_ENV.md). The table above lists only fields with profile-specific enforcement; for the full list of supported variables read CONFIG_ENV.md.

**`/ops/health` and `/metrics` (non-local):** See [PRODUCTION_GO_LIVE_CHECKLIST.md](PRODUCTION_GO_LIVE_CHECKLIST.md) §9. Summary: **`/metrics`** returns **401** unless `OPS_AUTH_TOKEN` is set **and** the request sends `Authorization: Bearer …`. Staging without a token therefore has **no** anonymous metrics scrape (by design). **`/ops/health`** without bearer is **redacted** in staging only when `OPS_AUTH_TOKEN` is **unset**; in **production** the token is always set, so unauthenticated callers get **401**.

Frontend: `NEXT_PUBLIC_API_URL`, `NEXT_PUBLIC_USE_DEV_HEADER=false` in production, session-based login.

Never commit real secrets; use a secret manager in production.

## 3. Migrations

- **API** runs [`db.MigrateUp`](../apps/api/internal/db/db.go) before opening the pool ([`cmd/api/main.go`](../apps/api/cmd/api/main.go)).
- **jobworker** and **connectorworker** also run `MigrateUp` on startup so workers can start before or without a prior API boot on a cold database.
- Idempotent second run: `ErrNoChange` is ignored.
- **Legacy databases** that previously hit duplicate `000011` may need a one-time `schema_migrations` fix; see [RELEASE_READINESS_AUDIT.md](RELEASE_READINESS_AUDIT.md) section 4.

## 3b. Post-deploy smoke (summary)

- **Local / compose:** [`scripts/smoke-local.sh`](../scripts/smoke-local.sh) after API is listening (includes `/metrics` check for `http_requests_total`). CI runs a subset after `go test` against a migrated Postgres ([`.github/workflows/ci.yml`](../.github/workflows/ci.yml)).
- **Staging / production:** session-authenticated calls to `/health`, scoped `/domains`, and bearer-authenticated `/metrics` per [STAGING_SMOKE_TEST.md](STAGING_SMOKE_TEST.md) and [PRODUCTION_GO_LIVE_CHECKLIST.md](PRODUCTION_GO_LIVE_CHECKLIST.md) §9.

## 4. Startup order

1. Start **Postgres** (healthy), **Redis** (healthy), **OpenSearch** (healthy if used).
2. Start **API** (applies migrations, ensures index, listens).
3. Start **jobworker** and **connectorworker** (same image, different entrypoints in compose).
4. Start **web** when needed (depends on API in compose).

## 5. API startup check

- Logs: `api listening on :8080` (or chosen port); no `log.Fatalf` from `config:`, `migrate:`, or `deps:`.
- HTTP: `GET /health` → `{"status":"ok"}` (no DB).
- HTTP: `GET /ops/health` — **local**: detailed JSON. **Staging** with `OPS_AUTH_TOKEN` unset: redacted booleans without auth. **Staging** with token set or **production**: bearer required for full detail; wrong/missing bearer → **401** (production always has token configured). See [PRODUCTION_GO_LIVE_CHECKLIST.md](PRODUCTION_GO_LIVE_CHECKLIST.md) §9.
- HTTP: `GET /metrics` — **local**: Prometheus exposition without auth. **Non-local**: **401** unless valid bearer and non-empty `OPS_AUTH_TOKEN`.

## 6. Worker startup check

- Processes stay running (no immediate exit).
- Logs: no `Fatalf` from `config:`, migrate, DB ping, Redis URL parse, or `NewDeps`.
- Redis must match `REDIS_URL` used by API for enqueue/consume.

## 7. Docker Compose commands

From repository root:

```bash
# Validate compose file
docker compose config -q

# Build images (CI runs this in Docker workflow)
docker compose build api web jobworker connectorworker

# Start full stack
docker compose up -d

# Follow API logs
docker compose logs -f api

# Stop
docker compose down
```

Local OpenSearch in compose has **security disabled**; do not use that configuration for production.

## 8. Post-start verification

Run [STAGING_SMOKE_TEST.md](STAGING_SMOKE_TEST.md) and [PRODUCTION_HARDENING.md](PRODUCTION_HARDENING.md) go-live checks.

- [ ] `/health` succeeds.
- [ ] `/ops/health` matches expected shape for your `APP_ENV` (redacted vs bearer).
- [ ] Authenticated API path works (dev header locally; session in production).
- [ ] Workers running without crash loop.

## Related docs

- [STAGING_SMOKE_TEST.md](STAGING_SMOKE_TEST.md) — scripted smoke and pass/fail.
- [PRODUCTION_HARDENING.md](PRODUCTION_HARDENING.md) — staging/production rules.
- [PRODUCTION_GO_LIVE_CHECKLIST.md](PRODUCTION_GO_LIVE_CHECKLIST.md) — operator go-live and go/no-go.
- [RELEASE_READINESS_AUDIT.md](RELEASE_READINESS_AUDIT.md) — governance and remaining product risks.
- [ACCESS_MODEL.md](ACCESS_MODEL.md) — HTTP access behavior.
