# Role Builder (control plane)

## Purpose

Role Builder is the **product layer** for reusable operational roles. It sits on top of the centralized permission pipeline (`identity_access.AccessEvaluator`) and the existing tables `roles`, `role_action_permissions`, `action_permissions`, and `user_role_bindings`.

It does **not** replace low-level permission resolution (domain grants, access levels, policies, entity ACL, sensitivity). It **adds**:

- Rich **role definitions** (metadata, categories, lifecycle).
- **Visibility and scope bindings** (domains, entity types, sources, scenarios, dashboards).
- **Governance capabilities** (review / approve / publish / override / manage assignments) as structured flags.
- **Job-level permissions** (per `knowledge_jobs` row) for future enforcement.
- **Presets** (seeded templates) and **clone** workflows.
- **Effective-access preview** for admins.

## Conceptual model

| Concept | Meaning |
|--------|---------|
| **User** | Concrete principal (`users` row). |
| **Role definition** | Reusable profile stored in **`roles`** plus binding tables and `role_action_permissions`. |
| **Role assignment** | **`user_role_bindings`**: user + role + scope (`global` or `domain` today). |
| **Effective access** | Outcome of `Evaluate` + grants: domain access level, role actions, role domain allowlist, role entity-type union, policies, ACL, sensitivity. |

Role definitions are **not** edited as ad hoc per-user permission dumps; users receive **assignments** to shared definitions.

## Role categories

`roles.category` is a free-form string; seeded values include:

- `platform`, `domain`, `operational`, `restricted_specialist`, `legacy` (built-in seed roles).

## Builder capabilities

- **Create** custom role (`POST /roles`) with full binding payload.
- **Create from preset** (`POST /roles/from-preset` with `preset_key`).
- **Clone** (`POST /roles/:id/clone`).
- **Assign** (`POST /roles/:id/assignments`) with duplicate protection (unique index on user + role + scope).
- **Preview** (`GET /roles/:id/preview`) returns structured effective shape for the **definition** (not yet merged with a user’s grants).

## Data model (tables)

| Table | Role |
|-------|------|
| `roles` | Extended with `code`, `category`, `active`, `scope_model`, `is_preset`, `preset_key`, `cloned_from_role_id`, `is_system`. |
| `role_action_permissions` | Action grants (canonical “role_permissions”). |
| `role_domain_bindings` | If **non-empty**, role applies only to listed domains (intersected with assignment scope and `domain_grants`). **Empty** = no extra domain restriction. |
| `role_entity_type_bindings` | If any assigned role that grants the action has rows here, entity `type` must appear in the **union** of those rows (for `entity` resources). **No rows** among applicable roles = no extra restriction. |
| `role_source_scope_bindings` | `scope_kind` + `scope_ref` (e.g. feed id, category key). Preview + future list enforcement. |
| `role_scenario_bindings` | `scenario_key` aligned with `scenario_definitions.code`; rows are **upserted from Scenario Builder** when a scenario’s `scenario_role_bindings.can_see` is true (full replace for that scenario’s mirror uses the scenario’s current code). |
| `role_dashboard_bindings` | Dashboard / view keys. |
| `role_job_permissions` | Per-job flags; preview + optional job-route enforcement later. |
| `role_governance_permissions` | Boolean governance flags. |
| `user_role_bindings` | Assignments; unique `(user_id, role_id, scope_type, coalesce(scope_id))`. |

## Preview JSON (`GET /roles/:id/preview`)

Stable fields:

- `allowed_domains`, `allowed_entity_types`, `allowed_source_scopes`, `allowed_scenarios`, `allowed_dashboards`
- `allowed_actions` (from `action_permissions.code`)
- `governance` (booleans)
- `job_permissions`

## Integration points

- **Permission pipeline**: `userHasRoleAction` requires `roles.active` and respects `role_domain_bindings`. **Step 6b** applies `role_entity_type_bindings` union for entity resources.
- **Assignments**: privileged roles (governance `can_override_policies` or actions `manage_permissions` / `manage_policies`) require the assigner to pass `manage_permissions` on the target domain (for domain-scoped assignment) or on at least one granted domain (for global assignment).
- **Scenarios / sources / jobs**: bindings are stored and exposed in preview; additional HTTP enforcement can be added incrementally without changing the core evaluator.

## Presets (seeded)

| `preset_key` | Intent |
|----------------|--------|
| `platform_admin` | All actions; full governance. |
| `domain_owner` | Domain operations; strong governance without global override. |
| `team_lead` | Lead workflow; review + manage assignments. |
| `reviewer` | Read + review. |
| `executive_viewer` | Read-only. |
| `support_lead` | Ops on sources/jobs + review. |
| `finance_restricted` | Narrow read/export/review; bind domains when cloning. |
| `legal_restricted` | Legal review posture; bind domains when cloning. |

## HTTP (identity-admin gate)

Same gates as identity admin (`requireCanManageIdentity`; domain-scoped assignment also `requirePublishOnDomain`).

| Method | Path |
|--------|------|
| GET | `/roles/presets` |
| GET | `/roles` |
| POST | `/roles` |
| POST | `/roles/from-preset` |
| GET | `/roles/:id` |
| PATCH | `/roles/:id` (`bindings` object replaces all bindings when present) |
| DELETE | `/roles/:id` |
| POST | `/roles/:id/clone` |
| GET | `/roles/:id/assignments` |
| POST | `/roles/:id/assignments` |
| GET | `/roles/:id/preview` |

## Code map

- `internal/role_builder` — repositories and services.
- `internal/httpserver/role_builder_routes.go` — transport.
- `internal/identity_access/access.go` — evaluator integration.

## Scenario Builder integration

Scenario definitions live in **`scenario_definitions`** (see [scenario-builder.md](./scenario-builder.md)). Role Builder still exposes `allowed_scenarios` on role read/write; **Scenario Builder** is the preferred place to attach `can_see` / `can_run` / `can_manage` / `can_review_publish` per scenario and role.

**Next enforcement steps (shared with Scenario Builder):**

1. Enforce `role_scenario_bindings` in Ask / retrieval entrypoints.
2. `PreviewUser` merging assignments + grants for `/auth/me`.
3. Enforce `role_source_scope_bindings` on source feed list/detail.
4. Enforce `role_job_permissions` on job run/patch routes.
