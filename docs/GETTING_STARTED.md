# Getting started

Fast path from zero to a running local instance. For production, continue to [DEPLOYMENT.md](DEPLOYMENT.md) and [PRODUCTION_GO_LIVE_CHECKLIST.md](PRODUCTION_GO_LIVE_CHECKLIST.md).

After the stack is up, contributors can follow [EXTERNAL_DEV_QUICKSTART.md](EXTERNAL_DEV_QUICKSTART.md) (canonical web URLs) and read [LIMITATIONS.md](LIMITATIONS.md) for stubs and degraded modes.

## Prerequisites

- Docker (Postgres + Redis + OpenSearch for full Search/hybrid behavior)
- Go 1.22+
- Node 20+

## 1. Clone and configure

```bash
cp .env.example .env
# Edit .env if needed (see CONFIG_ENV.md)
```

### Host `go run` + compose (default published ports)

When the API runs on your machine and databases run via this repo’s `docker-compose.yml`, align URLs with the default host ports (override with `POSTGRES_HOST_PORT`, `REDIS_HOST_PORT`, `OPENSEARCH_HOST_PORT` if you changed them):

| Variable | Typical value |
|----------|----------------|
| `DATABASE_URL` | `postgres://knowledge:knowledge@localhost:25432/knowledge?sslmode=disable` |
| `REDIS_URL` | `redis://localhost:16379` |
| `OPENSEARCH_URL` | `http://localhost:19200` |

`NEXT_PUBLIC_API_URL` should match where the API listens (default `http://localhost:8080` when using `API_PORT=8080`). If you use compose `web` on port **13000**, set `CORS_ALLOW_ORIGINS` (and optionally web dev URL) to that origin and point `NEXT_PUBLIC_API_URL` at the published API port (**18080** by default).

## 2. Start databases

```bash
make db-up
# or: docker compose up -d postgres redis opensearch
```

## 3. Run the API

From `apps/api`:

```bash
go run ./cmd/api
```

Migrations apply on startup. API default when running locally via Go: `http://localhost:8080` — `GET /health` for liveness.

If you run the API via this repo’s `docker compose`, use the **published host port** from `docker-compose.yml` (defaults to `http://localhost:18080`; see [STAGING_SMOKE_TEST.md](./STAGING_SMOKE_TEST.md)).

## 4. Run the web app

From `apps/web`:

```bash
npm install && npm run dev
```

Open `http://localhost:3000`. Local pilot often uses `NEXT_PUBLIC_USE_DEV_HEADER=true` and `X-Principal-User-ID` — see root [README.md](../README.md).

## 5. Optional workers

With `REDIS_URL` set:

```bash
cd apps/api && go run ./cmd/jobworker
cd apps/api && go run ./cmd/connectorworker
```

## 6. Smoke check

- Quick API checks (health, domains, search, optional Ask when `OPENAI_API_KEY` or `OPENROUTER_API_KEY` is set):  
  `API_BASE=http://localhost:8080 ./scripts/smoke-local-real-openai.sh`  
  (With an empty knowledge base, Ask may return 400 “no search evidence”; the script treats that as OK for infra smoke — add content for a full Ask path.)
- Broader checks: [STAGING_SMOKE_TEST.md](STAGING_SMOKE_TEST.md) (adapt for your `APP_ENV`).

## Next steps

| Audience | Read next |
|----------|-----------|
| Concepts | [PRODUCT_CONCEPTS.md](PRODUCT_CONCEPTS.md), [GLOSSARY.md](GLOSSARY.md) |
| Architecture | [ARCHITECTURE.md](ARCHITECTURE.md) |
| Control plane | [CONTROL_PLANE_OVERVIEW.md](CONTROL_PLANE_OVERVIEW.md) |
| End users | [USER_GUIDE.md](USER_GUIDE.md) |
| Operators | [DEPLOYMENT.md](DEPLOYMENT.md) |
| Contributors | [CONTRIBUTING.md](CONTRIBUTING.md), [BACKEND_ARCHITECTURE.md](BACKEND_ARCHITECTURE.md) |
