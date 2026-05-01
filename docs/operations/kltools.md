# `kltools` — operator CLI

`kltools` is the operator binary shipped inside the `knowledge-layer-api` image (v0.4.0+). It runs against the same composition root as the API server (`app.NewDeps`) but exits after a single subcommand instead of starting an HTTP server. Use it for backfills, schema introspection, and on-demand maintenance jobs.

> **Safe-by-default.** All write subcommands require `--yes` on first invocation. Without it, the command prints the plan and exits.

---

## Where it lives

| Location | What |
|---|---|
| `apps/api/cmd/kltools/` | Source — pattern matches `cmd/jobworker` minus the asynq.Server loop |
| `/app/knowledge-tools` inside `ghcr.io/.../knowledge-layer-api:vX.Y.Z` | Compiled binary, alongside `knowledge-api` / `knowledge-jobworker` / `knowledge-connectorworker` |
| `kltools` build target in `Makefile` | `make test` includes a build of the binary |

Operator usage:

```bash
# kubernetes
kubectl exec deploy/knowledge-layer-api -- /app/knowledge-tools schema-info

# docker compose
docker compose exec api /app/knowledge-tools schema-info

# bare-metal
/usr/local/bin/knowledge-tools schema-info
```

---

## Common flags

| Flag | Default | Description |
|---|---|---|
| `--yes` | — | Mandatory confirmation for write subcommands. Without it, dry-run only. |
| `--max-rows` | 100 | Per-run row cap (where applicable; hard cap 500). |
| `--max-concurrency` | 4 | Parallel LLM calls (`summarize` only). Keep low to bound cost. |

A small DB pool (4 connections) is used so the CLI can never starve the running API.

---

## Subcommands

### `summarize`

Backfills `entity_search_projection.synthesized_summary` via the `entity_summarize` knowledge job processor. Routes through the privacy gateway (`PromptTemplateID="entity_summarize.v1"`).

```bash
# dry-run (default)
kltools summarize --max-rows 50

# actually summarize, scope to one domain
kltools summarize --domain 32000000-0000-0000-0000-000000000001 --max-rows 200 --yes

# re-summarize specific entities (bypasses --max-rows)
kltools summarize --entity 7f6e... --entity a4b1... --yes
```

| Flag | Description |
|---|---|
| `--max-rows` | Cap entities processed per run. Hard cap 500. |
| `--domain` | Restrict to one domain UUID. |
| `--entity` | Process a specific entity UUID. Repeatable. Bypasses `--max-rows`. |
| `--yes` | Required for actual writes. Without it, prints the scope JSON and exits. |

**Prerequisites.** Set `OPENAI_API_KEY` (or `OPENROUTER_API_KEY`) before running. Without an LLM client the command errors clearly rather than silently producing empty summaries.

**Cost discipline.** Each entity = one LLM call + one rehydrate. For 10K entities at ~$0.001/call you're at ~$10. Budget accordingly; lower `--max-rows` if running cost-bound.

### `reindex`

Rebuilds chunks for an entity or drains the v0.3.0 normalized-record backfill queue. Re-enqueues embedding tasks when Redis is configured.

```bash
# dry-run (default)
kltools reindex --entity 7f6e...
kltools reindex --all-pending-records

# actually reindex
kltools reindex --entity 7f6e... --yes
kltools reindex --all-pending-records --batch-size 200 --yes
```

| Flag | Description |
|---|---|
| `--entity` | Rebuild chunks for one entity UUID. |
| `--all-pending-records` | Drain `normalized_records` rows where `chunks_rebuilt_at IS NULL`. |
| `--batch-size` | 1–500 rows per invocation when draining (default 100). |
| `--yes` | Required for actual writes. |

The connectorworker runs the same drain on a 30-s timer per [v0.3.0 chunk-decoupling](../../CHANGELOG.md). Use `kltools reindex --all-pending-records` only to force-converge on demand (e.g., right after an initial bulk ingestion).

### `schema-info`

Read-only pipeline-stage counts plus the registered job-type and connector inventories. Safe in any environment.

```bash
kltools schema-info
```

Sample output:

```
=== Pipeline stage counts ===
  raw_artifacts                    16
  normalized_records               16
    pending chunk rebuild          0
  entities                         8
    pending synthesized_summary    8
  chunks                           24
    entity-rooted                  0
    normalized_record-rooted       24
  embeddings                       0
  audit_events                     34
  source_feeds                     24
  connectors                       20

=== Implemented job types ===
  - weekly_digest
  - decision_extraction
  - planning_summary
  - stale_scan
  - support_trends_extraction
  - entity_summarize

=== Registered connectors ===
  - asana                Asana                          [draft]
  - confluence           Confluence                     [draft]
  - filesystem           Filesystem                     [draft]
  ... (etc)
  - openapi_v3           OpenAPI v3 (generic)           [draft]

=== Cache state ===
  cache: constructed (Cache.Null when CACHE_L1_ENABLED=false)
```

Useful for triage: if `pending chunk rebuild` is stuck > 0 for minutes, the connectorworker is unhealthy. If `pending synthesized_summary` is the entity count, you haven't run `kltools summarize --yes` yet.

---

## Common ops scenarios

### Bootstrap a freshly-onboarded operator

```bash
# 1. Migrate is automatic on API startup.
# 2. Confirm pipeline shape:
kltools schema-info

# 3. After first ingestion, force chunk convergence:
kltools reindex --all-pending-records --batch-size 500 --yes

# 4. Backfill summaries:
OPENAI_API_KEY=sk-... kltools summarize --max-rows 500 --yes
```

### Re-summarize entities after editing a prompt

```bash
# Drop synthesized_summary from a domain and re-fill from scratch.
docker compose exec postgres psql -U knowledge -d knowledge -c "
  UPDATE entity_search_projection SET synthesized_summary = NULL, synthesized_at = NULL
  WHERE entity_id IN (SELECT id FROM entities WHERE domain_id = '<uuid>')
"
OPENAI_API_KEY=sk-... kltools summarize --domain <uuid> --max-rows 500 --yes
```

### Diagnose a stuck connector

```bash
kltools schema-info | grep -A1 'pending chunk rebuild'
# Non-zero for > 5 min → connectorworker not draining.
# Force a one-off drain to see if it's a config error vs a worker outage:
kltools reindex --all-pending-records --batch-size 50 --yes
```

---

## Safety notes

- `summarize` and `reindex` write through the same code paths as the API and the connectorworker. There is no privileged mode.
- `kltools` does **not** bypass `AccessEvaluator`. Job processors don't need a principal because they're system-actor; UI-facing reads still require a session.
- The DB pool cap (4 conns) prevents starving the running API even if you accidentally launch 10 CLI instances in parallel.
- `--dry-run` is implicit on first invocation — pass `--yes` only when you've reviewed the printed scope.

---

## Related docs

- [`docs/operations/mcp.md`](mcp.md) — MCP endpoint operator guide
- [`docs/CONFIG_ENV.md`](../CONFIG_ENV.md) — env-var table (LLM keys, cache flags)
- [`apps/api/internal/knowledge_jobs/processor_capabilities.go`](../../apps/api/internal/knowledge_jobs/processor_capabilities.go) — registered job-type list
