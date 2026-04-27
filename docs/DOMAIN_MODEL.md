# Domain model (v1)

This file is the **operational index** (implemented tables, HTTP routes, UI field notes). The **full structural contract** (taxonomy, lifecycle, truth modes, long narrative) is **[DOMAIN_MODEL_CONTRACT.md](./DOMAIN_MODEL_CONTRACT.md)**. For access gates see [ACCESS_MODEL.md](./ACCESS_MODEL.md).

## 1. Operational index (implemented shapes)

### Identity (control plane)

| Concept | Table(s) | HTTP |
|---------|----------|------|
| User | `users` | `GET/POST/PATCH /users` |
| Domain | `domains` | `GET /domains`, `POST /domains`, `PATCH /domains/:id` |
| Team | `teams` | `GET/POST/PATCH /teams` |
| Team membership | `user_team_memberships` | `GET/POST/DELETE /user-team-memberships` |
| Domain grant | `domain_grants` | `GET/POST/PATCH/DELETE /domain-grants` |
| Role | `roles` | `GET /roles` |
| Role binding | `user_role_bindings` | `GET/POST/PATCH/DELETE /user-role-bindings` |

See [ACCESS_MODEL.md](./ACCESS_MODEL.md) for gates and audit events.

### Preset catalog & onboarding (control plane)

| Concept | Table(s) | HTTP |
|---------|----------|------|
| Preset catalog entry | `preset_catalog_entries`, `preset_categories`, `preset_catalog_category_assignments`, `preset_relationships`, `preset_instantiation_logs` | `GET/POST /api/presets` (see [preset-catalog.md](./preset-catalog.md)) |
| Setup session | `onboarding_templates`, `onboarding_sessions`, `onboarding_session_steps`, `onboarding_selected_presets`, `onboarding_connector_selections`, `onboarding_source_feed_drafts`, `onboarding_assignment_drafts`, `onboarding_launch_logs` | `GET/POST/PATCH /api/onboarding/...` (see [onboarding-setup-flow.md](./onboarding-setup-flow.md)) |

Provenance: optional `source_preset_code` on `roles`, `scenario_definitions`, `knowledge_jobs` for objects created from catalog or builder presets.

### Knowledge core

Entities, links, projections — see existing migrations and `knowledge_core` package; list/detail routes under `/entities`.

- **`entity_types`**: catalog of allowed `entities.type` values (seeded); supports policy scoping.
- **`entity_acl`**: optional per-entity deny rows for `user` / `team` principals.
- **`chunks` / `embeddings`**: chunk text and optional `vector` embeddings (pgvector) for future hybrid retrieval; linked to entities for provenance.

### Ingestion

Connectors and source feeds — see [INGESTION_AND_CONNECTORS.md](./INGESTION_AND_CONNECTORS.md) (update alongside connector CRUD).

### Editorial & topic surfacing (v1)

| Concept | Table(s) | HTTP (permission-gated) |
|---------|----------|-------------------------|
| Content hub | `content_hubs`, `content_hub_items` | `GET/POST /content-hubs`, `GET /content-hubs/:id/view` |
| Reusable block | `content_blocks`, `entity_content_block_refs` | `POST /content-blocks`, `GET/POST /entities/:id/content-blocks` |
| Editorial hold | `editorial_holdings` | `POST /governance/editorial/hold`, `POST /governance/editorial/feature` |
| Publishing queue | entities + holdings | `GET /governance/publishing-queue` |
| Search telemetry | `search_interaction_log` | written on `GET /search` (fail-open); `GET /ops/search-insights` |

UI surfaces: `/hubs`, `/governance/editorial`, `/ops/*`, workflow links from Home and Governance.

### User surfacing preferences (not access grants)

| Concept | Table | HTTP |
|---------|-------|------|
| Scope follow | `user_scope_follows` | `GET/POST/DELETE /me/follows` |

- `scope_type`: `domain` (`ref_id` = domain id), `content_hub` (`ref_id` = hub id; hub must exist), `knowledge_topic` (`ref_id` = domain id or empty for all granted domains + `entity_type` = canonical `entities.type`), `digest_stream` (`ref_id` = domain id for digest surfacing on Home).

These preferences affect **what appears on Home and notifications**, not whether the user can `view` an entity.

### Raw vs normalized connector fields (UI)

- **Raw artifact** (`GET /raw-artifacts/:id`): `source_feed_id`, `artifact_type`, `external_artifact_id`, `source_author_ref`, `source_created_at`, `storage_uri`, `metadata_json`, `created_at`.
- **Normalized record** (`GET /normalized-records/:id`): `record_type`, `raw_artifact_id`, `source_feed_id`, `detected_author_ref`, `source_timestamp`, `structured_payload_json`, `created_at`.

---

## 2. Structural contract (full specification)

The long-form taxonomy, metadata, lifecycle narrative, and modeling goals live in:

**[DOMAIN_MODEL_CONTRACT.md](./DOMAIN_MODEL_CONTRACT.md)**

Edit the contract there; keep §1 in this file as the **implemented-shape index** for engineers and external contributors. Follow [DOCS_MAINTENANCE_POLICY.md](./DOCS_MAINTENANCE_POLICY.md) when either file changes.
