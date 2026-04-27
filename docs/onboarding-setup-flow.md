# Onboarding / setup flow

## Purpose

Session-based **setup wizard** for a deployment: pick a template (seeded `onboarding_templates`), adjust selections, **preview** a launch plan (validation issues are explicit), then **launch** to instantiate catalog presets via the same path as `POST /api/presets/:id/instantiate`.

Feed creation from drafts is **v1 placeholder** (draft tables exist; launch focuses on preset instantiation).

## Data model (migration `000030`)

| Table | Role |
|-------|------|
| `onboarding_templates` | Seeded modes (`minimal`, `startup_product`, …) with `metadata_json` (`role_codes`, `scenario_codes`, `job_codes`, `connector_families`) |
| `onboarding_sessions` | `status` (`draft` \| `ready` \| `launched` \| `abandoned`), `template_code`, `created_by_user_id`, `org_profile_json` |
| `onboarding_session_steps` | Optional normalized step payloads (`step_key`, `payload_json`) |
| `onboarding_selected_presets` | `(session_id, preset_catalog_entry_id, slot)` + `customizations_json` |
| `onboarding_connector_selections` | Connector family toggles |
| `onboarding_source_feed_drafts` | JSON drafts for future feed creation |
| `onboarding_assignment_drafts` | `initial_admin_user_id`, `domain_owner_user_id`, `assignments_json` |
| `onboarding_launch_logs` | Per-launch `result_json`, `status`, `error_text` |

**Ownership**: sessions are scoped to `created_by_user_id`; only that principal can read, patch, preview, or launch (403 otherwise).

## HTTP API (`/api/onboarding`)

Same control-plane gate as preset catalog (`requireCanManageIdentity`).

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/onboarding/templates` | List seeded templates |
| POST | `/api/onboarding/sessions` | Create draft session |
| GET | `/api/onboarding/sessions` | List sessions for current user (`?limit=`, max 100) |
| GET | `/api/onboarding/sessions/:id` | Full `SessionView` |
| PATCH | `/api/onboarding/sessions/:id` | Org profile, steps, `selected_presets` (full replace), connectors, assignment |
| POST | `/api/onboarding/sessions/:id/select-template` | Apply template defaults to selections + connectors |
| POST | `/api/onboarding/sessions/:id/preview` | `LaunchPreview` + `validation_issues` |
| POST | `/api/onboarding/sessions/:id/launch` | Instantiate presets; 409 if already `launched`; 400 if validation fails |

## Validation (preview / launch)

Launch is blocked when preview would report issues, including:

- no template selected
- no `initial_admin_user_id` in assignment draft
- no selected presets

Job presets that require a configured source scope (e.g. `weekly_digest`) may fail at launch if defaults are incomplete; operators can trim `selected_presets` before launch or complete feeds first.

## Admin UI

- `/admin/setup` — templates + session list + link to wizard
- `/admin/setup/[sessionId]` — template re-select, PATCH-driven steps, preview, launch

## Related

- [preset-catalog.md](./preset-catalog.md)
- [API_SURFACE_V1.md](./API_SURFACE_V1.md)
