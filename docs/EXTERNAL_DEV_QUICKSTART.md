# External developer quickstart (canonical URLs)

For environment setup and stack overview, use [GETTING_STARTED.md](./GETTING_STARTED.md) and the root [README.md](../README.md). This page gives a **minimal golden path** after the API and web app are running locally, using **canonical web routes** (deprecated `/app/*` mirrors **redirect** here; there is no separate mounted “Knowledge app” shell).

## Assumptions

- Postgres + Redis up (`make db-up` or compose).
- API running: `cd apps/api && go run ./cmd/api` (migrations apply on start).
- Web running: `cd apps/web && npm run dev`.
- Local pilot auth: `NEXT_PUBLIC_USE_DEV_HEADER=true` and `X-Principal-User-ID` set to a seeded user UUID (see migration `000008_dev_seed` / README).

## 1. Open the product (canonical)

| Step | URL | What to verify |
|------|-----|----------------|
| Bootstrap (if needed) | `http://localhost:3000/bootstrap` | Instance gains a domain; skip if auto-bootstrap already ran. |
| Home | `http://localhost:3000/` | Dash shell; golden-path copy and “what works locally” callouts. |
| Search | `http://localhost:3000/search` | Scoped search UI (`GET /search`); OpenSearch must be up in compose. |
| Entity | From a search hit → `/entities/{id}` | Detail + trust signals. |
| Ask | `http://localhost:3000/ask` | Ask UI; `POST /ask` when LLM/embeddings are configured ([SELF_HOSTED.md](./SELF_HOSTED.md)). |
| Governance | `http://localhost:3000/governance` | Product governance hub (queues as implemented). |
| Entities | `http://localhost:3000/entities` | Entity list when present. |

**Admin / control plane:** `http://localhost:3000/control-plane` redirects to `.../control-plane/governance`. Canonical **operator** URLs use `/control-plane/*`; list and builder screens often **rewrite** to dash routes. See [ADMIN_UI_CONSOLIDATION_PLAN.md](./ADMIN_UI_CONSOLIDATION_PLAN.md) and [INFORMATION_ARCHITECTURE_V1.md](./INFORMATION_ARCHITECTURE_V1.md).

**Setup flow:** after bootstrap, operators can use `/control-plane/setup`, `/control-plane/setup/templates`, and `/control-plane/setup/session/new` to create a real onboarding session, preview the seeded preset mix, and launch instantiated roles/scenarios/jobs for the currently supported templates.

## 2. API smoke (optional)

```bash
# If you run the API via `go run ./cmd/api`, the default is usually :8080 on localhost.
curl -sS "http://localhost:8080/health"
curl -sS -H "X-Principal-User-ID: <your-seed-uuid>" "http://localhost:8080/domains"

# If you run the API via this repo's `docker compose`, the default published host port is :18080
# (see `API_HOST_PORT` in `docker-compose.yml`).
curl -sS "http://localhost:18080/health"
curl -sS -H "X-Principal-User-ID: <your-seed-uuid>" "http://localhost:18080/domains"
```

Use [STAGING_SMOKE_TEST.md](./STAGING_SMOKE_TEST.md) and [scripts/smoke-local.sh](../scripts/smoke-local.sh) for a fuller checklist.

## 3. Read next

- [LIMITATIONS.md](./LIMITATIONS.md) — stubs and degraded modes.
- [GLOSSARY.md](./GLOSSARY.md) — connector vs source feed vs entity.
- [MODULE_BOUNDARIES.md](./MODULE_BOUNDARIES.md) — where to add backend code.
- [docs/README.md](./README.md) — full documentation index.
