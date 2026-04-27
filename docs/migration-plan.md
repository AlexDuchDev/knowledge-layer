# Migration plan (logical phases)

Physical files live in [`apps/api/internal/db/migrations/`](../apps/api/internal/db/migrations/). This document maps **logical foundation phases** to those files. **Do not re-apply DDL** for tables that already exist.

## Naming equivalences (spec vs database)

| Spec / doc name | Actual table / mechanism |
|-----------------|---------------------------|
| `user_domain_grants` | `domain_grants` |
| `user_roles` | `user_role_bindings` + `role_action_permissions` / `action_permissions` |
| `sync_jobs` | `ingestion_runs` |

## Phase 1: Identity and access foundation

**Tables:** users, teams, memberships, roles, action permissions, bindings, domains, `access_policies`, `domain_grants`, `policy_overrides`.

**Files:** [`000001_identity_access.up.sql`](../apps/api/internal/db/migrations/000001_identity_access.up.sql)

**Order:** users/teams → roles/permissions → policies → domains (FK to policies) → domain_grants → policy_overrides.

**Additions (later migration):** [`000016_architecture_foundation.up.sql`](../apps/api/internal/db/migrations/000016_architecture_foundation.up.sql) adds `entity_acl` (object-level deny), aligned with centralized evaluation.

## Phase 2: Knowledge core foundation

**Tables:** entities, entity_versions, entity_links (and domain fields on entities).

**Files:** [`000002_knowledge_core.up.sql`](../apps/api/internal/db/migrations/000002_knowledge_core.up.sql)

**Additions:** `entity_types` catalog in `000016` (canonical `entities.type` labels).

## Phase 3: Connectors / ingestion foundation

**Tables:** connectors, source_feeds, raw_artifacts, normalized_records, ingestion_runs.

**Files:** [`000003_ingestion.up.sql`](../apps/api/internal/db/migrations/000003_ingestion.up.sql), connector stubs [`000010_google_drive_connector.up.sql`](../apps/api/internal/db/migrations/000010_google_drive_connector.up.sql), [`000012_slack_connector_stub.up.sql`](../apps/api/internal/db/migrations/000012_slack_connector_stub.up.sql), view/raw permission [`000009_view_raw_permission.up.sql`](../apps/api/internal/db/migrations/000009_view_raw_permission.up.sql).

## Phase 4: Knowledge jobs foundation

**Tables:** knowledge_jobs, job_triggers, job_runs, job_outputs.

**Files:** [`000005_knowledge_jobs.up.sql`](../apps/api/internal/db/migrations/000005_knowledge_jobs.up.sql)

**Additions:** `knowledge_job_sources`, `knowledge_job_operators` in `000016`.

## Phase 5: Governance foundation

**Tables:** review_tasks, policy/exception/feedback extensions, `approval_flows`.

**Files:** [`000006_review_tasks.up.sql`](../apps/api/internal/db/migrations/000006_review_tasks.up.sql), [`000011_policy_exceptions_and_feedback.up.sql`](../apps/api/internal/db/migrations/000011_policy_exceptions_and_feedback.up.sql), `approval_flows` in `000016`.

## Phase 6: Retrieval foundation

**Tables:** answer_traces, chunks, embeddings (pgvector).

**Files:** [`000011_answer_traces.up.sql`](../apps/api/internal/db/migrations/000011_answer_traces.up.sql) (answer_traces), `chunks` / `embeddings` / `vector` extension in `000016`.

## Phase 7: Audit foundation

**Tables:** audit_events (append-only).

**Files:** [`000004_audit_events.up.sql`](../apps/api/internal/db/migrations/000004_audit_events.up.sql)

## Other migrations (cross-cutting)

| File | Purpose |
|------|---------|
| `000007_placeholder` | Placeholder |
| `000008_dev_seed` | Dev seed data |
| `000013_content_hubs_*` | Content hubs / editorial |
| `000014_auth_invitations` | Auth / invitations |
| `000015_user_scope_follows` | Surfacing follows |

## Tech debt

- **Duplicate `000011_` prefix:** two different migration pairs (`000011_answer_traces` vs `000011_policy_exceptions_and_feedback`). golang-migrate runs in lexical order; both apply. Renaming files requires care in deployed environments—prefer documenting until a dedicated maintenance window.

## Rollout notes

- Apply migrations in order on an empty database; CI and local `db` package should use the same sequence.
- Postgres image should support **pgvector** if `000016` is used (e.g. `pgvector/pgvector` image in Docker).
- New migrations should be **additive** (new tables/columns/indexes) unless a coordinated data migration is planned.
