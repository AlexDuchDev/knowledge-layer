# Production go-live checklist

Operator-facing steps to reach a **production candidate** deployment. Assumes images/builds from this repository. For staging-only runs, skip production-only items marked **(production only)**.

Cross-references: [DEPLOY_CHECKLIST.md](DEPLOY_CHECKLIST.md), [PRODUCTION_HARDENING.md](PRODUCTION_HARDENING.md), [STAGING_SMOKE_TEST.md](STAGING_SMOKE_TEST.md), [ACCESS_MODEL.md](ACCESS_MODEL.md), [`.env.example`](../.env.example).

---

## 1. Required infrastructure prerequisites

- [ ] **PostgreSQL 16** with **pgvector** extension available (same major as migrations; see `docker-compose` / CI image `pgvector/pgvector:pg16`). Self-hosted expectations match [SELF_HOSTED.md](SELF_HOSTED.md) (same Postgres baseline).
- [ ] **Redis** reachable from API and all workers (`REDIS_URL`); used for Asynq job and connector queues.
- [ ] **OpenSearch** (if search/hybrid features are required): cluster with **TLS** and **authentication** appropriate for your network; not the compose profile with `plugins.security.disabled=true`.
- [ ] **Ingress / load balancer** terminating **HTTPS** for the API and web app; TLS certificates valid for public or internal hostnames.
- [ ] **Secret store** (or equivalent) for `DATABASE_URL`, `SESSION_SECRET`, `OPS_AUTH_TOKEN`, SMTP passwords, connector OAuth secrets, and `OPENAI_API_KEY` or `OPENROUTER_API_KEY` (whichever you use for LLM/embeddings).
- [ ] **Process manager** or container orchestration that restarts failed API/worker processes and rolls out config changes safely.

---

## 2. Required environment variables and secrets

Set in the **process environment** of API, **jobworker**, and **connectorworker** (workers: subset as below).

| Variable | Staging | Production | Notes |
|----------|---------|------------|--------|
| `APP_ENV` | `staging` | `production` | Drives [`ValidateAPI` / `ValidateWorker`](../apps/api/internal/config/hardening.go). |
| `DATABASE_URL` | Required | Required | Must be non-empty in env (not only implicit default). |
| `REDIS_URL` | Required | Required | **Fail-closed** for API and workers in staging/production. |
| `CORS_ALLOW_ORIGINS` | Required (API) | Required (API) | Comma-separated; no localhost default in non-local. |
| `AUTH_MODE` | `session` | `session` | `development_header` rejected outside local. |
| `SESSION_SECRET` | Required | Required | Min 16 effective bytes ([`SessionSecretBytes`](../apps/api/internal/config/config.go)). |
| `APP_PUBLIC_URL` | HTTPS recommended | **Must** be `https://…` **(production only)** | Startup fails in production if not HTTPS. |
| `OPS_AUTH_TOKEN` | Optional (see §9) | **Required**, min **16** chars trimmed **(production only)** | Loaded as [`OpsAuthToken`](../apps/api/internal/config/config.go). |
| `SESSION_COOKIE_SECURE` | Default secure when unset | **Must not** be `false`/`0` **(production only)** | Insecure cookies blocked in production. |
| `OPENSEARCH_URL` | If set: `https://…` or `OPENSEARCH_ALLOW_INSECURE_HTTP=1` | Same | `http://` rejected unless explicit insecure opt-in. |
| `AI_PRIVACY_VAULT_KEY` | Recommended (≥32 bytes) | **Required** (32 raw bytes or base64-of-32) **(production only)** | AES-256 key for placeholder vault; `AI_PRIVACY_DEV_PLAINTEXT_STORE=1` is rejected in production. |

Workers (`ValidateWorker`): `DATABASE_URL`, `REDIS_URL`, and OpenSearch URL rules above; in production also `AI_PRIVACY_VAULT_KEY`. No `CORS`/`AUTH_MODE` validation in worker binary—still set `APP_ENV` consistently.

> **Single source of truth for env variables:** [.env.example](../.env.example) and [CONFIG_ENV.md](CONFIG_ENV.md). The table above lists only fields with profile-specific enforcement; for the full list of supported variables (defaults, optional flags, OAuth credentials, blob storage, Second Brain optional module) read CONFIG_ENV.md and avoid duplicating tables across docs.

Frontend: `NEXT_PUBLIC_API_URL` (HTTPS API base), **`NEXT_PUBLIC_USE_DEV_HEADER=false`**.

---

## 3. Auth and session prerequisites

- [ ] `AUTH_MODE=session` on API for `APP_ENV=staging|production` (enforced at startup).
- [ ] Strong `SESSION_SECRET` stored only in secret manager; rotation procedure documented.
- [ ] HTTPS termination in front of API so session cookies marked **Secure** are sent (`SESSION_COOKIE_SECURE` true in production—enforced).
- [ ] No reliance on `X-Principal-User-ID` outside local: middleware allows dev header only when `AUTH_MODE=development_header` **and** `IsLocalDev()` ([`middleware.go`](../apps/api/internal/httpserver/middleware.go)).
- [ ] Web app uses real login/session flow; operators verify a non-admin test user can sign in and call one authenticated read.

---

## 4. OpenSearch prerequisites

- [ ] Production/staging: `OPENSEARCH_URL` uses **`https://`** OR operators explicitly set `OPENSEARCH_ALLOW_INSECURE_HTTP=1` for a **private** network only (documented exception).
- [ ] Cluster authentication, TLS trust, and network policies match your security model (not the local single-node insecure compose image).
- [ ] API and workers use the **same** `OPENSEARCH_URL` and TLS trust configuration.

---

## 5. Deploy steps

1. [ ] Apply infrastructure (DB, Redis, OpenSearch, ingress, secrets).
2. [ ] Inject environment variables / secrets into API and worker workloads per §2.
3. [ ] Deploy **API** image (entrypoint [`cmd/api`](../apps/api/cmd/api/main.go)): runs `MigrateUp` before listen.
4. [ ] Deploy **jobworker** and **connectorworker** (same image, different commands); each runs migrations on startup.
5. [ ] Deploy **web** (optional) with production `NEXT_PUBLIC_*` values.
6. [ ] Confirm no crash loop: `kubectl logs` / `docker compose logs` / platform equivalent—no `config:` validation fatals.

---

## 6. Migration verification

- [ ] API logs: migration step completes without `migrate:` fatal (duplicate version / pgvector missing are common failures).
- [ ] Connect to Postgres: `SELECT version, dirty FROM schema_migrations ORDER BY version DESC LIMIT 5;` — no unexpected `dirty=true`.
- [ ] If upgrading from a legacy DB with duplicate `000011` history, follow [RELEASE_READINESS_AUDIT.md](RELEASE_READINESS_AUDIT.md) §4 reconciliation if needed.

---

## 7. API health verification

```bash
curl -sf "https://<api-host>/health"
```

- [ ] HTTP **200**, JSON includes `"status":"ok"` (no DB check; liveness only).

---

## 8. Worker verification

- [ ] **jobworker** and **connectorworker** processes remain running after startup (no immediate exit).
- [ ] Logs: no `config:`, `migrate:`, `redis url:`, or `deps:` fatals.
- [ ] Redis instance matches API `REDIS_URL` (same logical queue).

---

## 9. Ops and metrics verification

Behavior is implemented in [`routes_health.go`](../apps/api/internal/httpserver/routes_health.go).

**`GET /ops/health`**

| Profile | `OPS_AUTH_TOKEN` | Unauthenticated | With `Authorization: Bearer <token>` |
|---------|------------------|-----------------|--------------------------------------|
| Local | Any | Full detail JSON | N/A (same as local) |
| Staging | **Unset** | **200**, redacted booleans only | If you set token later, bearer required for full detail |
| Staging | Set | **401** without bearer; 200 + full detail with valid bearer | Valid bearer → full detail |
| Production | **Required** at startup | **401** (token always set; anonymous never gets detail) | Valid bearer → full detail |

**`GET /metrics`**

- **Local:** Prometheus exposition without auth (same registry as non-local: Go, process, and `http_requests_total`).
- **Staging / production:** returns **401** unless `OPS_AUTH_TOKEN` is **non-empty** **and** request includes matching `Authorization: Bearer …`.  
  - If staging omits `OPS_AUTH_TOKEN`, `/metrics` is **always 401** (safe default; scrape not available until token is set).

Checks:

```bash
# Production / staging with token: expect 401 without header
curl -s -o /dev/null -w "%{http_code}" "https://<api-host>/metrics"
# Expect: 401

curl -sf -H "Authorization: Bearer $OPS_AUTH_TOKEN" "https://<api-host>/metrics"
# Expect: 200, Prometheus text/OpenMetrics (includes process/runtime and http_requests_total after traffic)
```

---

## 10. Post-deploy smoke checks

- [ ] §7 `/health` OK.
- [ ] §9 `/ops/health` and `/metrics` behave per table (production: always bearer for detail/metrics).
- [ ] Authenticated API call via **session** (e.g. login in web, then call a known read endpoint)—not dev header.
- [ ] Optional: enqueue a harmless job or connector tick and confirm worker log activity (environment-specific).

Full scripted local reference (dev header): [STAGING_SMOKE_TEST.md](STAGING_SMOKE_TEST.md)—**adapt** for session + HTTPS in real staging/production. **Session + HTTPS:** use [`scripts/smoke-session.sh`](../scripts/smoke-session.sh) with `SMOKE_EMAIL` / `SMOKE_PASSWORD` (see STAGING_SMOKE_TEST § Session smoke).

---

## 11. Rollback trigger conditions

Trigger **rollback** or **stop promotion** if any of the following occur during or immediately after deploy:

- API or workers **fail startup** due to `config:` / `ValidateAPI` / `ValidateWorker` (misconfigured env).
- **Migrations** fail or leave `schema_migrations.dirty=true`.
- **`/health`** does not return 200 on the new revision.
- **Session login** broken for standard users (not just admin).
- **OpenSearch** or DB connectivity shows sustained **degraded** in ops health **with** confirmed dependency outage (after ruling out token/bearer mistakes).
- Unexpected **5xx** rate spike or auth **401/403** regression vs baseline.

---

## 12. Go / no-go criteria

**Go** when:

- All §§1–4 prerequisites are satisfied and documented.
- §§5–8 complete without errors; §9 verified for your `APP_ENV` and `OPS_AUTH_TOKEN` policy.
- §10 smoke passes for session auth and critical read paths.
- Rollback path tested (previous image revision or traffic switch).
- Stakeholders accept remaining **product** gaps listed in [RELEASE_READINESS_AUDIT.md](RELEASE_READINESS_AUDIT.md) (e.g. metrics stub, blob store, connector depth).

**No-go** when:

- Any production-only validation would fail (`https` `APP_PUBLIC_URL`, `OPS_AUTH_TOKEN`, secure cookies, `AUTH_MODE=session`).
- OpenSearch or DB URL violates fail-closed rules and is “fixed” only by setting `APP_ENV=local` or insecure flags without approval.
- Secrets committed to repo or logged in plain text.

---

## After go-live

- Monitor logs and dependency health; restrict who holds `OPS_AUTH_TOKEN`.
- Schedule secret rotation for `SESSION_SECRET` and `OPS_AUTH_TOKEN` per policy.
