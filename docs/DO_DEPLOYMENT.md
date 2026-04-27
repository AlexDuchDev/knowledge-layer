# DigitalOcean deployment

Practical notes for deploying this **self-hosted** stack on DigitalOcean. Adapt to your org’s security and compliance requirements.

## Options

1. **App Platform** — managed containers; attach **managed Postgres** (with **pgvector**), **managed Redis**, and optionally a **Droplet** or managed OpenSearch alternative for search.
2. **Droplet + Docker Compose** — mirror local compose with production env; put **TLS** in front (DO Load Balancer or Caddy/Traefik).

## Repository artifacts

- Example App Platform spec (adjust names/regions): [deploy/do/app-spec.example.yaml](../deploy/do/app-spec.example.yaml)
- Folder README: [deploy/do/README.md](../deploy/do/README.md)

## Environment

Set the same variables as [DEPLOY_CHECKLIST.md](DEPLOY_CHECKLIST.md) and [PRODUCTION_GO_LIVE_CHECKLIST.md](PRODUCTION_GO_LIVE_CHECKLIST.md): `DATABASE_URL`, `REDIS_URL`, `AUTH_MODE=session`, `SESSION_SECRET`, `CORS_ALLOW_ORIGINS`, `APP_PUBLIC_URL` (https in production), `OPS_AUTH_TOKEN` (production), OpenSearch URL (https) or explicit insecure opt-in.

Store secrets in **DO Secrets** or a vault—never in the image.

## doctl examples

```bash
# Authenticate (one-time)
doctl auth init

# Create app from spec (after editing app-spec.example.yaml)
doctl apps create --spec deploy/do/app-spec.example.yaml

# List apps
doctl apps list

# Logs for a component (use IDs from describe)
doctl apps logs <app-id> --type run --component api
```

## See also

- [DO_INFRA_TOPOLOGY.md](DO_INFRA_TOPOLOGY.md)
- [DEPLOYMENT.md](DEPLOYMENT.md)
