# Extracted meeting tasks (draft → confirm)

## Purpose

Structured storage for **LLM-extracted action items** from meeting transcripts (or related normalized records), with a human-in-the-loop lifecycle. Supports quality metrics such as “confirmed without edits” without coupling to the client’s exact Web UI.

This complements:

- **Jira `work_item`** — authoritative issues in Jira.
- **`decision_extraction` job outputs** — decisions as entities.
- **Governance `review_tasks`** — human review queues for risky AI outputs.

Extracted tasks are **provisional** until confirmed.

## Lifecycle

| `review_status` | Meaning |
|-----------------|--------|
| `draft` | Created by extraction (job or future API); editable |
| `confirmed` | User accepted as-is (`confirm_no_edit` event) |
| `edited` | User saved changes (`confirm_after_edit` event) |
| `rejected` | User discarded (`reject` event) |

Transitions are append-only in `extracted_task_review_events` for OKR-style analytics.

## Schema (Postgres)

See migration `000037_extracted_meeting_tasks.up.sql`:

- **`extracted_meeting_tasks`** — title, description, assignee hints, deadline, priority, foreign keys to `domains`, optional `source_feeds` / `normalized_records` / `entities` (meeting).
- **`extracted_task_review_events`** — `event_type`: `created`, `confirm_no_edit`, `confirm_after_edit`, `reject`, `edit_save`; `detail_json` for diff summaries.

## Relations

- **Meeting context**: `source_normalized_record_id` when the source row is a `meeting_transcript` or `calendar_event`, and/or `linked_meeting_entity_id` when a canonical meeting entity exists.
- **Decisions**: `linked_decision_entity_ids` optional UUID array for traceability.
- **Participants**: `participant_refs` text array (opaque handles: email, Mattermost user id, etc.) — normalization to `users` is a later enrichment step.

## Application API (v1)

Domain-scoped list/create, single-task get/patch, confirm/reject routes, metrics and product-event listing are documented in [API_SURFACE_V1.md](./API_SURFACE_V1.md) §14.1. Web UI: **`/meeting-tasks`** in the admin shell. Optional **LLM auto-extraction** into new rows remains a separate knowledge-job type (not wired in this repo iteration).

## OKR-oriented metrics

From `extracted_task_review_events`:

- **Precision proxy**: ratio of `confirm_no_edit` to all confirms.  
- **Abandonment**: `reject` rate.  
- **Edit burden**: `confirm_after_edit` count / `edit_save` volume.
