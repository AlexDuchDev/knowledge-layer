
# INITIAL_SCHEMA_OUTLINE.md

## 1. Purpose

This document outlines the initial database schema for v1.

It is not a final migration spec.
It is the first structural map of the main tables and their purpose.

The goals are:
- make core entities explicit
- reflect the domain model
- support access, provenance, workflow, and jobs
- avoid premature over-modeling
- give implementation a clear starting point

---

## 2. Primary storage assumptions

Primary transactional database:
- PostgreSQL

Additional stores:
- S3-compatible storage for raw artifacts and large blobs
- Redis for queues/cache/coordination
- OpenSearch for search projections
- pgvector for embeddings where applicable

This document focuses on PostgreSQL tables.

---

## 3. Identity and access tables

## 3.1 users

Purpose:
human principals

Suggested columns:
- `id`
- `email`
- `name`
- `status`
- `primary_team_id`
- `created_at`
- `updated_at`

---

## 3.2 teams

Purpose:
organizational grouping

Suggested columns:
- `id`
- `name`
- `description`
- `owner_id`
- `status`
- `created_at`
- `updated_at`

---

## 3.3 user_team_memberships

Purpose:
many-to-many team membership

Suggested columns:
- `id`
- `user_id`
- `team_id`
- `membership_type`
- `created_at`

---

## 3.4 roles

Purpose:
reusable permission bundles

Suggested columns:
- `id`
- `name`
- `description`
- `created_at`
- `updated_at`

---

## 3.5 user_role_bindings

Purpose:
role assignment with scope

Suggested columns:
- `id`
- `user_id`
- `role_id`
- `scope_type`
- `scope_id`
- `granted_by`
- `granted_at`
- `expires_at`

---

## 3.6 domains

Purpose:
business knowledge boundary

Suggested columns:
- `id`
- `name`
- `description`
- `owner_id`
- `default_access_policy_id`
- `default_sensitivity_level`
- `status`
- `created_at`
- `updated_at`

---

## 3.7 domain_grants

Purpose:
user access to domains

Suggested columns:
- `id`
- `user_id`
- `domain_id`
- `access_level`
- `sensitivity_cap`
- `granted_by`
- `granted_at`
- `expires_at`

---

## 3.8 action_permissions

Purpose:
explicit action permission definitions

Suggested columns:
- `id`
- `code`
- `description`
- `created_at`

Examples:
- `view`
- `search`
- `retrieve`
- `edit`
- `review`
- `approve`
- `publish`
- `run_job`
- `manage_source_feed`

---

## 3.9 role_action_permissions

Purpose:
map roles to actions

Suggested columns:
- `id`
- `role_id`
- `action_permission_id`

---

## 3.10 access_policies

Purpose:
reusable policy objects

Suggested columns:
- `id`
- `name`
- `description`
- `domain_id`
- `entity_type_scope`
- `inheritance_mode`
- `status`
- `created_at`
- `updated_at`

Note:
detailed rules may live in JSON initially, then normalize later if needed.

---

## 3.11 policy_overrides

Purpose:
exception handling

Suggested columns:
- `id`
- `target_type`
- `target_id`
- `override_type`
- `policy_payload`
- `reason`
- `created_by`
- `created_at`
- `expires_at`

---

## 4. Ingestion and source tables

## 4.1 connectors

Purpose:
connector type or configured connector metadata

Suggested columns:
- `id`
- `type`
- `display_name`
- `auth_mode`
- `capabilities_json`
- `status`
- `created_at`
- `updated_at`

---

## 4.2 source_feeds

Purpose:
governed source input boundary

Suggested columns:
- `id`
- `connector_id`
- `source_uri`
- `display_name`
- `owner_id`
- `domain_id`
- `sensitivity_level`
- `allowed_job_types_json`
- `ingestion_mode`
- `sync_mode`
- `access_policy_id`
- `status`
- `health_status`
- `last_sync_at`
- `last_successful_sync_at`
- `created_at`
- `updated_at`

---

## 4.3 ingestion_runs

Purpose:
one ingestion execution record

Suggested columns:
- `id`
- `source_feed_id`
- `trigger_type`
- `status`
- `started_at`
- `completed_at`
- `records_ingested_count`
- `records_deduplicated_count`
- `warning_count`
- `error_count`
- `trace_ref`

---

## 4.4 raw_artifacts

Purpose:
durable raw source evidence

Suggested columns:
- `id`
- `source_feed_id`
- `ingestion_run_id`
- `artifact_type`
- `external_artifact_id`
- `storage_uri`
- `content_hash`
- `source_created_at`
- `source_author_ref`
- `metadata_json`
- `created_at`

---

## 4.5 normalized_records

Purpose:
stable normalized ingestion outputs

Suggested columns:
- `id`
- `raw_artifact_id`
- `source_feed_id`
- `record_type`
- `structured_payload_json`
- `record_hash`
- `source_timestamp`
- `detected_author_ref`
- `normalization_version`
- `created_at`

---

## 5. Knowledge core tables

## 5.1 entities

Purpose:
shared governed entity record

Suggested columns:
- `id`
- `type`
- `title`
- `summary`
- `body`
- `owner_id`
- `domain_id`
- `sensitivity_level`
- `truth_mode`
- `canonical_status`
- `approval_status`
- `freshness_status`
- `lifecycle_state`
- `access_policy_id`
- `policy_source_type`
- `policy_source_id`
- `external_ref`
- `review_due_at`
- `approved_at`
- `approved_by_id`
- `superseded_by_id`
- `created_by_type`
- `created_by_id`
- `created_at`
- `updated_at`
- `archived_at`

Note:
type-specific details can live in extension tables or JSON initially.

---

## 5.2 entity_payloads

Purpose:
type-specific structured data

Suggested columns:
- `id`
- `entity_id`
- `entity_type`
- `payload_json`
- `schema_version`
- `created_at`
- `updated_at`

Use for:
- decision rationale
- meeting participants
- project-specific fields
- policy-specific metadata

---

## 5.3 entity_versions

Purpose:
version history

Suggested columns:
- `id`
- `entity_id`
- `entity_type`
- `version_number`
- `snapshot_json`
- `change_summary`
- `changed_by_type`
- `changed_by_id`
- `created_at`

---

## 5.4 entity_links

Purpose:
typed relationships between entities

Suggested columns:
- `id`
- `from_entity_type`
- `from_entity_id`
- `relation_type`
- `to_entity_type`
- `to_entity_id`
- `created_by_type`
- `created_by_id`
- `confidence_score`
- `created_at`

---

## 5.5 provenance_records

Purpose:
source and creation lineage

Suggested columns:
- `id`
- `target_type`
- `target_id`
- `origin_type`
- `origin_ref`
- `source_feed_id`
- `job_run_id`
- `created_by_type`
- `created_by_id`
- `created_at`

---

## 5.6 provenance_raw_artifacts

Purpose:
many-to-many join for provenance -> raw artifacts

Suggested columns:
- `id`
- `provenance_record_id`
- `raw_artifact_id`

---

## 5.7 provenance_normalized_records

Purpose:
many-to-many join for provenance -> normalized records

Suggested columns:
- `id`
- `provenance_record_id`
- `normalized_record_id`

---

## 6. Workflow and governance tables

## 6.1 review_tasks

Purpose:
human review queue item

Suggested columns:
- `id`
- `target_type`
- `target_id`
- `reviewer_id`
- `owner_id`
- `status`
- `due_at`
- `resolution_note`
- `created_at`
- `completed_at`

---

## 6.2 approval_flows

Purpose:
approval rules by entity type and domain

Suggested columns:
- `id`
- `entity_type`
- `domain_id`
- `required_approver_roles_json`
- `approval_mode`
- `status`
- `created_at`
- `updated_at`

---

## 6.3 approval_records

Purpose:
approval decisions

Suggested columns:
- `id`
- `target_type`
- `target_id`
- `approver_id`
- `status`
- `comment`
- `created_at`

---

## 6.4 freshness_rules

Purpose:
stale/review timing rules

Suggested columns:
- `id`
- `entity_type`
- `domain_id`
- `review_interval_days`
- `stale_after_days`
- `severity`
- `created_at`
- `updated_at`

---

## 7. Knowledge jobs tables

## 7.1 knowledge_jobs

Purpose:
governed job definition

Suggested columns:
- `id`
- `name`
- `job_type`
- `purpose`
- `description`
- `owner_id`
- `operator_scope_json`
- `source_scope_json`
- `trigger_type`
- `output_type`
- `output_domain_id`
- `output_sensitivity_level`
- `publication_mode`
- `review_required`
- `approval_required`
- `sanitization_rules_json`
- `config_json`
- `status`
- `created_at`
- `updated_at`

---

## 7.2 job_triggers

Purpose:
trigger configuration

Suggested columns:
- `id`
- `knowledge_job_id`
- `trigger_type`
- `schedule_expr`
- `event_filter_json`
- `window_config_json`
- `status`
- `created_at`
- `updated_at`

---

## 7.3 job_runs

Purpose:
one execution instance

Suggested columns:
- `id`
- `knowledge_job_id`
- `initiated_by_type`
- `initiated_by_id`
- `trigger_type`
- `status`
- `input_scope_snapshot_json`
- `started_at`
- `completed_at`
- `warning_count`
- `error_count`
- `trace_ref`
- `execution_metrics_json`

---

## 7.4 job_outputs

Purpose:
outputs produced by job runs

Suggested columns:
- `id`
- `job_run_id`
- `output_type`
- `target_entity_id`
- `target_entity_type`
- `review_task_id`
- `publication_status`
- `created_at`

---

## 8. Retrieval and AI support tables

## 8.1 chunks

Purpose:
retrievable content segments

Suggested columns:
- `id`
- `parent_type`
- `parent_id`
- `chunk_index`
- `text`
- `token_count`
- `embedding_ref`
- `search_document_ref`
- `created_at`
- `updated_at`

---

## 8.2 embedding_records

Purpose:
vector metadata

Suggested columns:
- `id`
- `parent_type`
- `parent_id`
- `chunk_id`
- `model_name`
- `vector_ref`
- `created_at`

---

## 8.3 search_documents

Purpose:
search projection metadata

Suggested columns:
- `id`
- `parent_type`
- `parent_id`
- `index_name`
- `projection_version`
- `indexed_at`

---

## 8.4 answer_traces

Purpose:
AI answer traceability

Suggested columns:
- `id`
- `request_type`
- `requester_id`
- `scope_snapshot_json`
- `retrieved_refs_json`
- `model_name`
- `prompt_ref`
- `output_ref`
- `citation_refs_json`
- `created_at`

---

## 9. Platform operations tables

## 9.1 audit_events

Purpose:
sensitive action log

Suggested columns:
- `id`
- `event_type`
- `actor_type`
- `actor_id`
- `target_type`
- `target_id`
- `decision`
- `reason`
- `policy_refs_json`
- `trace_ref`
- `created_at`

---

## 9.2 notifications

Purpose:
system and workflow notifications

Suggested columns:
- `id`
- `recipient_type`
- `recipient_id`
- `notification_type`
- `payload_json`
- `status`
- `created_at`
- `sent_at`

---

## 10. Suggested first schema cut

Build this first, not everything at once:

### Must-have early tables
- users
- teams
- roles
- user_role_bindings
- domains
- domain_grants
- access_policies
- policy_overrides
- source_feeds
- ingestion_runs
- raw_artifacts
- normalized_records
- entities
- entity_versions
- entity_links
- provenance_records
- review_tasks
- knowledge_jobs
- job_runs
- job_outputs
- audit_events

### Can come slightly later
- approval_flows
- approval_records
- freshness_rules
- chunks
- embedding_records
- search_documents
- answer_traces
- notifications

---

## 11. Modeling guidance

### 11.1 Start pragmatic
Use:
- one shared `entities` table
- typed payload table or JSON payloads
- explicit joins for links and provenance
- explicit workflow tables

This is better for v1 than over-splitting every entity type into many separate schemas too early.

### 11.2 Do not under-model governance
Do not skip:
- policy source
- truth mode
- provenance
- workflow state
- ownership

These are not optional metadata.

### 11.3 Keep derived data derived
Chunks, embeddings, search docs, and traces are important, but they are not primary truth.

---

## 12. Final schema stance

The schema should make the platform governable.

A developer or operator should be able to inspect the database model and answer:
- who owns this object
- what domain it belongs to
- how sensitive it is
- where it came from
- what created it
- what reviewed it
- what it is linked to
- what policy governs it

If the schema cannot support those questions cleanly, it is not strong enough.