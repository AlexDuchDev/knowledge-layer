# Production cutover quick reference

One-page map to **deploy**, **verify**, and **rollback** without rereading the full checklist. Canonical detail: [PRODUCTION_GO_LIVE_CHECKLIST.md](./PRODUCTION_GO_LIVE_CHECKLIST.md).

## Before cutover

1. Infra ready: [INFRA_PRODUCTION_REFERENCE.md](./INFRA_PRODUCTION_REFERENCE.md).
2. Env validated for `APP_ENV=production`: §2 table in [PRODUCTION_GO_LIVE_CHECKLIST.md](./PRODUCTION_GO_LIVE_CHECKLIST.md) (`AUTH_MODE=session`, `REDIS_URL`, `CORS_ALLOW_ORIGINS`, `OPS_AUTH_TOKEN`, `APP_PUBLIC_URL` https, secure cookies, OpenSearch URL rules).
3. Images built from [Dockerfile.api](../Dockerfile.api) and [Dockerfile.web](../Dockerfile.web); same revision tag for API and both workers.

## Deploy order (§5)

1. Apply secrets and config to API, **jobworker**, **connectorworker**, **web**.
2. Roll **API** (runs migrations on start).
3. Roll **jobworker** and **connectorworker** (migrations idempotent).
4. Roll **web** with production `NEXT_PUBLIC_*` (`NEXT_PUBLIC_USE_DEV_HEADER=false`).

## Verify (§§7–10)

| Check | Command / action |
|-------|------------------|
| Liveness | `curl -sf "https://<api>/health"` → 200, `"status":"ok"` |
| Ops | Bearer rules per [PRODUCTION_GO_LIVE_CHECKLIST.md](./PRODUCTION_GO_LIVE_CHECKLIST.md) §9 |
| Metrics | Same §9; scrape with `Authorization: Bearer $OPS_AUTH_TOKEN` |
| Session smoke | [`scripts/smoke-session.sh`](../scripts/smoke-session.sh) with real user credentials |
| Migrations | §6 — no `dirty=true` in `schema_migrations` |

## Rollback triggers (§11)

Stop promotion or roll back if: startup `config:` / `ValidateAPI` fatals, migrations dirty, `/health` not 200, session login broken, sustained dependency outage confirmed, or 5xx / auth regression vs baseline.

## After cutover

- [OPERATIONS.md](./OPERATIONS.md), secret rotation for `SESSION_SECRET` and `OPS_AUTH_TOKEN` (§12).
