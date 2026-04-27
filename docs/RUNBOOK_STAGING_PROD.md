# Runbook: staging and production (golden path)

Single entry point for operators moving from local compose to **staging** or **production**. This file does not duplicate long procedures; it orders the documents and scripts you must follow.

## 1. Scope and OSS contract

- [OSS_V1_SCOPE.md](./OSS_V1_SCOPE.md) — what v1 OSS is expected to support.
- [LIMITATIONS.md](./LIMITATIONS.md) — stubs, degraded modes, runnable job types, connector normalization depth.

## 2. Bring-up and topology

- **Self-hosted / components:** [SELF_HOSTED.md](./SELF_HOSTED.md) (capability matrix, auth modes, env vars).
- **Repository layout & ports:** [REPO_STRUCTURE.md](./REPO_STRUCTURE.md), root [docker-compose.yml](../docker-compose.yml) (default **non-standard host ports** to avoid collisions — adjust `*_HOST_PORT` env vars).
- **DigitalOcean (if applicable):** [DO_DEPLOYMENT.md](./DO_DEPLOYMENT.md), [DO_INFRA_TOPOLOGY.md](./DO_INFRA_TOPOLOGY.md).

## 3. Configuration and secrets

- [CONFIG_ENV.md](./CONFIG_ENV.md) — full variable reference.
- Production: **`AUTH_MODE=session`**, strong `SESSION_SECRET`, no `development_header`. See [SELF_HOSTED.md](./SELF_HOSTED.md) § Auth modes.

## 4. Verification (smoke)

- **API / compose smoke:** [STAGING_SMOKE_TEST.md](./STAGING_SMOKE_TEST.md) and [scripts/smoke-local.sh](../scripts/smoke-local.sh) (set `API_BASE` to match published API port, e.g. `http://localhost:18080`).
- **Go-live checklist:** [PRODUCTION_GO_LIVE_CHECKLIST.md](./PRODUCTION_GO_LIVE_CHECKLIST.md).
- **Deploy checklist:** [DEPLOY_CHECKLIST.md](./DEPLOY_CHECKLIST.md).

## 5. Web canonical URLs (smoke in browser)

After API smoke passes, spot-check **canonical** routes from [INFORMATION_ARCHITECTURE_V1.md](./INFORMATION_ARCHITECTURE_V1.md):

- User shell: `/`, `/search`, `/ask`, `/governance` (session auth).
- Control plane: `/control-plane/governance`, `/control-plane/sources`, `/control-plane/jobs` (no 404 on primary nav).
- Legacy `/admin/*` and `/access` should **308** to `/control-plane/*` per [ADMIN_UI_CONSOLIDATION_PLAN.md](./ADMIN_UI_CONSOLIDATION_PLAN.md).

## 6. OpenSearch (mandatory read for prod)

**Repo `docker-compose` enables `plugins.security.disabled=true` — dev / lab only.**  
Production must use TLS + authentication and a locked-down network path. See [OPENSEARCH_PROD_VS_DEV.md](./OPENSEARCH_PROD_VS_DEV.md) and [LIMITATIONS.md](./LIMITATIONS.md) (OpenSearch row).

## 7. Hardening and day-2

- [PRODUCTION_HARDENING.md](./PRODUCTION_HARDENING.md)
- [OPERATIONS.md](./OPERATIONS.md)
