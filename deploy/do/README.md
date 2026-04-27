# DigitalOcean deployment pack

Example artifacts for running Knowledge Layer on DigitalOcean.

## Files

| File | Purpose |
|------|---------|
| [app-spec.example.yaml](app-spec.example.yaml) | **App Platform** starter spec (API + web placeholders). Replace registry, regions, and secrets. |

## Before you apply

1. Provision **Postgres** with **pgvector** (managed DB or custom image).
2. Provision **Redis** for Asynq.
3. Provision **OpenSearch** (or alternative) with **TLS** in production.
4. Fill environment variables per [docs/DEPLOY_CHECKLIST.md](../../docs/DEPLOY_CHECKLIST.md) and [docs/PRODUCTION_GO_LIVE_CHECKLIST.md](../../docs/PRODUCTION_GO_LIVE_CHECKLIST.md).

## Docs

- [docs/DO_DEPLOYMENT.md](../../docs/DO_DEPLOYMENT.md)
- [docs/DO_INFRA_TOPOLOGY.md](../../docs/DO_INFRA_TOPOLOGY.md)
- [docs/DEPLOYMENT.md](../../docs/DEPLOYMENT.md)
