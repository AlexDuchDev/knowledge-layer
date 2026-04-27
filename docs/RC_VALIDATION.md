# Release-candidate validation runbook

The release CI publishes container images to GHCR on every `vX.Y.Z[-rcN]` tag. Before promoting an `-rcN` candidate to a stable `vX.Y.Z` tag, run this runbook end-to-end against the **published** images — not against a fresh source build. The point is to validate the artifact an external operator would actually pull.

This is the gate between [Phase R2 (publish RC)](../README.md) and [Phase R4 (cut stable v0.1.0)] in the v0.1.0 production-release plan.

## Prerequisites

- Docker Desktop or compatible engine running locally (compose v2).
- The GHCR images for the candidate tag exist (workflow `.github/workflows/release.yml` succeeded).
- For a **private** repo (typical pre-launch state), authenticate Docker to GHCR before pulling. With [`gh`](https://cli.github.com/) logged in:

  ```bash
  echo "$(gh auth token)" | docker login ghcr.io -u "$(gh api user --jq .login)" --password-stdin
  ```

  For a public repo this is optional — anonymous pulls work.

- Stop any locally-built `docker compose up` stack to free port 18080:

  ```bash
  docker compose down
  ```

## 1. Pull and bring up the stack

The `docker-compose.rc.yml` override at the repo root replaces the four services that have a local build context (`api`, `jobworker`, `connectorworker`, `web`) with pulls from GHCR. Postgres, Redis, OpenSearch, Neo4j stay at their pinned upstream images.

```bash
# Choose the tag and owner. Defaults work for AlexDuchDev / current rc.
export GHCR_OWNER=alexduchdev
export RC_TAG=v0.1.0-rc2

docker compose -f docker-compose.yml -f docker-compose.rc.yml pull
docker compose -f docker-compose.yml -f docker-compose.rc.yml up -d
docker compose ps
```

Expect: 7 services Up, the four service rows reporting an `IMAGE` column starting with `ghcr.io/...`.

```bash
curl -sfS http://localhost:18080/health
# {"status":"ok"}
```

If `/health` returns within 30 s and `docker compose ps` shows everything healthy, the pulled images run end-to-end.

## 2. Smoke checklist (in order)

Run each step and capture observed deviations in `docs/rc-validation-log.md` (a throwaway local file — do not commit).

### 2.1 Header-auth API smoke

```bash
API_BASE=http://localhost:18080 bash scripts/smoke-local.sh
# expect: smoke-local: ok
```

### 2.2 Bootstrap (UI)

Open `http://localhost:13000/bootstrap`. Complete the first-admin form. The instance should reach `/` with the home dashboard rendered.

### 2.3 Source feed creation (no-creds path)

In Control Plane → Sources → New, pick a connector type that needs no external credentials. Either works:

- **Filesystem** — point at `/data/example.txt` (mounted read-only into the api container).
- **HTTP URL** — point at any small public HTML page.

Save. The wizard should succeed (config validation is the new Phase 4.2.2 inline check).

### 2.4 Job run trigger

Control Plane → Jobs → Runs → "New run". Pick `weekly_digest` (one of the five wired processors). Submit.

Expected:
- Run row shows up immediately.
- Within ~30 s, the run state advances past `queued` (jobworker is processing).
- Open the run detail. Status should be `succeeded` or, at worst, a clearly-described domain-empty failure (not a crash).

### 2.5 Search

Visit `/search` and search for any term that appears in your seeded source feed. Results render without 500.

### 2.6 Ask

Visit `/ask`. Ask a one-line question. Without an LLM provider configured, the response should be a graceful "no provider configured" message — **not** a 500 or a panic. (If you have an LLM provider env-var configured, expect a real answer.)

### 2.7 Governance hub

Visit `/control-plane/governance`. Each of the six queue cards (Reviews, Approvals, Stale, Failed jobs, Failed syncs, Policy exceptions) should load (likely empty) without 500s.

### 2.8 Worker `/ops/health`

The worker `/ops/health` ports default to `9001` (jobworker) and `9002` (connectorworker), but these are **not published** in `docker-compose.yml` for the bundled stack. Validate by exec'ing into the workers:

```bash
docker compose exec jobworker /bin/sh -c 'wget -qO- http://localhost:9001/ops/health'
docker compose exec connectorworker /bin/sh -c 'wget -qO- http://localhost:9002/ops/health'
```

Expect a JSON payload with `db: "ok"`, queue depths, and per-task last-completed timestamps. (The bundled image includes `wget` via the BusyBox shell; if that fails, fall back to `curl` or skip — the `/health` endpoint on the api container is the public liveness signal.)

## 3. Audit-event spot check

After steps 2.3–2.6, the audit table should contain entries for the operations you triggered:

```bash
docker compose exec postgres psql -U knowledge -d knowledge -c "
  SELECT event_type, count(*) AS n
  FROM audit_events
  WHERE created_at > now() - interval '1 hour'
  GROUP BY 1
  ORDER BY 2 DESC, 1;
"
```

Expected event types (subset):
- `ingestion.artifact_processed` — at least one row from your source-feed sync.
- `entity.published` or `entity.created` — depending on what the wizard surfaced.
- `vault.placeholder_*` — only if you actually invoked an LLM-bound flow with provider configured.

A run that shows zero events at all is a regression — the audit pipeline is supposed to fire on every governed action.

## 4. Tear-down

```bash
# Stops + removes containers but keeps the named volumes (Postgres data persists).
docker compose -f docker-compose.yml -f docker-compose.rc.yml down

# Add `-v` only after you have copied any audit-event observations elsewhere — `-v`
# nukes the Postgres volume and you lose the validation evidence.
docker compose -f docker-compose.yml -f docker-compose.rc.yml down -v
```

## 5. Decide

Per the v0.1.0 release plan:

- **Zero blockers** → proceed to R4 (cut stable `v0.1.0`).
- **Critical bug** (smoke fails, golden path broken, container crashes) → fix in source, push, tag the next `v0.1.0-rcN+1`, repeat this runbook.
- **Minor cosmetic issue** (typo, empty state copy, missing UI polish) → log as a `v0.1.1` issue, proceed to R4 anyway.

## What this runbook deliberately does NOT validate

- **External SaaS connectors** (Slack, Notion, Google Drive, etc.) — they require real OAuth / PATs that operators provide. The Filesystem / HTTP URL paths in §2.3 are the no-credentials substitute.
- **LLM provider behaviour** — the Ask step (§2.6) verifies the no-provider graceful-failure path; live LLM behaviour is a separate provider-specific check.
- **Multi-pod deployment / Kubernetes manifests** — those have their own upgrade-path validation in [`UPGRADE_AND_ROLLBACK.md`](UPGRADE_AND_ROLLBACK.md).
- **Performance** — RC validation is correctness, not load. SLO checks are operator-specific per [`ALERTING_PLAYBOOK.md`](ALERTING_PLAYBOOK.md).

## Related

- [`.github/workflows/release.yml`](../.github/workflows/release.yml) — the release CI that publishes the artifacts validated here.
- [`docker-compose.rc.yml`](../docker-compose.rc.yml) — the published-image override used in this runbook.
- [`docs/RELEASING.md`](RELEASING.md) — full release flow including the R4 stable-tag step.
- [`docs/UPGRADE_AND_ROLLBACK.md`](UPGRADE_AND_ROLLBACK.md) — what operators run when adopting a new tag in a real deployment.
