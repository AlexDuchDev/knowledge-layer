# Staging / local smoke test

Minimal runtime proof after deploy or `docker compose up`.

**Default compose / local:** assumes **development header** auth (`APP_ENV=local`, `AUTH_MODE=development_header`) as in [docker-compose.yml](../docker-compose.yml).

**Real staging or production:** use **`AUTH_MODE=session`**, HTTPS, and **no** dev header; smoke steps that call `/domains` must use a **real session** (e.g. browser login or cookie from `POST` auth). Re-run the same logical checks (health, scoped domains, search) with session cookies instead of `X-Principal-User-ID`. For `/metrics` in staging/production, set `OPS_AUTH_TOKEN` and use `Authorization: Bearer …` (see §5). Use [PRODUCTION_GO_LIVE_CHECKLIST.md](PRODUCTION_GO_LIVE_CHECKLIST.md) for go-live verification.

### Session smoke (automated)

Use [`scripts/smoke-session.sh`](../scripts/smoke-session.sh) against an API with **`AUTH_MODE=session`** and a user that has **`password_hash` set** (e.g. created via [`POST /instance/bootstrap`](../apps/api/internal/httpserver/auth_routes.go) or staging auto-bootstrap per [CONFIG_ENV.md](./CONFIG_ENV.md)):

```bash
export API_BASE="https://staging-api.example.com"   # or http://localhost:8080
export SMOKE_EMAIL="admin@yourcompany.com"
export SMOKE_PASSWORD="…"
# Staging/production metrics detail: export OPS_AUTH_TOKEN and pass bearer (script uses it when set)
# Self-signed TLS only: CURL_INSECURE=1
./scripts/smoke-session.sh
```

The script logs in with `POST /auth/login`, stores the signed cookie `kl_session`, then calls `/domains`, `/search?q=test`, `/knowledge-jobs/engine-metadata`, and `/auth/me` with that cookie—matching [PRODUCTION_GO_LIVE_CHECKLIST.md](PRODUCTION_GO_LIVE_CHECKLIST.md) §10 “authenticated read via session”.

**Optional full compose in CI:** [`.github/workflows/smoke-compose.yml`](../.github/workflows/smoke-compose.yml) (`workflow_dispatch`) builds images, runs `docker compose up`, then [`scripts/smoke-local.sh`](../scripts/smoke-local.sh).

**Canonical web redirects (Next.js):** with the `web` service up, [`scripts/smoke-canonical-routes.sh`](../scripts/smoke-canonical-routes.sh) (`WEB_BASE` default `http://localhost:13000`) curls legacy `/admin/*`, `/access`, and `/app/*` paths.

## Preconditions

- API reachable at `API_BASE`.
  - **`scripts/smoke-local.sh` default:** `http://localhost:8080` (matches `go run ./cmd/api` unless you changed `API_PORT`).
  - **This repo’s `docker compose` published API port:** defaults to **`18080`** on the host (container still listens on `:8080`). Set `API_BASE=http://localhost:18080` (or override `API_HOST_PORT` in [`docker-compose.yml`](../docker-compose.yml)).
- Migrations applied (automatic on API start).
- Dev seed present: admin user `30000000-0000-0000-0000-000000000001` from [`000008_dev_seed.up.sql`](../apps/api/internal/db/migrations/000008_dev_seed.up.sql).

## Exact steps

1. **Start stack** (from repo root):

   ```bash
   docker compose up -d
   ```

   Wait until Postgres, Redis, and OpenSearch are healthy (compose waits on `depends_on` conditions).

2. **Confirm API boot**

   ```bash
   docker compose logs api 2>&1 | tail -30
   ```

   **Expected:** No `migrate:` / `deps:` / `listen:` fatal lines after startup; line containing `api listening on :8080`.

3. **Liveness**

   ```bash
   curl -sf "$API_BASE/health"
   ```

   **Expected:** HTTP 200, JSON body includes `"status":"ok"`.

4. **Dependency health**

   ```bash
   curl -sf "$API_BASE/ops/health"
   ```

   **Expected (local / `APP_ENV=local`):** HTTP 200, `"database":"ok"` (exact string in JSON value). OpenSearch may show an error string until the cluster is ready; retry once if needed.

   **Expected (`APP_ENV=staging`, `OPS_AUTH_TOKEN` unset):** HTTP 200, **redacted** JSON: `status`, `database_ok`, `opensearch_ok` (booleans). No raw driver errors.

   **Expected (`APP_ENV=staging`, `OPS_AUTH_TOKEN` set):** Unauthenticated → **401**. Detailed JSON only with:

   ```bash
   curl -sf -H "Authorization: Bearer $OPS_AUTH_TOKEN" "$API_BASE/ops/health"
   ```

   **Expected (`APP_ENV=production`):** `OPS_AUTH_TOKEN` is **required** at startup, so unauthenticated `/ops/health` → **401**; use the same bearer `curl` as above for detail.

   Wrong or missing bearer when a token is configured → **401**.

5. **Metrics (non-local)**

   ```bash
   curl -s -o /dev/null -w "%{http_code}" "$API_BASE/metrics"
   ```

   **Expected:** **401** unless `OPS_AUTH_TOKEN` is set **and** you pass a valid bearer. Local (`APP_ENV=local`) returns **200** with Prometheus text (Go/process/`http_requests_total`) without auth.

   With a valid bearer (or on local), confirm the body mentions **`http_requests_total`** (HTTP middleware counter). [`scripts/smoke-local.sh`](../scripts/smoke-local.sh) checks this when run against a warm API.

6. **Authenticated read (dev header)** — **local compose only** (skipped when `AUTH_MODE=session`)

   ```bash
   curl -sf -H "X-Principal-User-ID: 30000000-0000-0000-0000-000000000001" \
     "$API_BASE/domains"
   ```

   **Expected:** HTTP 200, JSON array with at least one object; default domain name `"Default"` from seed.

7. **Workers (optional minimal pass)**

   ```bash
   docker compose logs jobworker 2>&1 | tail -15
   docker compose logs connectorworker 2>&1 | tail -15
   ```

   **Expected:** No `Fatalf` / `migrate:` / `redis url:` / `deps:` error lines after startup; processes keep running.

## Expected results summary

| Step | Pass signal |
|------|-------------|
| API logs | Listening message, no fatal |
| `/health` | 200, `status` ok |
| `/ops/health` | 200; local: `database` string ok; staging (no token): redacted; staging+token / prod: 401 without bearer |
| `/metrics` | Local: 200 stub; non-local: 401 without bearer + `OPS_AUTH_TOKEN` |
| `/domains` + header | 200, non-empty array |
| Workers | Running, no startup fatals |

## Failure signals

| Symptom | Likely cause |
|---------|----------------|
| `migrate: ...` in API logs | Postgres without pgvector, wrong DSN, or migration conflict |
| `deps: ... job queue publisher` | Invalid `REDIS_URL` |
| `listen: address already in use` | API port conflict (host `API_HOST_PORT` mapping, or `API_PORT` when running `go run`) |
| `/health` fails | API not running or wrong `API_BASE` |
| `/ops/health` degraded / false booleans | `DATABASE_URL`, network, Postgres, or OpenSearch down (staging/prod: use bearer for details) |
| `/ops/health` 401 | `OPS_AUTH_TOKEN` set but bearer missing or invalid |
| `/domains` 401 | Missing `X-Principal-User-ID` or auth mode does not allow header |
| `/domains` 403 / empty | Principal not in seed DB or grants missing |
| Worker exits immediately | Same DB/Redis issues as API |

## Where to inspect logs

- API: `docker compose logs -f api`
- Job worker: `docker compose logs -f jobworker`
- Connector worker: `docker compose logs -f connectorworker`
- Postgres: `docker compose logs postgres`

## Pass / fail criteria

- **Pass:** Steps 3–6 all succeed on first or second retry where applicable (OpenSearch only for step 4; step 6 only for local dev header).
- **Partial pass:** API steps pass, workers not run (document as partial).
- **Fail:** Any required step fails after retry, or API/worker crash loop.

## Automated helper

From repo root (requires running stack and `curl`):

```bash
# docker compose (default published API host port in this repo)
API_BASE=http://localhost:18080 ./scripts/smoke-local.sh

# go run ./cmd/api (default)
./scripts/smoke-local.sh
```
