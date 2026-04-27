# Preset catalog

## Purpose

The preset catalog is a **governed index** over existing builder presets (roles, scenarios, jobs). It does not replace native tables (`roles` with `is_preset`, `scenario_presets`, `job_builder_presets`); it adds:

- stable UUID primary keys for cross-cutting APIs
- categories (axes: function, usage_mode, maturity, object_type)
- directed relationships between catalog entries (e.g. role recommends scenario/job)
- instantiation audit (`preset_instantiation_logs`)
- optional `source_preset_code` on instantiated rows for provenance

Updating catalog metadata or template rows does **not** mutate already-instantiated builder objects.

## Data model (migrations `000029`)

| Table | Role |
|-------|------|
| `preset_catalog_entries` | `preset_type` (`role` \| `scenario` \| `job`), `code` (matches `preset_key` / role preset code), `name`, `description`, `active`, `metadata_json` |
| `preset_categories` | Axis + `code` + `label` |
| `preset_catalog_category_assignments` | Many-to-many entry ↔ category |
| `preset_relationships` | `from_preset_id`, `to_preset_id`, `relationship_type`, optional `metadata_json` |
| `preset_instantiation_logs` | Who instantiated what target (`role` / `scenario` / `job`), payload snapshot |

## HTTP API (`/api/presets`)

Requires the same **identity / publish-style** gate as other control-plane routes (`requireCanManageIdentity`).

| Method | Path | Description |
|--------|------|---------------|
| GET | `/api/presets` | Query: `type`, `category_axis`, `category_code` |
| GET | `/api/presets/:id` | Entry + categories + structured **preview** |
| GET | `/api/presets/:id/related` | Outgoing `preset_relationships` as `RelatedEntry` list |
| POST | `/api/presets/:id/instantiate` | Body: `name`, `code`, `description`, `purpose`, `overrides` — delegates to Role / Scenario / Job builders; writes audit + `preset_instantiation_logs` |

Legacy unprefixed routes (`/roles/presets`, `/scenarios/presets`, `/job-builder/presets`) remain for compatibility.

## Instantiation behavior

- **Role**: `RoleBuilder` preset clone; optional `source_preset_code` on new role.
- **Scenario**: `ScenarioBuilder` from preset; sets `cloned_from_scenario_id` and `source_preset_code`; clears `preset_key` on the new definition.
- **Job**: `CreateFromBuilderPreset` — merges `job_builder_presets.defaults_json` with templates; sets `source_preset_code`.

## Further reading

- [API_SURFACE_V1.md](./API_SURFACE_V1.md) — endpoint index
- [DOMAIN_MODEL.md](./DOMAIN_MODEL.md) — table index
