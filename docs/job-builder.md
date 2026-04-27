# Job Builder (control plane)

The **Job Builder** is the HTTP + admin UI layer for **knowledge job definitions** stored in `knowledge_jobs` (there is no separate `job_definitions` table). It adds richer validation, preview, cloning, operator and scenario binding APIs, and DB-seeded **builder presets** merged with the in-code template catalog.

Product context: [KNOWLEDGE_JOBS.md](./KNOWLEDGE_JOBS.md). Implementation map: [knowledge-jobs-engine.md](./knowledge-jobs-engine.md).

## Conceptual model

| Concept | Storage / code |
|--------|----------------|
| **Template catalog** | `internal/knowledge_jobs/job_templates.go` → `GET /knowledge-job-templates` |
| **Builder presets** | Table `job_builder_presets` (migration `000028`) → merged in `GET /job-builder/presets` |
| **Job definition** | Row in `knowledge_jobs` (columns include `template_key`, `processing_mode`, `citations_required`, `provenance_required`, `scenario_only_exposure`, `allow_domain_run_job`, `cloned_from_job_id`) |
| **Triggers** | `job_triggers` |
| **Source bindings** | `knowledge_job_sources` (synced from `source_scope_json` on create/update) |
| **Operators** | `knowledge_job_operators` (`principal_type = user` in v1 replace API) |
| **Scenario links** | `scenario_job_bindings` (job-centric replace: `POST /knowledge-jobs/:id/scenario-bindings`) |

## HTTP routes (no `/api` prefix)

| Method | Path | Gate | Notes |
|--------|------|------|--------|
| GET | `/job-builder/presets` | Authenticated principal | Merges DB presets with template catalog |
| GET | `/knowledge-jobs?expand=scenarios` | View filter per job | Returns `scenario_binding_count` and `scenario_codes` when expanded |
| PATCH | `/knowledge-jobs/:id` | `principalCanManageKnowledgeJob` | Full definition patch (`PatchJobInput`), not status-only |
| POST | `/knowledge-jobs/:id/clone` | Manage | New job id; copies triggers, operators, sources; **not** scenario bindings |
| GET | `/knowledge-jobs/:id/preview` | View | `JobPreview` + validation errors/warnings |
| POST | `/knowledge-jobs/:id/test-run` | Run (`principalMayRunKnowledgeJob`) | Body `{ "dry_run": true }` → `{ valid, preview }`; else enqueues run |
| POST | `/knowledge-jobs/:id/scenario-bindings` | Manage | **Raw JSON array** of `{ "scenario_id", "relationship" }` |
| GET | `/knowledge-jobs/:id/operators` | View | Lists operator principals |
| POST | `/knowledge-jobs/:id/operators` | Manage | `{ "user_ids": ["uuid", ...] }` replaces user rows |
| GET/POST/… | `/knowledge-jobs/:id/triggers`, `/job-triggers/:id` | **Manage job** (`requireManageKnowledgeJobByID`) | Domain owners who can manage the job can configure triggers (not platform identity-admin only) |
| GET | `/knowledge-jobs/:id/runs` | View | Run history for operators who may view the job |

## Validation (domain)

`internal/knowledge_jobs/definition_validator.go`:

- **Publication / review** — existing rules (`auto_publish` vs `review_required`, output domain when publishing).
- **Source scope** — job-type-specific scope; **`unrestricted: true` is rejected** for all job types in v1.
- **Triggers vs primary type** — non-`manual` `trigger_type` on the job requires at least one **active** `job_triggers` row with the same `trigger_type` (`ValidateTriggerRowsForPrimaryType`).
## Run permission (`allow_domain_run_job`)

When `allow_domain_run_job` is **false**, `principalMayRunKnowledgeJob` allows only the **owner** and users listed in `knowledge_job_operators` — **not** broad domain `run_job` / `manage_jobs`. When **true**, domain `run_job` / `manage_jobs` applies as before in addition to owner/operators.

## Admin UI

- **List:** `/admin/jobs` (rewrites to `/jobs`) — table with template, triggers, processing, scenario summary (`expand=scenarios`).
- **Detail:** `/admin/jobs/:id` — sections for basic fields, scope/config JSON, triggers, processing mode, output policy, governance flags, operators, scenario bindings, preview, dry run / enqueue.

Navigation label: **Job Builder**.

## Tests

- Unit: `internal/knowledge_jobs/definition_validator_test.go` (unrestricted scope, trigger alignment, operator policy).
- Integration (`E2E_DB=1` + `DATABASE_URL`): `internal/integration/job_builder_test.go` (clone, preview, dry run, owner trigger, scenario bindings).

## API equivalence

If an external spec uses `/api/...`, map to these paths **without** the `/api` prefix (same handlers).
