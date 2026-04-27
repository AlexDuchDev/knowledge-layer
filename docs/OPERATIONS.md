# Operations

Quick reference for running a self-hosted Knowledge Layer instance.

## Health

### API (`apps/api/cmd/api`)

| Endpoint | Notes |
|----------|--------|
| `GET /health` | Liveness: `{ "status": "ok" }` |
| `GET /ops/health` | DB ping + OpenSearch status. Bearer-gated outside `APP_ENV=local` (see [PRODUCTION_HARDENING.md](PRODUCTION_HARDENING.md) §3). |
| `GET /metrics` | Prometheus Go/process collectors plus: `http_requests_total{method,route}`, `knowledge_job_run_duration_seconds{job_type,status}`, `connector_sync_duration_seconds{adapter_kind,status}`, on-scrape `postgres_pool_*` gauges, and `asynq_queue_*{queue}` gauges (when Redis is configured). Bearer-gated outside local. |

### Workers (`cmd/jobworker`, `cmd/connectorworker`)

Each worker boots a dedicated HTTP server alongside its Asynq processor. Default ports `:9001` (jobworker) and `:9002` (connectorworker); override with `JOBWORKER_HEALTH_PORT` / `CONNECTORWORKER_HEALTH_PORT`.

| Endpoint | Notes |
|----------|--------|
| `GET /health` | Always-public liveness ping. |
| `GET /ops/health` | Returns JSON snapshot: worker name, uptime, DB ping, Redis ping, queue depths (per Asynq queue), and **last-processed timestamp per task type** (the most reliable stuck-worker signal — queue depth alone is misleading when there is no inbound traffic). Bearer-gated outside local using the same `OPS_AUTH_TOKEN` as the API. |

A "hung worker" is detected by the `last_processed_by_task` field falling far behind your expected SLO for that task. Combine with `queues[*].pending` to distinguish "stuck" from "idle".

For **`GET /ops/health`** in production, call from inside the network with `Authorization: Bearer $OPS_AUTH_TOKEN`. Operators are encouraged to scrape this endpoint into their observability stack.

## Failure signals

| Endpoint | Purpose |
|----------|---------|
| `GET /ops/failed-runs` | Recent failed ingestion runs and job runs (identity admin) |
| `GET /audit-events` | Audit trail (identity admin) |
| Web `/audit` | Same data via UI |

## Logs

- API: structured application logs from Fiber handlers and services (ensure log aggregation in prod).
- Workers: separate process — monitor exit codes, restarts, **and the per-worker `/ops/health` snapshot** for queue depth and last-processed staleness. Vault audit events (`vault.placeholder_stored`, `vault.placeholder_decrypted`, `vault.rehydration_applied`) appear in `audit_events` and via `GET /audit-events`; their absence after Ask traffic indicates a vault wiring issue.

## Incident checklist

1. Confirm Postgres reachable and disk not full.
2. Check `GET /health` and recent errors in API logs.
3. Inspect `/ops/failed-runs` and ingestion connector status.
4. Verify `AUTH_MODE` and `SESSION_SECRET` if users cannot sign in.
5. Verify SMTP if invitations or magic links are not delivered.

## Security

- Never expose Postgres or Redis to the public internet.
- Rotate `SESSION_SECRET` invalidates all sessions — plan a maintenance window.
- See [SECURITY.md](../SECURITY.md) for vulnerability reporting.
