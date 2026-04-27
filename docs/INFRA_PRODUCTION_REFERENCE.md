# Production infrastructure reference

Operator checklist derived from [PRODUCTION_GO_LIVE_CHECKLIST.md](./PRODUCTION_GO_LIVE_CHECKLIST.md) §1–4 and [SELF_HOSTED.md](./SELF_HOSTED.md). Use this page as a **single inventory** when provisioning; keep values in your secret manager, not in git.

## TLS and ingress

- Terminate **HTTPS** at your load balancer or ingress; upstream to API/web can be HTTP on a private network if policy allows.
- Certificates must match public hostnames used by browsers (`APP_PUBLIC_URL`, web origin in `CORS_ALLOW_ORIGINS`).
- **Production:** `APP_PUBLIC_URL` must be `https://…` (enforced by [`ValidateAPI`](../apps/api/internal/config/hardening.go)).

## Secret store

Store at minimum: `DATABASE_URL`, `REDIS_URL`, `SESSION_SECRET`, `OPS_AUTH_TOKEN` (production), LLM keys (`OPENAI_API_KEY` or `OPENROUTER_API_KEY`), connector OAuth secrets, SMTP credentials if used.

## PostgreSQL

- **Postgres 16** with **pgvector** (same baseline as [docker-compose.yml](../docker-compose.yml) and CI `pgvector/pgvector:pg16`).
- Backups and restore drills: [OPERATIONS.md](./OPERATIONS.md).

## Redis

- Single logical Redis for API + **jobworker** + **connectorworker** (`REDIS_URL`); required in `staging` / `production` for [`ValidateAPI`](../apps/api/internal/config/hardening.go) / `ValidateWorker`.

## OpenSearch

- **Production:** cluster with **TLS** and **authentication**; set `OPENSEARCH_URL` to `https://…` (or document `OPENSEARCH_ALLOW_INSECURE_HTTP=1` only on a private network).
- Do **not** reuse local compose images with `plugins.security.disabled=true` outside dev.

## Optional S3-compatible blobstore

- When raw payload retention across restarts is required: `BLOBSTORE_BACKEND=s3` and related env vars ([CONFIG_ENV.md](./CONFIG_ENV.md), [`blobstore/wire.go`](../apps/api/internal/blobstore/wire.go)).
- Verify bucket policy (least privilege), encryption at rest, and network path from API/workers before promoting to production.

## Related

- [PRODUCTION_HARDENING.md](./PRODUCTION_HARDENING.md)
- [RUNBOOK_STAGING_PROD.md](./RUNBOOK_STAGING_PROD.md)
- [PRODUCTION_CUTOVER_QUICKREF.md](./PRODUCTION_CUTOVER_QUICKREF.md)
