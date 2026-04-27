# Self-hosted deployment

This guide is for operators who run **one Knowledge Layer instance per organization** (your own Postgres, API, web UI, optional OpenSearch).

## Components

| Component | Role |
|-----------|------|
| `apps/api` | Go (Fiber) API — migrations on startup, business logic |
| `apps/web` | Next.js UI |
| `apps/api/cmd/jobworker` | Asynq worker for queued knowledge job runs (`REDIS_URL`); see `apps/workers/README.md` |
| PostgreSQL | System of record |
| Redis | Queues / cache (see `docker-compose` / `.env.example`) |
| OpenSearch | Optional full-text search (`OPENSEARCH_URL`) |

## Capability matrix (air-gapped and degraded modes)

Use this when planning **no cloud**, **no OpenSearch**, or **no LLM** deployments. “Full” means typical production with optional components configured.

| Capability | Full stack | `OPENSEARCH_URL` empty | No `OPENAI_API_KEY` / `OPENROUTER_API_KEY` (no mock) |
|------------|------------|-------------------------|-------------------------------|
| **API + Postgres + session auth** | Yes | Yes | Yes |
| **Web UI** | Yes | Yes | Yes |
| **Browse entities / entity detail** | Yes | Yes | Yes |
| **`GET /search` with free-text `q`** | Semantic-quality via OpenSearch + permission filter | **Degraded:** title `ILIKE` match on `entity_search_projection` within granted domains (see `search.Service.Search`) | Same as OpenSearch column |
| **Filters-only search** (domain, type, lifecycle, etc., `q` empty) | Yes | Yes | Yes |
| **Ask / scoped Q&A** (`POST /ask`, retrieval modes `semantic` / `hybrid`) | Needs embeddings + chat model | Same | **Fails or falls back** to `keyword_only` where implemented; synthesis still needs a **chat completion** path unless disabled at product level |
| **Embeddings / chunk semantic retrieval** | `OPENAI_API_KEY` or `OPENROUTER_API_KEY` (OpenAI-compatible embedder); tests use `OPENAI_MOCK=1` | Independent of OpenSearch | **Unavailable** without a key or mock |
| **Knowledge jobs worker** | Redis + API | Same | Same (jobs may still invoke LLM if job definition requires it) |
| **AI privacy vault / rehydration** | `AI_PRIVACY_VAULT_KEY` in production | Same | Same |

**Notes:**
- **OpenSearch** improves full-text recall for `q`; it is **not** a substitute for access control (domain grants + per-entity `view` still apply).
- **Air-gapped** operators should run an **embedding and chat-compatible endpoint** on-network (`OPENAI_BASE_URL` points to local gateway) or limit the product to **browse + filter search + governance** until models are available.
- **Embeddings vs privacy vault:** chunk embedding calls do **not** use `PrivacyGateway` / `AI_PRIVACY_VAULT_KEY` today; only generative chat paths do. See [adr/0013-embeddings-privacy-boundary.md](./adr/0013-embeddings-privacy-boundary.md) and [AI_PRIVACY_FLOW.md](./AI_PRIVACY_FLOW.md) §6.
- See [CONFIG_ENV.md](./CONFIG_ENV.md) for variables (`OPENAI_*`, `RETRIEVAL_*`, `OPENSEARCH_URL`).

## Requirements

- **PostgreSQL 16** with **pgvector** (same baseline as CI and [PRODUCTION_GO_LIVE_CHECKLIST.md](./PRODUCTION_GO_LIVE_CHECKLIST.md); migrations assume this major). Older majors are not validated in this repo.
- Go 1.22+ (to build API)
- Node 20+ (to build web)
- TLS termination in front of API and web in production

## Environment variables

Copy [.env.example](/.env.example) and extend with:

| Variable | Purpose |
|----------|---------|
| `DATABASE_URL` | Postgres connection string |
| `API_PORT` | API listen port |
| `APP_ENV` | `local`, `staging`, `production` |
| `OPENSEARCH_URL` | Empty to disable OpenSearch; set for full-text `q` search |
| `AUTH_MODE` | `development_header` (default, dev only) or `session` (production) |
| `SESSION_SECRET` | Required when `AUTH_MODE=session` — random 32+ bytes, base64 or hex |
| `ALLOW_SELF_REGISTRATION` | Default `false` — users join via invitation only |
| `APP_PUBLIC_URL` | Public origin of the API, e.g. `https://api.example.com` — used in invitation emails |
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASSWORD`, `SMTP_FROM` | Outbound mail for invitations and magic links |
| `BUILD_VERSION` | Optional; shown on `/settings` and `GET /settings/instance` |

### Auth modes

- **`development_header`**: API accepts `X-Principal-User-ID` for every request. **Do not use in production.**
- **`session`**: Sign-in via password or magic link; session cookie `kl_session`. Header can still be parsed in `local` for tests.

## Container image (API)

A minimal API image is defined in [`Dockerfile.api`](../Dockerfile.api) at the repository root:

```bash
docker build -f Dockerfile.api -t knowledge-layer-api:local .
```

## Docker Compose (development)

```bash
make db-up
```

Brings up Postgres and Redis as defined in the repo `docker-compose.yml`. Add OpenSearch locally if you use `OPENSEARCH_URL`.

The repo includes [`docker-compose.override.yml`](../docker-compose.override.yml), merged automatically with `docker-compose.yml`, which bind-mounts Postgres, Redis, and OpenSearch data under `./data/` so volumes survive container recreation. Create the directories once: `mkdir -p data/postgres data/redis data/opensearch`. The `data/` path is listed in `.gitignore`.

## Bare-metal / VM

For Linux hosts (systemd-based) the recommended layout puts the binaries under `/opt/knowledge-layer/`, runtime data under `/var/lib/knowledge-layer/`, and logs through journald. Adapt paths and the service user to your environment.

### 1. System user and directories

```bash
sudo useradd --system --home /var/lib/knowledge-layer --shell /usr/sbin/nologin knowledge
sudo install -d -o knowledge -g knowledge /opt/knowledge-layer/{bin,web,env}
sudo install -d -o knowledge -g knowledge /var/lib/knowledge-layer
sudo install -d -o knowledge -g knowledge /var/log/knowledge-layer
```

### 2. Database

```bash
sudo -u postgres psql <<SQL
  CREATE ROLE knowledge LOGIN PASSWORD '<secret>';
  CREATE DATABASE knowledge OWNER knowledge;
  -- pgvector is required for embeddings; the API does not create the extension itself.
  \c knowledge
  CREATE EXTENSION IF NOT EXISTS vector;
  CREATE EXTENSION IF NOT EXISTS pgcrypto;
SQL
```

Set `DATABASE_URL=postgres://knowledge:<secret>@localhost:5432/knowledge?sslmode=require` (or `sslmode=verify-full` with a CA bundle).

### 3. Build the binaries

On a build host with Go 1.22+ and Node 20+:

```bash
git clone https://github.com/<org>/knowledge-layer.git && cd knowledge-layer
git checkout v0.2.0   # or the tag you intend to deploy

cd apps/api
go build -o /tmp/knowledge-api          ./cmd/api
go build -o /tmp/knowledge-jobworker    ./cmd/jobworker
go build -o /tmp/knowledge-connectorworker ./cmd/connectorworker

cd ../web
npm ci
npm run build         # produces .next/
```

Ship `/tmp/knowledge-*` to `/opt/knowledge-layer/bin/` on the production host (use your CD tool — `scp`, `rsync`, container image, etc.) and the `apps/web` directory (with `.next/`, `package.json`, `node_modules/`) to `/opt/knowledge-layer/web/`. Strip dev dependencies from the web bundle if size matters: `npm prune --production` after `next build`.

### 4. Environment file

Drop one canonical env file at `/opt/knowledge-layer/env/api.env`, owned `knowledge:knowledge`, mode `0600`. Start from [`.env.example`](../.env.example) and apply the production rules from [PRODUCTION_HARDENING.md](./PRODUCTION_HARDENING.md):

```ini
# /opt/knowledge-layer/env/api.env
APP_ENV=production
AUTH_MODE=session
SESSION_SECRET=<32 hex bytes>
SESSION_COOKIE_SECURE=true
APP_PUBLIC_URL=https://kl.example.com
DATABASE_URL=postgres://knowledge:<secret>@db.internal:5432/knowledge?sslmode=verify-full
REDIS_URL=redis://redis.internal:6379/0
OPENSEARCH_URL=https://opensearch.internal:9200
NEO4J_URL=bolt://neo4j.internal:7687       # optional
OPS_AUTH_TOKEN=<24 hex bytes>
CORS_ALLOW_ORIGINS=https://kl.example.com
AI_PRIVACY_VAULT_KEY=<base64 of 32 random bytes>
OPENAI_API_KEY=...
JOBWORKER_HEALTH_PORT=:9001
CONNECTORWORKER_HEALTH_PORT=:9002
```

The same env file is loaded by all three binaries — `ValidateAPI` and `ValidateWorker` enforce the production rules at startup.

### 5. systemd units

Three units mirror the three binaries. Workers depend on the API only as a soft ordering hint — they each run their own migration check on startup and operate over the queue independently.

`/etc/systemd/system/knowledge-api.service`:

```ini
[Unit]
Description=Knowledge Layer API
After=network-online.target postgresql.service redis.service
Wants=network-online.target

[Service]
Type=simple
User=knowledge
Group=knowledge
WorkingDirectory=/opt/knowledge-layer
EnvironmentFile=/opt/knowledge-layer/env/api.env
ExecStart=/opt/knowledge-layer/bin/knowledge-api
Restart=on-failure
RestartSec=5s
LimitNOFILE=65535
# Ship structured logs to journald; rely on logrotate for journald, not on app file logs.
StandardOutput=journal
StandardError=journal
# Hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/knowledge-layer /var/log/knowledge-layer
PrivateTmp=true

[Install]
WantedBy=multi-user.target
```

`/etc/systemd/system/knowledge-jobworker.service`:

```ini
[Unit]
Description=Knowledge Layer job worker
After=network-online.target redis.service postgresql.service knowledge-api.service
Wants=network-online.target

[Service]
Type=simple
User=knowledge
Group=knowledge
WorkingDirectory=/opt/knowledge-layer
EnvironmentFile=/opt/knowledge-layer/env/api.env
ExecStart=/opt/knowledge-layer/bin/knowledge-jobworker
Restart=on-failure
RestartSec=5s
StandardOutput=journal
StandardError=journal
NoNewPrivileges=true
ProtectSystem=strict
PrivateTmp=true

[Install]
WantedBy=multi-user.target
```

`/etc/systemd/system/knowledge-connectorworker.service`:

```ini
[Unit]
Description=Knowledge Layer connector worker
After=network-online.target redis.service postgresql.service knowledge-api.service
Wants=network-online.target

[Service]
Type=simple
User=knowledge
Group=knowledge
WorkingDirectory=/opt/knowledge-layer
EnvironmentFile=/opt/knowledge-layer/env/api.env
ExecStart=/opt/knowledge-layer/bin/knowledge-connectorworker
Restart=on-failure
RestartSec=5s
StandardOutput=journal
StandardError=journal
NoNewPrivileges=true
ProtectSystem=strict
PrivateTmp=true

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now knowledge-api knowledge-jobworker knowledge-connectorworker
sudo systemctl status knowledge-api
journalctl -u knowledge-api -f
```

### 6. Logs

Logs go through journald — there is no app-side file logging. Configure journald retention as needed:

```ini
# /etc/systemd/journald.conf.d/knowledge-layer.conf
[Journal]
SystemMaxUse=4G
MaxRetentionSec=14day
```

If you must ship to files for a third-party shipper, use `systemd-journal-upload` or `journalctl --output=json | <shipper>`. Do not introduce app-side log files unless you are also wiring logrotate — divergent rotation paths are a known footgun.

### 7. Web app

Two common deployments:

- **Same host, behind a reverse proxy (nginx / caddy)** — run `npm start` from `/opt/knowledge-layer/web/` under its own systemd unit (`knowledge-web.service`), proxy `https://kl.example.com/` to `http://127.0.0.1:3000`.
- **Static export** — if you can drop server components, `next export` produces a static bundle you can serve from any web server.

Set `NEXT_PUBLIC_API_URL=https://kl.example.com` (same origin avoids CORS) and `NEXT_PUBLIC_USE_DEV_HEADER=false`. For cookie auth the web and API should be on the **same site** so the session cookie is sent on every API call.

A minimal `knowledge-web.service`:

```ini
[Unit]
Description=Knowledge Layer Web
After=network-online.target

[Service]
Type=simple
User=knowledge
Group=knowledge
WorkingDirectory=/opt/knowledge-layer/web
Environment=NODE_ENV=production
Environment=PORT=3000
ExecStart=/usr/bin/node /opt/knowledge-layer/web/node_modules/next/dist/bin/next start
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

### 8. Health and metrics scraping

Once running, point your monitoring at:

- `https://kl.example.com/health` — public liveness, no auth.
- `https://kl.example.com/ops/health` — bearer (`OPS_AUTH_TOKEN`); detailed snapshot.
- `https://kl.example.com/metrics` — bearer; Prometheus text. Includes `knowledge_job_run_duration_seconds`, `connector_sync_duration_seconds`, `postgres_pool_*`, `asynq_queue_*` (Phase 2.2.2).
- `http://<host>:9001/ops/health` — jobworker, bearer.
- `http://<host>:9002/ops/health` — connectorworker, bearer.

The worker `/ops/health` exposes `last_processed_by_task` — the canonical "is this worker stuck or just idle" signal (Phase 2.2.1).

### 9. Backups

Schedule logical dumps to off-host storage:

```bash
# /etc/cron.d/knowledge-layer-backup
0 */6 * * * knowledge \
  pg_dump --format=custom --no-owner --no-acl "$(grep ^DATABASE_URL /opt/knowledge-layer/env/api.env | cut -d= -f2-)" \
  | aws s3 cp - s3://kl-backups/$(date -u +\%Y-\%m-\%d/)/knowledge-$(date -u +\%H\%M\%SZ).dump
```

Test restore quarterly. See [UPGRADE_AND_ROLLBACK.md](./UPGRADE_AND_ROLLBACK.md) for the upgrade-with-snapshot workflow and the destructive-rollback path.

### 10. Upgrades on bare-metal

Follow the single-pod path in [UPGRADE_AND_ROLLBACK.md](./UPGRADE_AND_ROLLBACK.md) §4. Summary:

```bash
sudo systemctl stop knowledge-jobworker knowledge-connectorworker
pg_dump --format=custom --no-owner --no-acl "$DATABASE_URL" > /var/lib/knowledge-layer/backups/pre-upgrade-$(date -u +%Y%m%dT%H%M%SZ).dump
# Build new binaries on the build host, scp to /opt/knowledge-layer/bin/
sudo systemctl restart knowledge-api
journalctl -u knowledge-api -n 100 --no-pager
sudo systemctl start knowledge-jobworker knowledge-connectorworker
bash scripts/smoke-session.sh
```

## First workspace (bootstrap)

If the database has **no domains** (e.g. production without `000008_dev_seed`):

1. **Automatic (API startup):** With **`APP_ENV=local`**, the API creates a first admin + domain by default (`admin@local.test` / `changeme12345`, domain `Default`) unless `AUTO_BOOTSTRAP_INSTANCE=0`. Check logs for `instance-bootstrap:` and the new `user_id` for dev-header mapping. For **staging/production**, set **`AUTO_BOOTSTRAP_INSTANCE=1`** and **`BOOTSTRAP_ADMIN_EMAIL`** + **`BOOTSTRAP_ADMIN_PASSWORD`** (min 8 chars); optional `BOOTSTRAP_DOMAIN_NAME`, `BOOTSTRAP_ADMIN_NAME`. See [CONFIG_ENV.md](./CONFIG_ENV.md).
2. **Manual:** Call **`POST /instance/bootstrap`** (unauthenticated, only when no domains exist) — [ADMIN_GUIDE_V1.md](./ADMIN_GUIDE_V1.md).
3. Or use SQL seeds only in non-production environments.

## Backups

- Back up PostgreSQL on a schedule (logical dumps or volume snapshots).
- Store connector secrets and `SESSION_SECRET` in a secrets manager, not in git.

## Upgrades

Migrations are forward-only and applied automatically on API/worker startup. Read [UPGRADE_AND_ROLLBACK.md](./UPGRADE_AND_ROLLBACK.md) before any non-trivial upgrade — it covers:

- Compatibility matrix (HTTP API, schema, queue payloads, audit events).
- Zero-downtime rolling upgrade for multi-pod deployments.
- Single-pod / bare-metal upgrade with backup snapshot.
- Code-only vs migration-shipped vs destructive rollback paths.
- Postgres-vs-Redis consistency hazards and recovery.
- Known upgrade gotchas (vault key now required in production, dirty-migration recovery, etc.).

## What is configured in UI vs env

| In UI (when logged in as admin) | Typically env / deploy only |
|----------------------------------|-----------------------------|
| Domains, users, grants, roles, invitations | `DATABASE_URL`, `SESSION_SECRET`, TLS |
| Connectors & source feed configs | Some secrets may stay in env per policy |
| Knowledge jobs, governance queues | `OPENSEARCH_URL`, `SMTP_*`, LLM keys |

See also [OPERATIONS.md](./OPERATIONS.md).
