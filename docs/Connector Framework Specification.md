# Connector Framework Specification

> **Scope:** product-level framework principles, families, and rollout. For the technical reference (tables, code paths, normalization pipeline) see [connector-framework.md](./connector-framework.md). The two documents are complementary, not duplicates: this one is the framework contract; `connector-framework.md` is the implementation map.

## 1. Purpose

This document defines the connector framework for the Organizational Memory / Company Brain platform.

The goal of the framework is to ingest governed knowledge from multiple external systems in a consistent, extensible and policy-aware way.

The framework must support:

* multiple connector families;
* explicit source feed registration;
* source ownership;
* domain mapping;
* sensitivity inheritance;
* raw artifact persistence;
* normalization into internal models;
* safe async sync flows;
* extensibility without architectural rewrite.

This is not a collection of ad hoc integrations.
This is a governed ingestion framework.

---

## 2. Core Principles

### 2.1 Connector, Source Feed and Raw Artifact are separate concepts

These must never be collapsed into one model.

### 2.2 No unrestricted ingestion

No connector may read or import everything by default.
Every feed must be explicitly configured.

### 2.3 Governance starts at ingestion

Domain, sensitivity, owner and allowed usage must be attached at source feed level.

### 2.4 Raw context is preserved

Imported raw artifacts may be stored for provenance, replay and debugging.

### 2.5 Canonical knowledge is downstream

Connectors do not write directly into final knowledge areas without controlled normalization and policy application.

### 2.6 Connector families share contracts, not identical payloads

The framework should standardize ingestion mechanics without pretending all source systems are semantically identical.

---

## 3. Core Model

## 3.1 Connector

**Canonical definition:** matches [GLOSSARY.md](./GLOSSARY.md) and [connector-framework.md](./connector-framework.md). A **connector** is a **registered integration kind** (plugin type): it defines how to talk to a class of external systems (`connectors.type`, adapter registry, shared validation). It is **not** the governed instance.

The **governed instance** is the **source feed** (`source_feeds`): domain, sensitivity, sync mode, scope, and connector-specific config live on the feed row.

*(Historical note: earlier drafts of this spec used “configured integration account” for connector; that wording conflated kind and instance. Prefer glossary terminology everywhere new text is written.)*

Examples of **connector types** (kinds):

* Slack
* Telegram
* Jira
* Notion
* Microsoft 365
* HubSpot

### Connector responsibilities

* store integration identity/config references;
* validate connection configuration;
* resolve adapter implementation;
* support feed discovery where relevant.

### Minimum connector fields

* id
* type
* name
* status
* auth_config_ref
* config_json optional
* created_at
* updated_at

---

## 3.2 Source Feed

Represents a governed ingestible source inside a connector.

Examples:

* one Telegram chat
* one Slack channel
* one Jira board
* one Notion space
* one Fireflies transcript feed
* one Outlook mailbox folder
* one Teams channel

### Source Feed responsibilities

* define governance boundary for ingestion;
* define domain and sensitivity;
* define owner;
* define sync mode;
* define allowed knowledge job usage.

### Minimum source feed fields

* id
* connector_id
* external_ref
* name
* owner_user_id and/or owner_team_id
* domain_id
* knowledge_scope
* sensitivity_level
* sync_mode
* allowed_jobs_policy
* active
* config_json optional
* last_synced_at optional
* sync_status optional
* created_at
* updated_at

---

## 3.3 Raw Artifact

Represents imported raw source material.

Examples:

* Telegram message batch
* Slack thread payload
* Jira issue payload
* Notion page payload
* transcript payload
* Outlook email thread payload
* Teams conversation payload

### Raw Artifact responsibilities

* preserve source payload reference;
* preserve provenance;
* support replay/reprocessing;
* support debugging and audit.

### Minimum raw artifact fields

* id
* source_feed_id
* external_artifact_id
* artifact_type
* raw_storage_path
* checksum
* metadata_json
* imported_at

---

## 3.4 Normalized Artifact

Represents connector-family-shaped normalized content before canonical knowledge mapping.

Examples:

* normalized chat message
* normalized task item
* normalized document page
* normalized meeting transcript block
* normalized email message

This layer may be implemented as a persisted object or as a processing model depending on implementation choice.

---

## 3.5 Canonical Entity

Represents final governed internal knowledge object.

Examples:

* Decision
* Meeting
* Project update
* Insight
* SOP draft
* Policy summary

Canonical entities are created downstream from ingestion and normalization.

---

## 4. Connector Families

The connector framework should be built around universal families rather than one-off integration logic.

---

## 4.1 Chat Connector Family

### Covered systems

* Slack
* Telegram
* Microsoft Teams later or through Microsoft 365 family

### Typical feed types

* channel
* private channel
* group chat
* direct chat
* thread collection

### Typical artifacts

* message
* message batch
* thread
* reply set
* attached file reference

### Common normalized fields

* external_message_id
* external_thread_id optional
* sender_id
* sender_name optional
* text
* timestamp
* channel_id/chat_id
* attachments metadata
* reply_count optional
* source_feed_id

### Common use cases

* daily digest
* blocker extraction
* decision extraction
* recurring issue monitoring
* project communication summary

### Special note for Telegram

Telegram in v1 is ingestion-only and only via explicitly connected chats.

---

## 4.2 Microsoft 365 Connector Family

### Covered systems

* Outlook / Exchange Online
* Microsoft Teams
* OneDrive / SharePoint
* Microsoft Calendar

### Why this family exists

Microsoft data sources are often deployed as one ecosystem and should be modeled as one connector family with multiple feed types rather than as isolated one-off integrations.

### Typical feed types

* mailbox
* folder
* shared mailbox
* Teams channel
* Teams chat
* OneDrive folder
* SharePoint document library
* calendar scope
* event collection

### Typical artifacts

* email message
* email thread
* chat message
* channel thread
* file metadata
* document reference
* calendar event
* meeting metadata

### Common normalized fields

* external_object_id
* external_parent_id optional
* title_or_subject
* body_or_text optional
* sender_or_owner
* participants optional
* path_or_container_ref optional
* timestamp_or_modified_at
* attachments metadata optional
* source_feed_id

### Common use cases

* executive communication memory
* team communication ingestion in Microsoft environments
* document and file memory
* meeting boundaries and trigger context
* cross-source weekly digests in Microsoft-heavy companies

---

## 4.3 Email Connector Family

### Covered systems

* Gmail
* Outlook / Microsoft 365 Mail later if not handled through Microsoft family abstraction
* shared inboxes later

### Typical feed types

* inbox
* label/folder
* shared mailbox
* search-defined mailbox scope

### Typical artifacts

* message
* message thread
* attachment metadata

### Common normalized fields

* external_message_id
* external_thread_id
* sender
* recipients
* cc
* subject
* body
* timestamp
* labels/folders
* attachment refs
* source_feed_id

### Common use cases

* client communication memory
* decision trail extraction
* executive summary
* support escalation intelligence

---

## 4.4 Docs / Wiki Connector Family

### Covered systems

* Notion
* Confluence
* Google Docs
* Coda later

### Typical feed types

* workspace scope
* space
* database
* document collection
* page collection

### Typical artifacts

* page
* doc
* block tree
* database row/page

### Common normalized fields

* external_doc_id
* title
* content/body
* parent_ref
* owner/editor optional
* last_modified_at
* tags/labels optional
* source_feed_id

### Common use cases

* SOP ingestion
* wiki synchronization
* policy tracking
* project documentation memory

---

## 4.5 File Storage Connector Family

### Covered systems

* Google Drive
* OneDrive / SharePoint later or through Microsoft 365 family
* Dropbox later

### Typical feed types

* folder
* shared drive
* file query set

### Typical artifacts

* file metadata
* exported text payload
* binary file reference

### Common normalized fields

* external_file_id
* title/file_name
* mime_type
* owner
* path/folder context
* modified_at
* storage reference
* extracted_text optional
* source_feed_id

### Common use cases

* document corpus ingestion
* file-linked project memory
* compliance/reference storage

---

## 4.6 Work Management Connector Family

### Covered systems

* Jira
* Trello
* Asana later
* Linear later

### Typical feed types

* board
* project
* sprint/cycle
* filtered issue/task set

### Typical artifacts

* issue
* card
* task
* comment thread
* sprint snapshot

### Common normalized fields

* external_item_id
* title
* description
* status
* assignee
* reporter/creator
* labels
* comments summary or refs
* created_at
* updated_at
* project/board ref
* source_feed_id

### Common use cases

* sprint planning memory
* execution digest
* blocker monitoring
* decision linkage to work items

---

## 4.7 CRM / Support Connector Family

### Covered systems

* HubSpot
* Intercom
* Zendesk

### Typical feed types

* pipeline
* inbox
* ticket queue
* object collection
* saved view

### Typical artifacts

* ticket
* conversation
* CRM object
* note
* activity/event

### Common normalized fields

* external_object_id
* object_type
* title/subject
* owner
* customer/company ref
* stage/status
* notes/comments/messages
* created_at
* updated_at
* source_feed_id

### Common use cases

* customer insight extraction
* recurring objections
* support issue trend analysis
* client relationship memory

---

## 4.8 Meeting / Transcript Connector Family

### Covered systems

* Zoom
* Fireflies
* Granola
* Google Calendar as event/context layer

### Typical feed types

* meeting stream
* transcript feed
* tagged meeting collection
* calendar scope

### Typical artifacts

* meeting record
* transcript
* meeting notes
* participants list
* summary payload

### Common normalized fields

* external_meeting_id
* title
* organizer
* participants
* start_time
* end_time
* transcript text
* summary text optional
* calendar_event_ref optional
* source_feed_id

### Common use cases

* planning summary
* decision extraction
* action extraction
* weekly meeting digest

---

## 5. Universal Connector Contract

Each connector adapter must conform to a common framework contract.

### Required responsibilities

* validate connector configuration
* list available feeds where applicable
* sync one source feed
* map connector-specific raw metadata into framework shape
* support incremental sync where possible
* support event/webhook handling where possible

### Conceptual adapter interface

* `ValidateConfig()`
* `ListAvailableFeeds()` optional depending on source type
* `SyncFeed(feed, cursor/window)`
* `NormalizeArtifactMetadata(raw)`
* `HandleWebhook(event)` optional

Exact names may vary, but the responsibility model must remain stable.

---

## 6. Sync Modes

Supported sync modes:

* `full_import`
* `incremental`
* `event_driven`
* `manual`

### full_import

Used for initial historical ingestion.

### incremental

Used for periodic sync based on cursor/time/version.

### event_driven

Used when the source supports webhook/event push.

### manual

Used when sync is explicitly triggered by user/admin.

Not every connector must support every mode, but each source feed must declare one of them.

---

## 7. Universal Sync Flow

The connector framework must support the following sync flow:

1. sync requested manually, by scheduler or event
2. load source feed
3. validate source feed is active and governed
4. resolve connector
5. resolve adapter implementation
6. fetch source data
7. persist raw artifacts
8. attach provenance metadata
9. emit sync audit/run records
10. hand off to normalization / downstream processing

Connectors must not bypass this flow.

---

## 8. Policy Inheritance Rules

Governance begins at source feed level.

### What must be inherited from source feed

* domain
* sensitivity level
* ownership context
* allowed job usage
* source provenance

### What downstream processing must preserve

* source_feed_id
* source artifact references
* effective governance context

Connector data must not enter the platform as unclassified neutral text.

---

## 9. Telegram v1 Constraints

Telegram is supported in v1 only as a controlled ingestion source.

### Mandatory rules

* only explicitly connected chats may be ingested
* each Telegram source feed must define owner, domain, knowledge scope, sensitivity and allowed jobs
* no unrestricted crawling
* no "read all available chats" behavior
* imported artifacts must inherit Telegram source feed policy

### Telegram feed types for v1

* group chat
* private internal team chat
* project chat
* announcement channel only if explicitly supported later

### Telegram artifacts for v1

* messages
* grouped message batches
* metadata about sender/time/threading where available

**Implementation reference:** [TELEGRAM_CONNECTOR_V1.md](./TELEGRAM_CONNECTOR_V1.md)

---

## 10. Permissions and Governance Integration

The connector framework must integrate with platform access and governance rules.

### Required controls

* only authorized users may create/update connectors
* only authorized users may create/update source feeds
* only authorized users may trigger syncs
* source feeds must have controlled visibility
* raw artifacts must not become broadly visible by default

Ingestion is not outside the permission model.

---

## 11. Normalization Strategy

Do not build one giant universal parser.

### Correct approach

* common ingestion framework
* family-specific normalizers
* common provenance model
* common governance model
* common downstream canonical mapping layer

### Incorrect approach

* forcing chat messages, emails, transcripts, tasks and docs into the same raw semantic shape too early

The framework should unify mechanics, not erase meaning.

---

## 12. Data Model Alignment

The connector framework should align with these core tables:

* connectors
* source_feeds
* raw_artifacts
* sync_jobs or sync_runs

Optional supporting tables:

* connector_run_logs
* feed_sync_state
* webhook_events
* normalized_artifacts if persisted separately

---

## 13. Worker Architecture

The framework must support async execution for:

* scheduled syncs
* manual syncs
* event-driven ingestion
* retry of failed syncs
* downstream normalization triggers

Long-running sync work must not run inside HTTP handlers.

Suggested worker categories:

* source sync worker
* raw artifact processing worker
* webhook event processor
* retry worker

---

## 14. Rollout Priority

The rollout should be based on connector families, not random integrations.

## Phase 1

### Chat family

* Slack
* Telegram

### Docs family

* Notion
* Google Docs / Drive

### Work family

* Jira

### Meeting family

* Fireflies or Zoom
* Google Calendar as trigger/context

## Phase 2

* Microsoft 365 Mail / Outlook
* Microsoft Teams
* Trello
* Confluence

## Phase 3

* Intercom
* HubSpot
* OneDrive / SharePoint
* Zendesk
* Asana
* Linear

This order gives the highest initial coverage of real organizational memory.

---

## 15. Top-20 Candidate Systems

Priority list of common data suppliers for founders, teams and corporations:

1. Slack
2. Telegram
3. Microsoft 365 Mail / Outlook
4. Microsoft Teams
5. Microsoft OneDrive / SharePoint
6. Google Workspace Mail / Gmail
7. Google Drive
8. Google Docs
9. Google Calendar
10. Notion
11. Confluence
12. Jira
13. Trello
14. Asana
15. Linear
16. Intercom
17. Zendesk
18. HubSpot
19. Zoom
20. Fireflies / Granola

---

## 16. Implementation Guidance for Cursor

When implementing the connector framework:

* keep Connector, SourceFeed and RawArtifact as separate models
* implement family-safe adapter contracts
* preserve provenance from the first sync onward
* apply source feed governance defaults before downstream processing
* keep Telegram restrictions explicit in code
* avoid connector-specific hacks inside generic orchestration layers
* design for async execution from the start
* model Microsoft 365 as a connector family, not as a single narrow Outlook-only integration

---

## 17. Next Documents

After this document, the next logical connector-related documents are:

1. Connector API Contract
2. Source Feed Admin UX Spec
3. Normalization Spec by Connector Family
4. Telegram Connector Spec v1
5. Slack Connector Spec v1
6. Jira Connector Spec v1
7. Notion / Google Docs Connector Spec v1
8. Microsoft 365 Connector Family Spec v1
