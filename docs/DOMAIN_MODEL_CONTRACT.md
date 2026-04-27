# Domain model — structural contract (reference)

This file holds the **long-form structural specification** (taxonomy, lifecycle, truth modes, narrative) split from [DOMAIN_MODEL.md](./DOMAIN_MODEL.md) so the latter stays a short **operational index**.

**For implemented tables and HTTP routes, start with** [DOMAIN_MODEL.md](./DOMAIN_MODEL.md) §1.

---



## 2. Structural contract (full specification)

The subsections below keep the original specification outline (their heading numbers are independent of §1).


## 1. Purpose

This document defines the core domain model for the Organizational Memory & Knowledge Operations Platform.

It establishes:
- the primary entity types in the system
- required fields and shared metadata
- object relationships
- provenance and versioning rules
- lifecycle and ownership expectations
- truth classification rules
- modeling boundaries for v1

This document is a structural contract.
It should guide product design, backend modeling, retrieval behavior, governance workflows, and AI output shaping.

---

## 2. Modeling goals

The domain model should allow the system to:
- represent important company knowledge as structured objects
- preserve source provenance and decision context
- distinguish between authoritative and derived materials
- support review, approval, freshness, and lifecycle workflows
- enable strong retrieval and relation-aware navigation
- remain simple enough for v1 implementation
- expand gradually without breaking foundational assumptions

---

## 3. Core modeling principles

### 3.1 Entities over files
The platform is built around canonical entities, not document trees or file folders.

### 3.2 Structure over raw storage
Raw source artifacts matter, but the core product value comes from structured knowledge objects and their relationships.

### 3.3 Truth must be explicit
Every significant entity must clearly indicate whether it is:
- canonical in platform
- mirrored from an external source
- derived from source inputs

### 3.4 Provenance is mandatory
Important entities must preserve where they came from, how they were created, and what source evidence supports them.

### 3.5 Ownership is mandatory
Every governed artifact should have a clear owner.

### 3.6 Relationships matter
The platform should model links between entities explicitly, not infer everything at query time.

### 3.7 Start narrow
v1 should keep the model focused and resist ontology sprawl.

---

## 4. Entity taxonomy overview

The domain model is organized into five broad groups:

1. Identity and governance entities
2. Source and ingestion entities
3. Knowledge entities
4. Workflow and job entities
5. Retrieval and derived-data entities

The most important product-facing objects in v1 are the **knowledge entities**.

---

## 5. Shared metadata model for governed entities

Most governed entities should include a common metadata envelope.

### 5.1 Required common fields

All major governed entities should include:

- `id`
- `type`
- `title`
- `owner_id`
- `domain_id`
- `sensitivity_level`
- `truth_mode`
- `canonical_status`
- `approval_status`
- `freshness_status`
- `lifecycle_state`
- `access_policy_id`
- `created_at`
- `updated_at`
- `created_by_type`
- `created_by_id`
- `review_due_at`
- `version_count`

### 5.2 Recommended common fields

Where applicable, entities should also support:

- `summary`
- `body`
- `tags`
- `source_refs`
- `related_entity_ids`
- `last_reviewed_at`
- `approved_at`
- `approved_by_id`
- `archived_at`
- `superseded_by_id`
- `external_ref`
- `confidence_score` for derived artifacts only

### 5.3 Field semantics

#### `truth_mode`
Allowed values:
- `canonical_in_platform`
- `mirrored_authority`
- `derived_artifact`

#### `canonical_status`
Describes whether the object is currently part of the platform’s canonical working set.

Suggested values:
- `canonical`
- `candidate`
- `reference_only`
- `superseded`

#### `approval_status`
Suggested values:
- `not_required`
- `pending_review`
- `approved`
- `rejected`

#### `freshness_status`
Suggested values:
- `fresh`
- `review_due`
- `stale`
- `unknown`

#### `created_by_type`
Suggested values:
- `user`
- `system`
- `job_run`
- `connector_sync`
- `migration`

---

## 6. Identity and governance entities

These entities define who exists in the system, how permissions are assigned, and how governance is enforced.

### 6.1 User

Represents an individual system actor.

Core fields:
- `id`
- `name`
- `email`
- `status`
- `primary_team_id`
- `role_ids`
- `domain_grants`
- `created_at`
- `updated_at`

Notes:
- A user may belong to multiple teams
- A user may have multiple role bindings
- A user may have explicit overrides

---

### 6.2 Team

Represents an organizational grouping.

Core fields:
- `id`
- `name`
- `description`
- `status`
- `owner_id`
- `created_at`
- `updated_at`

---

### 6.3 Role

Represents a reusable access and responsibility pattern.

Core fields:
- `id`
- `name`
- `description`
- `default_permissions`
- `assignable_scope`
- `created_at`
- `updated_at`

Examples:
- Admin
- Governance Operator
- Domain Owner
- Reviewer
- Operator
- Viewer

---

### 6.4 Domain

Represents a business or organizational knowledge boundary.

Core fields:
- `id`
- `name`
- `description`
- `owner_id`
- `default_access_policy_id`
- `default_sensitivity_level`
- `status`
- `created_at`
- `updated_at`

Examples:
- Finance
- Marketing
- Product
- Engineering
- Legal
- Operations

Important rule:
Many access and governance policies should inherit from the domain layer.

---

### 6.5 AccessPolicy

Represents a reusable policy object.

Core fields:
- `id`
- `name`
- `description`
- `domain_id`
- `entity_type_scope`
- `sensitivity_rules`
- `action_permissions`
- `inheritance_mode`
- `status`
- `created_at`
- `updated_at`

---

### 6.6 PolicyOverride

Represents an exception to inherited policy.

Core fields:
- `id`
- `target_type`
- `target_id`
- `override_type`
- `reason`
- `created_by_id`
- `expires_at`
- `created_at`

Important rule:
Overrides are exceptions, not the default operating model.

---

## 7. Source and ingestion entities

These entities model external systems, source feeds, raw inputs, and normalized ingestion state.

### 7.1 Connector

Represents a connector type or configured connector instance.

Core fields:
- `id`
- `type`
- `display_name`
- `auth_mode`
- `status`
- `capabilities`
- `created_at`
- `updated_at`

Examples:
- Telegram
- Slack
- Email
- Fireflies
- Granola
- Jira
- Trello
- Notion
- Google Drive

---

### 7.2 SourceFeed

Represents a governed source input boundary.

Core fields:
- `id`
- `connector_id`
- `source_uri`
- `display_name`
- `owner_id`
- `domain_id`
- `sensitivity_level`
- `allowed_job_types`
- `ingestion_mode`
- `sync_mode`
- `status`
- `last_sync_at`
- `last_successful_sync_at`
- `health_status`
- `access_policy_id`
- `created_at`
- `updated_at`

Examples:
- `telegram://finance_ops_internal`
- `slack://product-daily`
- `email://leadership-updates`
- `fireflies://weekly-exec-review`

Important rule:
A source feed is the main governance control point for source ingestion.

---

### 7.3 RawArtifact

Represents raw source material ingested from an external system.

Core fields:
- `id`
- `source_feed_id`
- `artifact_type`
- `external_artifact_id`
- `storage_uri`
- `content_hash`
- `source_created_at`
- `source_author_ref`
- `metadata_json`
- `ingestion_run_id`
- `created_at`

Examples:
- Telegram message batch
- meeting transcript file
- email thread snapshot
- Notion page snapshot
- Jira issue export
- Google Doc snapshot

Important rule:
Raw artifacts are not knowledge objects.
They are source evidence and reprocessing inputs.

---

### 7.4 NormalizedRecord

Represents a normalized ingestion unit derived from raw source material.

Core fields:
- `id`
- `raw_artifact_id`
- `source_feed_id`
- `record_type`
- `structured_payload`
- `record_hash`
- `source_timestamp`
- `detected_author_ref`
- `normalization_version`
- `created_at`

Purpose:
Acts as a stable ingestion boundary between raw data and downstream processing.

---

### 7.5 IngestionRun

Represents a single ingestion execution.

Core fields:
- `id`
- `source_feed_id`
- `trigger_type`
- `status`
- `started_at`
- `completed_at`
- `records_ingested_count`
- `records_deduplicated_count`
- `error_count`
- `warning_count`
- `trace_ref`

---

## 8. Knowledge entities

These are the main product-facing domain objects.

### 8.1 Entity superclass concept

All knowledge entities conceptually inherit from a common governed entity shape with:
- metadata
- truth classification
- provenance
- lifecycle
- ownership
- policy
- versioning
- relationships

Implementation may use a single table, joined tables, JSON extension fields, or typed sub-models depending on architecture, but product semantics should stay consistent.

---

### 8.2 Decision

Represents a meaningful business, product, operational, or organizational decision.

Why it matters:
Decision is one of the highest-value objects in v1 because it captures rationale, tradeoffs, authority, and consequences.

Core fields:
- `id`
- `type = decision`
- `title`
- `summary`
- `rationale`
- `status`
- `owner_id`
- `domain_id`
- `sensitivity_level`
- `decision_date`
- `effective_from`
- `alternatives_considered`
- `impact_summary`
- `related_project_id`
- `confirmed_by_id`
- `confirmation_method`
- `review_due_at`
- `truth_mode`
- `created_at`
- `updated_at`

Suggested status values:
- `proposed`
- `confirmed`
- `superseded`
- `archived`

Typical relations:
- `decides`
- `affects`
- `derived_from`
- `created_in_meeting`
- `linked_to_project`
- `superseded_by`

Important rule:
A decision extracted by AI should default to a candidate or derived state until confirmed where required.

---

### 8.3 Project

Represents a project-level initiative or body of coordinated work.

Core fields:
- `id`
- `type = project`
- `title`
- `summary`
- `status`
- `owner_id`
- `domain_id`
- `sensitivity_level`
- `external_ref`
- `start_state_ref`
- `end_state_ref`
- `related_initiative_id`
- `review_due_at`
- `truth_mode`
- `created_at`
- `updated_at`

Suggested status values:
- `draft`
- `active`
- `blocked`
- `completed`
- `archived`

Typical relations:
- `related_to`
- `contains`
- `depends_on`
- `affected_by_decision`
- `has_meeting`
- `has_insight`

---

### 8.4 Initiative

Represents a larger thematic effort that may span projects.

Core fields:
- `id`
- `type = initiative`
- `title`
- `summary`
- `status`
- `owner_id`
- `domain_id`
- `sensitivity_level`
- `truth_mode`
- `created_at`
- `updated_at`

---

### 8.5 SOP

Represents a standard operating procedure that is expected to guide recurring work.

Core fields:
- `id`
- `type = sop`
- `title`
- `summary`
- `body`
- `status`
- `owner_id`
- `domain_id`
- `sensitivity_level`
- `review_due_at`
- `effective_from`
- `truth_mode`
- `created_at`
- `updated_at`

Suggested status values:
- `draft`
- `in_review`
- `approved`
- `active`
- `stale`
- `archived`

Important rule:
For many teams, SOP should usually be `canonical_in_platform`.

---

### 8.6 Process

Represents a repeatable business process or workflow.

Core fields:
- `id`
- `type = process`
- `title`
- `summary`
- `body`
- `status`
- `owner_id`
- `domain_id`
- `sensitivity_level`
- `review_due_at`
- `truth_mode`
- `created_at`
- `updated_at`

Difference from SOP:
- Process describes the broader flow or operational sequence
- SOP usually describes a more instruction-oriented operating procedure

---

### 8.7 Policy

Represents a rule, standard, or governing statement.

Core fields:
- `id`
- `type = policy`
- `title`
- `summary`
- `body`
- `status`
- `owner_id`
- `domain_id`
- `sensitivity_level`
- `effective_from`
- `review_due_at`
- `truth_mode`
- `created_at`
- `updated_at`

Suggested status values:
- `draft`
- `in_review`
- `approved`
- `active`
- `stale`
- `archived`

Important rule:
Policies should have strong review and approval semantics.

---

### 8.8 Meeting

Represents a structured meeting record or interpreted meeting artifact.

Core fields:
- `id`
- `type = meeting`
- `title`
- `summary`
- `meeting_date`
- `participants`
- `owner_id`
- `domain_id`
- `sensitivity_level`
- `raw_transcript_refs`
- `meeting_type`
- `truth_mode`
- `created_at`
- `updated_at`

Typical relations:
- `contains_decision`
- `contains_action_item`
- `linked_to_project`
- `produced_insight`

Important rule:
A meeting object may be derived from transcripts and notes but still serve as a key context hub.

---

### 8.9 Incident

Represents an operational failure, issue, or exception event.

Core fields:
- `id`
- `type = incident`
- `title`
- `summary`
- `status`
- `severity`
- `owner_id`
- `domain_id`
- `sensitivity_level`
- `occurred_at`
- `resolved_at`
- `truth_mode`
- `created_at`
- `updated_at`

Typical relations:
- `learned_from_incident`
- `caused_policy_change`
- `triggered_decision`
- `related_to_process`

---

### 8.10 Experiment

Represents a structured test, hypothesis, or learning effort.

Core fields:
- `id`
- `type = experiment`
- `title`
- `summary`
- `hypothesis`
- `status`
- `owner_id`
- `domain_id`
- `sensitivity_level`
- `result_summary`
- `truth_mode`
- `created_at`
- `updated_at`

---

### 8.11 Insight

Represents a generalized learning or observation.

Core fields:
- `id`
- `type = insight`
- `title`
- `summary`
- `body`
- `status`
- `owner_id`
- `domain_id`
- `sensitivity_level`
- `confidence_score`
- `truth_mode`
- `created_at`
- `updated_at`

Suggested status values:
- `candidate`
- `reviewed`
- `approved`
- `archived`

---

### 8.12 CustomerInsight

Represents a customer-specific or customer-derived learning.

Core fields:
- `id`
- `type = customer_insight`
- `title`
- `summary`
- `body`
- `customer_segment`
- `evidence_strength`
- `owner_id`
- `domain_id`
- `sensitivity_level`
- `truth_mode`
- `created_at`
- `updated_at`

---

### 8.13 RoleHandbook

Represents guidance for a specific role.

Core fields:
- `id`
- `type = role_handbook`
- `title`
- `body`
- `role_name`
- `owner_id`
- `domain_id`
- `sensitivity_level`
- `review_due_at`
- `truth_mode`
- `created_at`
- `updated_at`

---

### 8.14 TeamHandbook

Represents shared guidance for a team.

Core fields:
- `id`
- `type = team_handbook`
- `title`
- `body`
- `team_id`
- `owner_id`
- `domain_id`
- `sensitivity_level`
- `review_due_at`
- `truth_mode`
- `created_at`
- `updated_at`

---

### 8.15 Template

Represents a reusable knowledge template.

Core fields:
- `id`
- `type = template`
- `title`
- `body`
- `template_kind`
- `owner_id`
- `domain_id`
- `sensitivity_level`
- `truth_mode`
- `created_at`
- `updated_at`

---

### 8.16 ReferenceDocument

Represents a document-like reference object useful for retrieval and context.

Core fields:
- `id`
- `type = reference_document`
- `title`
- `summary`
- `body`
- `external_ref`
- `owner_id`
- `domain_id`
- `sensitivity_level`
- `truth_mode`
- `created_at`
- `updated_at`

Important note:
This should not become a dumping ground for every unknown thing.
Use only where other entity types do not fit.

---

## 9. Workflow and governance entities

These entities govern review, approvals, freshness, lifecycle, and job execution.

### 9.1 ReviewTask

Represents a request for a human to review a governed object or job output.

Core fields:
- `id`
- `target_type`
- `target_id`
- `reviewer_id`
- `owner_id`
- `status`
- `due_at`
- `created_at`
- `completed_at`
- `resolution_note`

Suggested status values:
- `open`
- `in_progress`
- `approved`
- `changes_requested`
- `rejected`
- `expired`

---

### 9.2 ApprovalFlow

Represents an approval rule or workflow definition.

Core fields:
- `id`
- `entity_type`
- `domain_id`
- `required_approver_roles`
- `approval_mode`
- `status`
- `created_at`
- `updated_at`

---

### 9.3 ApprovalRecord

Represents an approval action instance.

Core fields:
- `id`
- `target_type`
- `target_id`
- `approver_id`
- `status`
- `comment`
- `created_at`

---

### 9.4 FreshnessRule

Represents freshness and review expectations for a class of objects.

Core fields:
- `id`
- `entity_type`
- `domain_id`
- `review_interval_days`
- `stale_after_days`
- `severity`
- `created_at`
- `updated_at`

---

## 10. Knowledge operations entities

These entities model knowledge jobs, their definitions, executions, and outputs.

### 10.1 KnowledgeJob

Represents a reusable, governable operation over knowledge.

Core fields:
- `id`
- `name`
- `job_type`
- `purpose`
- `owner_id`
- `operator_scope`
- `source_scope`
- `trigger_type`
- `output_domain_id`
- `output_sensitivity_level`
- `publication_mode`
- `review_required`
- `sanitization_rules`
- `status`
- `created_at`
- `updated_at`

Suggested job types:
- `summarization`
- `extraction`
- `consolidation`
- `monitoring`
- `transformation`
- `publishing`

Suggested trigger types:
- `manual`
- `scheduled`
- `event_driven`
- `window_based`
- `conditional`

---

### 10.2 JobTrigger

Represents stored trigger configuration for a job.

Core fields:
- `id`
- `knowledge_job_id`
- `trigger_type`
- `schedule_expr`
- `event_filter`
- `window_config`
- `status`
- `created_at`
- `updated_at`

---

### 10.3 JobRun

Represents one execution of a knowledge job.

Core fields:
- `id`
- `knowledge_job_id`
- `initiated_by_type`
- `initiated_by_id`
- `status`
- `input_scope_snapshot`
- `started_at`
- `completed_at`
- `warning_count`
- `error_count`
- `trace_ref`

Suggested status values:
- `queued`
- `running`
- `completed`
- `failed`
- `cancelled`
- `partial_success`

---

### 10.4 JobOutput

Represents output artifacts or routed results from a job run.

Core fields:
- `id`
- `job_run_id`
- `output_type`
- `target_entity_id`
- `target_entity_type`
- `review_task_id`
- `publication_status`
- `created_at`

Suggested publication status values:
- `draft`
- `in_review`
- `published`
- `rejected`

---

## 11. Retrieval and derived-data entities

These entities support search, semantic retrieval, and AI context assembly.

### 11.1 Chunk

Represents a retrievable segment of entity content or source-derived content.

Core fields:
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

Important rule:
Chunks are derived retrieval structures, not truth-bearing entities.

---

### 11.2 EmbeddingRecord

Represents vectorized representation for retrieval.

Core fields:
- `id`
- `parent_type`
- `parent_id`
- `chunk_id`
- `model_name`
- `vector_ref`
- `created_at`

---

### 11.3 SearchDocument

Represents indexed document projection for keyword or hybrid retrieval.

Core fields:
- `id`
- `parent_type`
- `parent_id`
- `index_name`
- `projection_version`
- `indexed_at`

---

### 11.4 AnswerTrace

Represents traceability for AI-generated answers.

Core fields:
- `id`
- `request_type`
- `requester_id`
- `scope_snapshot`
- `retrieved_refs`
- `model_name`
- `prompt_ref`
- `output_ref`
- `citation_refs`
- `created_at`

Important rule:
Answer traces are essential for trust and debugging.

---

## 12. Relationship model

The platform should support explicit typed links between entities.

### 12.1 EntityLink

Core fields:
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

### 12.2 Recommended relation types

Suggested initial relation types:
- `related_to`
- `derived_from`
- `decides`
- `affects`
- `created_in_meeting`
- `owned_by`
- `linked_to_project`
- `learned_from_incident`
- `depends_on`
- `supersedes`
- `documents_process`
- `implements_policy`
- `uses_template`
- `references`

Important rule:
Keep relation types constrained in v1.
Do not introduce dozens of near-duplicate link semantics.

---

## 13. Provenance model

Provenance is required for all important governed entities.

### 13.1 ProvenanceRecord

Core fields:
- `id`
- `target_type`
- `target_id`
- `origin_type`
- `origin_ref`
- `source_feed_id`
- `raw_artifact_ids`
- `normalized_record_ids`
- `job_run_id`
- `created_by_type`
- `created_by_id`
- `created_at`

### 13.2 Provenance origin types

Suggested values:
- `manual_creation`
- `connector_ingestion`
- `job_generation`
- `system_migration`
- `external_sync`

### 13.3 Provenance requirements by object class

#### Canonical entities
Must show:
- who created or confirmed them
- what sources support them
- version history
- review/approval path if applicable

#### Mirrored entities
Must show:
- external source reference
- sync source
- sync time
- mapping history where relevant

#### Derived artifacts
Must show:
- source inputs
- generating job or prompt type
- reviewer or approver when present
- confidence context where relevant

---

## 14. Versioning model

Important knowledge entities must support version history.

### 14.1 EntityVersion

Core fields:
- `id`
- `entity_type`
- `entity_id`
- `version_number`
- `snapshot_ref`
- `change_summary`
- `changed_by_type`
- `changed_by_id`
- `created_at`

### 14.2 Versioning rules

Versioning should apply at minimum to:
- Decision
- SOP
- Process
- Policy
- Meeting
- Insight
- TeamHandbook
- RoleHandbook
- ReferenceDocument

### 14.3 Versioning semantics

- Version history should preserve meaningful snapshots
- Minor metadata-only changes may be handled differently from substantive content changes
- Approval and lifecycle transitions should remain visible in history
- Retrieval should generally default to the current active version, unless historical view is requested

---

## 15. Lifecycle model

Not every entity type needs the same lifecycle, but lifecycle must be explicit.

### 15.1 Common lifecycle families

#### Document-like governed artifacts
Examples:
- SOP
- Policy
- Handbook
- Process

Suggested lifecycle:
- `draft`
- `in_review`
- `approved`
- `active`
- `stale`
- `archived`

#### Decision-like artifacts
Examples:
- Decision

Suggested lifecycle:
- `proposed`
- `confirmed`
- `superseded`
- `archived`

#### Work-state artifacts
Examples:
- Project
- Initiative
- Incident

Suggested lifecycle:
- `draft`
- `active`
- `blocked`
- `completed`
- `archived`

#### Derived insights
Examples:
- Insight
- CustomerInsight
- generated summaries

Suggested lifecycle:
- `candidate`
- `reviewed`
- `approved`
- `archived`

### 15.2 Lifecycle constraints

- Lifecycle transitions should be controlled by Workflow & Governance
- Certain transitions should require review or approval
- Superseded objects must remain retrievable as historical context
- Stale objects must remain visible as stale, not silently hidden

---

## 16. Truth classification model

Truth classification is one of the most important modeling concepts in the system.

### 16.1 Canonical in platform

Meaning:
The platform is authoritative for the object’s current state.

Typical examples:
- SOP
- Policy
- Team Handbook

Implications:
- native editing and workflow
- platform-controlled lifecycle
- strong governance expectations

---

### 16.2 Mirrored authority

Meaning:
The source of truth lives in another system; the platform reflects it.

Typical examples:
- Jira-driven project
- Trello-driven initiative
- external reference documents

Implications:
- display external authority clearly
- preserve external references
- do not mislead users into thinking the platform owns truth

---

### 16.3 Derived artifact

Meaning:
The object was synthesized or extracted from inputs and may require confirmation or review.

Typical examples:
- weekly summary
- planning summary
- extracted decision candidate
- synthesized meeting summary

Implications:
- confidence may be shown
- provenance is critical
- review gating may apply
- UI must not overstate authority

---

## 17. Modeling rules for v1 AI outputs

AI outputs should map into the domain model explicitly.

### 17.1 Allowed patterns

AI may create:
- candidate Decision objects
- draft Insight objects
- derived Meeting summaries
- draft JobOutput artifacts
- suggested EntityLink relations
- stale or duplicate suggestions

### 17.2 Disallowed patterns

AI should not:
- silently mutate canonical objects with no review path
- create ambiguous generic blobs when a typed entity fits
- collapse provenance into plain text only
- skip truth classification

### 17.3 Preferred output style

Use typed output structures and then route them into:
- draft entity creation
- review queue
- suggestion lists
- publication flow

---

## 18. Recommended data model strategy

For v1, a pragmatic model is preferred.

Suggested approach:
- strong shared base metadata
- typed entity records or typed payload extensions
- explicit relation table
- explicit provenance table
- explicit version table
- explicit workflow tables
- separate raw artifacts from governed entities

Do not over-normalize too early if it slows delivery.
Do not under-model core governance concepts either.

---

## 19. Anti-patterns to avoid

Do not:
- treat every source document as automatically canonical
- use one giant untyped blob for all entity semantics
- skip owner and domain on important objects
- mix raw artifacts and knowledge entities in one conceptual layer
- make truth classification implicit
- rely on vector search objects as primary truth
- create too many entity types too early
- treat review and approval as optional metadata with no workflow meaning

---

## 20. Open modeling questions

- Should Action Item become a first-class entity in v1 or remain derived/supporting output?
- Should summaries always live as JobOutput-derived artifacts, or sometimes as Meeting/Insight entities?
- Which entity types require mandatory approval versus optional review?
- How strict should Decision confirmation requirements be by domain?
- Should relation typing stay globally shared or be partially domain-specific later?
- How much structured schema should exist per entity type in v1 versus extension fields?

---

## 21. Final modeling stance

The domain model should make the system feel trustworthy and operable.

A user should be able to tell:
- what an object is
- who owns it
- whether it is authoritative
- where it came from
- how current it is
- what it is related to
- what changed over time

If the model cannot answer those questions clearly, it is not strong enough.