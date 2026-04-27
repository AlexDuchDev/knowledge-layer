# Knowledge jobs engine (technical)

Product contract: [KNOWLEDGE_JOBS.md](./KNOWLEDGE_JOBS.md). This note maps **tables**, **HTTP gates**, **run pipeline**, and **Go entrypoints** in `apps/api`.

## Concepts

| Term | Meaning |
|------|---------|
| **Template** | Catalog entry in `internal/knowledge_jobs/job_templates.go` (`template_id` on create). Sets defaults for `job_type`, `publication_mode`, scope hints, etc. |
| **Job** | Row in `knowledge_jobs` — definition + policy (`publication_mode`, `output_domain_id`, `source_scope_json`). |
| **Trigger** | Row in `job_triggers` — `scheduled`, `event_driven`, `window_based`, `conditional`, etc. **`scheduled`** triggers with `schedule_expr` are evaluated on `knowledge:scheduled_tick` (cron, UTC minute windows; at most one run per job per UTC minute). |
| **Run** | Row in `job_runs` — one execution attempt; `input_scope_snapshot_json` snapshots scope; `execution_metrics_json` holds `processor`, `source_feed_ids_read`, `warnings`. |
| **Output** | Row in `job_outputs` — links run to derived entity / review task / `publication_status`. |

`job_type` on the job row is the **processing class** (e.g. `weekly_digest`); it corresponds to the template’s `job_type` when created from the catalog.

## Tables (migrations)

- `000005_knowledge_jobs.up.sql` — core `knowledge_jobs`, `job_triggers`, `job_runs`, `job_outputs`.
- `000016_architecture_foundation.up.sql` — `knowledge_job_sources`, `knowledge_job_operators`.
- `000021_knowledge_jobs_publication_mode.up.sql` — legacy `publication_mode = 'draft'` → `reviewed_publish`; default column `draft_only` for new rows.
- `000028_job_builder.up.sql` — Job Builder columns on `knowledge_jobs` (`template_key`, `processing_mode`, citation/provenance/scenario exposure, `allow_domain_run_job`, `cloned_from_job_id`) and table `job_builder_presets` (seeded presets merged in `GET /job-builder/presets`).

**Job Builder overview (routes, UI, validation notes):** [job-builder.md](./job-builder.md).

## Publication modes

Canonical values: `draft_only`, `reviewed_publish`, `auto_publish`.

- **`draft_only`**: digest builds a derived entity in **`draft`**, **no** review task; `job_outputs.publication_status` = `draft_only`.
- **`reviewed_publish`**: entity **`pending_review`**, review task created, output `pending_review`.
- **`auto_publish`**: entity **`published`**, no review task (blocked at create time if `review_required` is true).

API responses normalize legacy DB value `draft` on `publication_mode` to **`reviewed_publish`** (historical semantics).

## Definition validation

- `internal/knowledge_jobs/definition_validator.go` — `ValidateCreateJobInput`, `ValidateUpdateJobInput`, `ValidateTriggerInput`, `ValidatePublicationMode`, `ValidateTriggerRowsForPrimaryType`, `ValidateProcessingMode`, unrestricted scope rejection.
- **Scope:** `weekly_digest` requires `source_feed_id` and `domain_id` in `source_scope_json`. Other `job_type` values require at least `domain_id` in scope (definition may still be saved; see orchestrator below).
- **`unrestricted: true` in `source_scope_json`:** rejected for all job types in v1.
- **Output domain:** required when `publication_mode` is not `draft_only`.
- **Triggers vs job row:** non-`manual` `trigger_type` on `knowledge_jobs` requires at least one **active** `job_triggers` row with the same `trigger_type`.

## `job_type` matrix (executor vs catalog)

Catalog templates in `job_templates.go` may advertise many `job_type` strings for inspectable drafts. **Runtime execution** is the source of truth in `processor_capabilities.go` / `IsKnowledgeJobProcessorImplemented` (must match `orchestrator.executeProcessor`).

| `job_type` | Implemented (queued run can execute) | Fail-closed on create | Notes |
|------------|---------------------------------------|-------------------------|--------|
| `weekly_digest` | yes | no | `DigestRunner.RunWeeklyDigest`; requires governed scope + feed policy. |
| `decision_extraction` | yes | no | `DigestRunner.RunDecisionExtraction` |
| `planning_summary` | yes | no | `DigestRunner.RunPlanningSummary` |
| `stale_scan` | yes | no | `DigestRunner.RunStaleScan` |
| `support_trends_extraction` | yes | no | `DigestRunner.RunSupportTrendsExtraction` |
| Other catalog types (`blocker_detection`, …) | no | **yes** — `POST /knowledge-jobs` returns 400 with `job_type has no runtime processor` | Updates (`PATCH`) do not re-check processor so legacy rows stay editable; runs still fail at orchestrator. |

API: `GET /knowledge-jobs/engine-metadata` (authenticated) returns `implemented_job_types` and `job_type_capabilities[]` with `fail_closed_message`. List/detail job JSON includes `processor_implemented`. Templates and `GET /job-builder/presets` include `processor_implemented` on each template and merged preset row.

## Execution orchestrator

- `internal/knowledge_jobs/orchestrator.go` — `runOrchestrator.execute`: validate context → resolve sources (`JobAllowsSourceFeed` for digest) → build input refs / warnings → `executeProcessor` → merge `execution_metrics_json`.
- **Implemented `job_type` values:** `weekly_digest`, `decision_extraction`, `planning_summary`, `stale_scan`, `support_trends_extraction` all route through `DigestRunner` methods in `executeProcessor`. Any other `job_type` **fails the run** with `unsupported job_type` (no silent success). If `DigestRunner` is not configured (`nil`), implemented types fail with `… processor not configured`.
- `internal/knowledge_jobs/jobs.go` — `JobService.executeRun` delegates to the orchestrator (sync and queued worker paths).

## Output policy

- `internal/knowledge_jobs/output_policy.go` — `PlanWeeklyDigestOutput` centralizes lifecycle / review / `job_outputs.publication_status` for the digest processor.
- `internal/knowledge_jobs/digest.go` — `RunWeeklyDigest` uses that plan when creating entities and outputs.

## HTTP authorization (`internal/httpserver/routes_register.go`)

| Route | Gate |
|-------|------|
| `GET /knowledge-jobs/engine-metadata` | Authenticated; executor matrix + capabilities for UI/docs clients. |
| `GET /job-builder/presets` | Authenticated principal; merged template + DB presets. |
| `GET /knowledge-jobs` | Principal required; list filtered by `principalCanViewKnowledgeJob`. Optional `?expand=scenarios` adds binding counts/codes. |
| `GET /knowledge-jobs/:id` | Same view rule. |
| `POST /knowledge-jobs` | `manage_jobs` on `output_domain_id` when set; if `output_domain_id` is omitted (draft-only jobs), owner must equal principal. |
| `PATCH /knowledge-jobs/:id` | `principalCanManageKnowledgeJob` — full definition patch (`PatchJobInput`). |
| `POST /knowledge-jobs/:id/clone` | Manage job. |
| `GET /knowledge-jobs/:id/preview` | View job. |
| `POST /knowledge-jobs/:id/test-run` | `principalMayRunKnowledgeJob`; `dry_run` returns validation + preview JSON. |
| `POST /knowledge-jobs/:id/scenario-bindings` | Manage job; body is a **raw JSON array** of bindings. |
| `GET` / `POST /knowledge-jobs/:id/operators` | View / manage; POST replaces user operator rows. |
| `GET` / `POST /knowledge-jobs/:id/triggers`, `PATCH` / `DELETE /job-triggers/:id` | **`requireManageKnowledgeJobByID`** (same as job manage: owner or `manage_jobs` on output domain) — domain stewards can configure schedules. |
| `GET /knowledge-jobs/:id/runs` | View job. |
| `POST /knowledge-jobs/:id/run` | `principalMayRunKnowledgeJob` — owner or `knowledge_job_operators`, or (if `allow_domain_run_job`) domain `run_job` / `manage_jobs`. |
| `GET /job-runs/:id` | View access to parent job (same as job list/detail). |

See [job-builder.md](./job-builder.md) for narrative and UI mapping.

## Workers

- `knowledge:job_run` — real handler: `JobService.ProcessQueuedRun`.
- `knowledge:scheduled_tick` — `cmd/jobworker` calls `JobService.ProcessScheduledTick`: active `scheduled` triggers whose cron fires in the current UTC minute enqueue `knowledge:job_run` (or run inline when Redis queue is disabled). Idempotent per job per minute via `job_runs.started_at` window.

## Tests

- Unit: `internal/knowledge_jobs/definition_validator_test.go`.
- Integration (set `E2E_DB=1` + `DATABASE_URL`): `internal/integration/knowledge_jobs_governance_test.go`, `job_builder_test.go`, `digest_flow_test.go`, `permission_foundation_test.go` (operator / wrong source).
