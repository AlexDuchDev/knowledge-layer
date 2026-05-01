# Upgrade and rollback

Operator runbook for moving a self-hosted Knowledge Layer instance from one release to the next, and recovering when an upgrade goes wrong.

This complements [SELF_HOSTED.md](./SELF_HOSTED.md) (initial install), [PRODUCTION_GO_LIVE_CHECKLIST.md](./PRODUCTION_GO_LIVE_CHECKLIST.md) (first cutover), and [OPERATIONS.md](./OPERATIONS.md) (day-2). Read those first if you are not already running.

## TL;DR

- Knowledge Layer migrations are **forward-only and applied automatically** on API/worker startup.
- Aim for **backward-compatible** schema changes per release so old API pods keep working during a rolling deploy.
- Upgrade order: **drain workers → backup Postgres → deploy API (one pod at a time) → deploy workers → smoke**.
- Rollback strategy depends on whether the migration was backward-compatible (just redeploy the previous binary) or destructive (restore + apply only-current).

## 1. Pre-upgrade

1. **Read the release notes** — `CHANGELOG.md` between your current tag and the target tag. Look for entries marked `BREAKING:` or any explicit migration note (env var changes, schema changes that drop data, deleted endpoints).
2. **Check `docs/LIMITATIONS.md`** — verify nothing you depend on dropped from "supported" to "stubbed" or vice versa.
3. **Confirm a working backup**:
   ```bash
   pg_dump --format=custom --no-owner --no-acl \
     "$DATABASE_URL" > backups/knowledge-$(date -u +%Y%m%dT%H%M%SZ).dump
   ```
   Test the dump can be restored on a scratch database. A backup you have not tested is not a backup.
4. **Audit the queue depth** — if `asynq_queue_pending` (Phase 2.2.2 metric) is non-zero on `connectors`, `ingestion`, or `ai`, decide whether to drain before upgrading or accept that those tasks resume on the new version.
5. **Capture the current version**:
   ```bash
   curl -s -H "Authorization: Bearer $OPS_AUTH_TOKEN" https://api.example/ops/health
   git -C /path/to/checkout rev-parse --short HEAD
   ```

## 2. Compatibility matrix

The repo ships these contracts; understand which apply to your jump:

| Contract | Where defined | Compatibility window |
|---|---|---|
| HTTP API | `apps/api/internal/httpserver/*.go`, declared in [API_STABILITY.md](./API_STABILITY.md) | v0.x: breaking changes allowed between minors. v1+: semver, deprecation cycle of 2 minors. |
| Database schema | `apps/api/internal/db/migrations/` | Forward-only. New migrations should be backward-compatible with the previous minor's binaries (additive columns, no `DROP NOT NULL`-without-default-on-new-rows surprises). |
| Asynq task payloads | `apps/api/internal/platform/queue/tasks_*.go` | New fields are tolerated by older workers; renames break consumption — release them in two steps (add new, deprecate old, remove old next major). |
| Audit event types | `audit_events.event_type` strings | Append-only; never repurpose an existing string. |
| Optional module env flags | `.env.example`, [CONFIG_ENV.md](./CONFIG_ENV.md) | Stable across minors. Removing a flag is a breaking change. |

## 3. Zero-downtime rolling upgrade (recommended)

Assumes you run **at least 2 API pods** + **separate worker pods** behind a load balancer, all sharing one Postgres + Redis.

### Step 1 — Pause queue scheduling

To avoid mixed-version handlers consuming new payloads, **stop the workers first**:

```bash
# Container orchestrator examples — adapt to your platform.
kubectl scale deployment/jobworker --replicas=0
kubectl scale deployment/connectorworker --replicas=0
```

In-flight tasks finish; queues build up briefly. The API keeps serving reads and accepting writes.

### Step 2 — Snapshot Postgres

```bash
pg_dump --format=custom --no-owner --no-acl "$DATABASE_URL" > backups/pre-upgrade-$(date -u +%Y%m%dT%H%M%SZ).dump
```

Take it even if your platform offers automated backups. Migrations run on the next pod start; this is your point-in-time pin.

### Step 3 — Roll the API

Update one API pod at a time. The first pod to start applies pending migrations (`db.MigrateUp`); subsequent pods see the schema already at the target version and skip migration.

```bash
# Example with kubectl
kubectl set image deployment/api api=ghcr.io/<org>/knowledge-api:v0.2.0
kubectl rollout status deployment/api
# Wait for at least one pod ready before continuing.
```

Health-check the new pods:

```bash
curl -s -H "Authorization: Bearer $OPS_AUTH_TOKEN" https://api.example/ops/health
# expect status:"ok", database:"ok", opensearch:"ok"
```

If the rollout stalls (new pods crash on `ValidateAPI`, new migration fails), see Section 5 — Rollback.

### Step 4 — Roll the workers

```bash
kubectl set image deployment/jobworker jobworker=ghcr.io/<org>/knowledge-api:v0.2.0
kubectl set image deployment/connectorworker connectorworker=ghcr.io/<org>/knowledge-api:v0.2.0
kubectl scale deployment/jobworker --replicas=2
kubectl scale deployment/connectorworker --replicas=2
```

Watch their `/ops/health` (Phase 2.2.1) — `last_processed_by_task` should resume advancing within a minute or two as queues drain.

### Step 5 — Post-upgrade smoke

```bash
bash scripts/smoke-session.sh   # session-auth flavour for staging/prod
```

Plus targeted curls for any feature called out in CHANGELOG. If `make test` is available against staging, run a subset that exercises the changed surface.

## 4. Single-pod / single-host upgrade (compose, bare-metal)

When you only have one API pod (or a docker-compose host), zero-downtime is not possible — accept a short window:

```bash
# 1. Drain workers
docker compose stop jobworker connectorworker

# 2. Backup
pg_dump --format=custom --no-owner --no-acl "$DATABASE_URL" > backups/$(date -u +%Y%m%dT%H%M%SZ).dump

# 3. Pull the new release tag
git fetch --tags
git checkout v0.2.0

# 4. Restart everything
docker compose pull
docker compose up -d --force-recreate api jobworker connectorworker

# 5. Smoke
curl -s http://localhost:18080/health
bash scripts/smoke-local.sh
```

Expect 30–60 seconds of API unavailability while migrations apply on first start.

## 5. Rollback

The strategy depends on what changed.

### A) Code-only rollback (no migration shipped)

If the bad release added no new migration, you can re-deploy the previous binary directly:

```bash
kubectl set image deployment/api api=ghcr.io/<org>/knowledge-api:v0.1.9
kubectl set image deployment/jobworker jobworker=ghcr.io/<org>/knowledge-api:v0.1.9
kubectl set image deployment/connectorworker connectorworker=ghcr.io/<org>/knowledge-api:v0.1.9
```

Done. Schema is unchanged.

### B) Migration shipped but is backward-compatible

Most additive migrations (new column, new table, new index) leave the prior binary functional. Re-deploy the previous binary; new schema sits idle until you redeploy a forward-fix.

```bash
kubectl set image deployment/api api=ghcr.io/<org>/knowledge-api:v0.1.9
# Old binary ignores the new column / new table.
```

Do **not** roll the migration back manually unless you have a documented down-script — the project does not ship down migrations.

### C) Destructive or breaking migration

If the release renamed/dropped a column, changed a constraint that the old binary now violates, or introduced a payload-incompatible queue task:

1. **Stop the new release.**
2. **Restore the snapshot from Section 3 step 2** to a new Postgres database (or in-place if downtime is acceptable):
   ```bash
   createdb knowledge_restore
   pg_restore --no-owner --no-acl \
     -d "postgres://$USER:$PASS@$HOST:$PORT/knowledge_restore" \
     backups/pre-upgrade-...dump
   ```
3. **Redirect the cluster** to the restored DB (rotate `DATABASE_URL`).
4. **Re-deploy the previous binary**.
5. **Drain Redis** of any payloads the previous binary cannot decode:
   ```bash
   # nuclear option for non-critical environments
   redis-cli -u "$REDIS_URL" FLUSHDB
   ```
   For production, walk the queues with `asynq` CLI to inspect first.

This path is destructive — you lose any writes between the snapshot and now. Communicate accordingly.

## 6. Backup ↔ Redis consistency

Postgres and Redis evolve independently. A point-in-time restore of Postgres can leave Asynq holding tasks that reference rows the restore deleted (orphan task → handler errors out → automatic retry).

Mitigations:

- **Drain workers + flush Redis** before a destructive Postgres restore (Section 5C).
- For production, prefer **logical migration with downtime** over PIT-restore: announce a window, stop writes, snapshot, apply schema, resume.
- The Asynq inspector exposes archived/dead-letter queues — review them after recovery and either re-enqueue or accept loss.

## 7. Verification after every upgrade

Mandatory:

- [ ] `GET /health` returns ok on every API pod.
- [ ] `GET /ops/health` (with bearer) shows `database:"ok"` and `opensearch:"ok"`.
- [ ] Worker `/ops/health` (`:9001`, `:9002`) shows `last_processed_by_task` advancing.
- [ ] `GET /ops/failed-runs` shows no spike in failures relative to baseline.
- [ ] `GET /metrics` exposes `knowledge_job_run_duration_seconds` and `connector_sync_duration_seconds` for the last 5 minutes.
- [ ] One end-to-end Ask returns an answer with citations.

Recommended:

- [ ] `audit_events` shows `vault.placeholder_stored` after an Ask that uses sensitive content (Phase 1.1.3 hardening).
- [ ] `ValidateAPI` did not regress: API logs show no `WARNING`-level startup messages about missing required env.

## 8. Known upgrade hazards

| Symptom | Likely cause | Recovery |
|---|---|---|
| API fails startup with `AI_PRIVACY_VAULT_KEY required in production` | Phase 1.1.2 hardening — vault key now mandatory in prod. | Set `AI_PRIVACY_VAULT_KEY` (32 bytes raw or base64 of 32) in env; do not enable `AI_PRIVACY_DEV_PLAINTEXT_STORE`. |
| Worker startup loops on `migrate: dirty version N` | Previous run aborted mid-migration. | `pg_dump` for safety, then on a maintenance window: `migrate -database "$DATABASE_URL" -path apps/api/internal/db/migrations force N` (current version); restart. |
| Asynq queue depth growing without bound | Worker stuck (DB lock, exhausted LLM key, OpenSearch down). | Check worker `/ops/health` `last_processed_by_task`. Restart worker; if persistent, check `/ops/failed-runs`. |
| `/control-plane/setup/session/[id]` 404 after upgrade from <2026-04-25 | Phase 2.1.5 removed the rewrite to legacy `/admin/setup/[sessionId]`. | Re-bookmark to canonical CP URL; sessions still load by id. |
| API fails startup with `MCP_ENABLED=true requires OAUTH_PROXY_ENABLED=true` (v0.5.1+) | MCP endpoint is bearer-gated; without the proxy there's no way to issue valid tokens. | Either set `OAUTH_PROXY_ENABLED=true` + the OIDC keys ([CONFIG_ENV.md](CONFIG_ENV.md)), or set `MCP_ENABLED=false` and roll back. |
| API fails startup with `OAUTH_SECRET_KEY must be at least 32 bytes` (v0.5.0+) | OAuth proxy enabled but key missing/short. | Set ≥32 bytes (`openssl rand -hex 32`); restart. |
| MCP clients receive 401 every request after rotating `OAUTH_SECRET_KEY` (v0.5.0+) | Documented behaviour — rotation invalidates every issued JWT. | Operator: no action; Claude Desktop / Cursor silently re-auth via IDP on next call. |

## 9. Per-migration upgrade notes

Subset of recent migrations with operator-relevant nuances. The full list lives at `apps/api/internal/db/migrations/`.

### 000041 — chunks_normalized_record_source (v0.3.0)

- Adds polymorphic `chunks` rooted in either `entities` or `normalized_records`. CHECK constraint enforces exactly-one-source per row.
- Adds `normalized_records.chunks_rebuilt_at` column for the connectorworker's 30-s backfill loop.
- **Forward-compatible.** v0.2.x binaries still see only entity-rooted chunks (the new column NULL means "pending"); v0.3.0+ binaries fill it.
- **Rollback:** drops every normalized_record-rooted chunk and the new column. Operators who already produced chat / docs / meeting chunks under v0.3.0 must accept loss of those embeddings — re-ingestion or `kltools reindex --all-pending-records --yes` rebuilds them after the next forward upgrade.

### 000042 — entity_summarize_projection (v0.4.0)

- Adds `entity_search_projection.synthesized_summary TEXT` + `synthesized_at TIMESTAMPTZ`. Both nullable; existing rows untouched.
- Adds partial index for the entity_summarize backfill query.
- **Forward-compatible.** v0.3.x binaries ignore the new columns. v0.4.0+ binaries populate them only when `entity_summarize` jobs run via API or `kltools summarize`.
- **Rollback:** drops the columns. No data loss outside the synthesized summaries themselves; on next forward upgrade, re-run `kltools summarize --yes` to rebuild.

### 000043 — oauth_clients (v0.5.0)

- Adds `oauth_clients` table for RFC 7591 dynamic-registration (one row per registered MCP client). `client_secret_hash` is bcrypt; secret never stored plaintext.
- **Forward-compatible.** v0.4.x binaries don't read this table; it just sits empty.
- **Rollback:** drops the table. Registered MCP clients lose their entry and must re-register on next forward upgrade. JWT bearers issued before rollback continue to work for their 1-h lifetime against any v0.5.0+ replica that still holds the original `OAUTH_SECRET_KEY`.

### 000044 — openapi_v3_connector (v0.6.0)

- INSERT ONLY — no DDL. Adds one row to `connectors` for the new `openapi_v3` connector type at id `…0014`.
- **Forward-compatible.** v0.5.x binaries don't have the adapter, so any feed pointed at this connector_id returns "unknown connector type" at activation. v0.6.0+ binaries register the adapter and the feed becomes valid.
- **Rollback:** deletes the row. Existing source_feeds with `connector_id` pointing at it become orphaned (validation fails); operator must `archive` or `delete` them before the rollback is clean.

### 000045 — manual_connector (v0.7.0)

- INSERT ONLY — no DDL. Adds one row to `connectors` for the new `manual` connector type at id `…0015`. Active from day zero (`status='active'`) since the connector needs no per-instance configuration.
- **Forward-compatible.** v0.6.x binaries don't have the manual adapter; existing `manual` source_feeds (none in v0.6.x deployments) would be unreachable but no API surface tries to reach them. v0.7.0+ binaries register the adapter and `/api/manual/*` routes become operative.
- **Toolchain bump:** v0.7.0 also bumps `apps/api/go.mod` and `go.work` to **Go 1.26**. Operators building from source need Go 1.26 installed. Release CI's `actions/setup-go@v5` reads `go-version-file` and installs the right version automatically; no CI yaml change needed. Container images ship a 1.26 build.
- **Body limit:** `apps/api/cmd/api/main.go` raises Fiber's `BodyLimit` from 4 MiB (default) to 64 MiB so multipart uploads up to the in-handler 50 MiB cap fit. Other endpoints unaffected; the 64 MiB only matters when the request body actually approaches it.
- **Rollback:** deletes the connector row. Existing manual collections (each = one source_feed) become orphaned; operator must archive them through the v0.7.0 UI **before** rollback, otherwise feed validation fails on next read.

## Related

- [SELF_HOSTED.md](./SELF_HOSTED.md) — install + first run
- [PRODUCTION_HARDENING.md](./PRODUCTION_HARDENING.md) — startup validation rules
- [PRODUCTION_GO_LIVE_CHECKLIST.md](./PRODUCTION_GO_LIVE_CHECKLIST.md) — first-cutover checklist
- [OPERATIONS.md](./OPERATIONS.md) — health, audit, observability day-2
- [API_STABILITY.md](./API_STABILITY.md) — what callers may rely on across releases
- [CONFIG_ENV.md](./CONFIG_ENV.md) — full env-var reference
