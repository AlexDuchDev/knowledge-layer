# Setup examples

## Environment snippets

### Local development

See root [`.env.example`](../.env.example). Minimal local:

```bash
APP_ENV=local
DATABASE_URL=postgres://...
# If you port-forward / publish non-default host ports, match REDIS_URL / OPENSEARCH_URL to those ports.
REDIS_URL=redis://localhost:6379
OPENSEARCH_URL=http://localhost:9200
AUTH_MODE=development_header
```

### Staging-style (session, no dev header)

```bash
APP_ENV=staging
AUTH_MODE=session
SESSION_SECRET=<32+ bytes secret>
DATABASE_URL=postgres://...?sslmode=require
REDIS_URL=redis://...
CORS_ALLOW_ORIGINS=https://app.example.com
APP_PUBLIC_URL=https://app.example.com
OPENSEARCH_URL=https://opensearch.example:9200
```

### Production extras

See [PRODUCTION_GO_LIVE_CHECKLIST.md](PRODUCTION_GO_LIVE_CHECKLIST.md): `OPS_AUTH_TOKEN`, https `APP_PUBLIC_URL`, secure cookies, etc.

## Docker Compose

```bash
docker compose config -q
docker compose up -d
# Default published API host port in this repo's docker-compose.yml is 18080 (override with API_HOST_PORT).
API_BASE=http://localhost:18080 ./scripts/smoke-local.sh
```

## DigitalOcean

```bash
doctl auth init
doctl apps create --spec deploy/do/app-spec.example.yaml
```

Edit [deploy/do/app-spec.example.yaml](../deploy/do/app-spec.example.yaml) before apply.

## Bootstrap / first admin

Use `/bootstrap` when API reports `needs_bootstrap`, then control plane **Setup** sessions ([onboarding-setup-flow.md](onboarding-setup-flow.md)).

## Verification

- [STAGING_SMOKE_TEST.md](STAGING_SMOKE_TEST.md)
- [PRODUCTION_GO_LIVE_CHECKLIST.md](PRODUCTION_GO_LIVE_CHECKLIST.md) §§7–10
