# Scenario Builder (control plane)

## Purpose

Scenario Builder is the **productized usage layer**: reusable patterns for how real roles interact with organizational memory (Ask, digests, process follow-ups, explorers, governance queues). It is **not** a raw prompt menu, **not** a saved search, and **not** the same thing as a knowledge job—though scenarios may **bind** to zero or more jobs.

See also: [role-builder.md](./role-builder.md), [KNOWLEDGE_JOBS.md](./KNOWLEDGE_JOBS.md).

## Conceptual distinction

| Concept | Meaning |
|--------|---------|
| **Role** | Who may see, run, manage, or publish (Role Builder + access evaluator). |
| **Scenario** | A reusable **usage pattern** (purpose, inputs, trigger, processing intent, outputs, governance, UI surfaces). |
| **Job** | Execution recipe / automation (`knowledge_jobs` + triggers/runs). |
| **Scenario binding** | How a scenario connects to roles, sources, jobs, and UI. |
| **Scenario output policy** | Governance for what the scenario produces (review, publication, citations, provenance, sensitivity). |

Scenarios **do not** default to unrestricted corpus access: `input_scope_json` is validated and auditable; runtime enforcement must still intersect with role grants and domain policy.

## Scenario types (`scenario_type`)

| Type | Intent |
|------|--------|
| `ask` | Scoped Q&A over allowed knowledge (interactive). |
| `digest` | Recurring summaries (often scheduled). |
| `process` | Reacts to organizational process moments (manual/event). |
| `explorer` | Structured navigation (memory pages, decision explorer). |
| `governance` | Quality and compliance surfaces (queues, reviews). |

## Configuration dimensions (stored fields)

- **Identity**: `code`, `name`, `description`, `scenario_type`, `active`, `notes`, owners.
- **Intended users**: `target_role_scope_json` (intent) + **`scenario_role_bindings`** (enforceable flags).
- **Inputs**: `input_scope_json` + **`scenario_source_bindings`** (optional explicit feeds).
- **Trigger**: `trigger_type`, `trigger_config_json` (`scheduled` requires `schedule_expr`, etc.).
- **Processing**: `processing_mode` (e.g. ask, summarize, explore).
- **Output**: `output_mode`, `ui_surface`, **`scenario_ui_bindings`**, **`scenario_output_policies`**.
- **Jobs**: **`scenario_job_bindings`** (`primary_support` \| `supports` \| `optional`) — **many-to-many** with `knowledge_jobs`.

## Data model (tables)

| Table | Role |
|-------|------|
| `scenario_presets` | Catalog templates (`preset_key`, `template_json`) for “create from preset”. |
| `scenario_definitions` | Live scenario rows (including seeded system presets with `is_preset = true`). |
| `scenario_output_policies` | 1:1 policy row per scenario (CASCADE with definition). |
| `scenario_role_bindings` | Per-role: `can_see`, `can_run`, `can_manage`, `can_review_publish`. |
| `scenario_source_bindings` | Optional `source_feed_id` links. |
| `scenario_job_bindings` | Optional `knowledge_job_id` links. |
| `scenario_ui_bindings` | Surface keys, nav grouping, sort order. |

## Role mirror (`role_scenario_bindings`)

When **`scenario_role_bindings.can_see`** is set, Scenario Builder **replaces** mirror rows for that scenario’s **`code`** in `role_scenario_bindings` (`scenario_key` = `scenario_definitions.code`). This keeps Role Builder previews (`allowed_scenarios`) aligned with Scenario Builder visibility.

## Preview model (`GET /scenarios/:id/preview`)

Returns a single JSON object including: definition fields, flattened **governance_summary**, **visible_roles** (from role bindings), **source_bindings**, **job_bindings** (with job names when present), **ui_bindings**, and embedded **output_policy**.

## Presets (catalog + seeded scenarios)

`scenario_presets` holds nine templates; migration also seeds matching **`scenario_definitions`** rows (`is_preset = true`) with policies and UI bindings:

- `ask_allowed_knowledge`
- `weekly_team_digest`
- `planning_summary`
- `retro_summary`
- `project_memory_page`
- `decision_explorer`
- `executive_weekly_brief`
- `support_trends_digest`
- `governance_review_queue`

**Create from preset** (`POST /scenarios/from-preset`) clones template JSON into a new non-preset definition (unique `code` / `name` required).

## HTTP API

Base path matches other control-plane routes (no `/api` prefix in this repo). All routes use **`requireCanManageIdentity`** (same as Role Builder).

| Method | Path |
|--------|------|
| GET | `/scenarios/presets` |
| POST | `/scenarios/from-preset` |
| GET | `/scenarios` |
| POST | `/scenarios` |
| GET | `/scenarios/:id` |
| PATCH | `/scenarios/:id` |
| DELETE | `/scenarios/:id` |
| GET | `/scenarios/:id/preview` |
| POST | `/scenarios/:id/role-bindings` |
| POST | `/scenarios/:id/source-bindings` |
| POST | `/scenarios/:id/job-bindings` |
| POST | `/scenarios/:id/ui-bindings` |

## Code map

- `internal/scenario_builder` — types, validation, repositories, services.
- `internal/httpserver/scenario_builder_routes.go` — transport + audit events.
- `internal/db/migrations/000027_scenario_builder.*.sql` — schema + seeds.

## Admin UI

- `apps/web/src/app/(dash)/admin/scenarios` — list, presets, create-from-preset scaffold.
- `apps/web/src/app/(dash)/admin/scenarios/[id]` — detail + preview JSON scaffold.

## Validation (high level)

- Missing `name` / `code` / `scenario_type` / `output_mode` rejected.
- `digest` / `process` / `governance` / `explorer` require explicit `input_scope_json` (non-empty with recognized keys); `ask` requires inherit-retrieval flag or explicit scope.
- `unrestricted: true` in input scope requires `config_json.allow_unrestricted_input_scope`.
- `scheduled` / `event_driven` / `conditional` triggers require matching `trigger_config_json` keys.
- Types that require output policy (`ask`, `digest`, `process`, `governance`) must have **`output_policy`** on create.
- Bindings validate FKs to `roles`, `source_feeds`, `knowledge_jobs`.
