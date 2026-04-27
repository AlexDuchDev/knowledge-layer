# INGESTION_AND_CONNECTORS.md

**Connector framework specification (product + families):** [Connector Framework Specification.md](./Connector%20Framework%20Specification.md) — governed ingestion model, separate Connector / Source Feed / Raw Artifact concepts, connector families, universal adapter contract, sync modes and flow, policy inheritance, Telegram v1 constraints, data model alignment, worker architecture, rollout phases.

**Implementation map (code + schema):** [connector-framework.md](./connector-framework.md) — connectors vs source feeds vs raw artifacts, adapter interface (+ optional `WebhookHandler`), sync pipeline helpers, Telegram v1 rules, migrations `000020`, `000023` (indexes).

**Telegram v1 (operational):** [TELEGRAM_CONNECTOR_V1.md](./TELEGRAM_CONNECTOR_V1.md) — supported scope, restrictions, sync flow, governance checkpoints.

**Chat connector family (normalized shape, feed kinds):** [chat-connector-family.md](./chat-connector-family.md).

**Other connector families (DTOs + v1 notes):** [docs-wiki-connector-family.md](./docs-wiki-connector-family.md), [work-management-connector-family.md](./work-management-connector-family.md), [meeting-transcript-connector-family.md](./meeting-transcript-connector-family.md), [microsoft-365-connector-family.md](./microsoft-365-connector-family.md), [email-connector-family.md](./email-connector-family.md), [crm-support-connector-family.md](./crm-support-connector-family.md).

**Operator truth table (sync vs normalize):** [CONNECTOR_CAPABILITY_MATRIX.md](./CONNECTOR_CAPABILITY_MATRIX.md).

## 1. Purpose

This document defines the ingestion and connector model for the Organizational Memory & Knowledge Operations Platform.

It establishes:
- the purpose of the ingestion layer
- the connector framework contract
- the source feed model
- ingestion pipeline stages
- sync modes and trigger modes
- normalization and deduplication behavior
- failure and retry expectations
- governance and access control points
- Telegram-specific constraints for v1
- operational expectations for connector quality

This document is the product and system contract for how external knowledge enters the platform.

---

## 2. Why ingestion matters

The platform does not begin with retrieval.
It begins with controlled ingestion.

If the system ingests the wrong things, ingests too broadly, loses provenance, or fails to preserve source boundaries, then:
- knowledge objects become untrustworthy
- review workflows become noisy
- retrieval quality degrades
- access inheritance becomes unreliable
- AI outputs become unsafe

The ingestion layer must therefore be:
- explicit
- governed
- traceable
- resilient
- reprocessable
- connector-extensible

---

## 3. Core ingestion principles

### 3.1 Ingestion is controlled, not ambient
The platform only ingests from explicitly configured sources.
No connector should operate as a broad, implicit company-wide data vacuum.

### 3.2 Source feeds are governance boundaries
A configured source feed is the main control point for determining:
- owner
- domain
- sensitivity
- allowed jobs
- ingestion mode
- access inheritance

### 3.3 Raw inputs must be preserved
The system should preserve raw artifacts so that:
- provenance remains durable
- normalization can be improved later
- reprocessing is possible
- audit remains explainable

### 3.4 Ingestion does not equal publication
A source being ingested does not mean its content becomes canonical knowledge automatically.

### 3.5 Connectors should normalize, not decide truth
Connector logic should fetch, parse, and normalize source data.
It should not decide what becomes an authoritative knowledge object.

### 3.6 Connector-specific complexity must not leak too far
The rest of the platform should depend on stable ingestion contracts, not on the quirks of individual external tools.

### 3.7 Ingestion must remain explainable
For any important downstream artifact, the system should be able to explain:
- which source feed it came from
- which raw artifacts were used
- when ingestion happened
- what parsing and normalization version was applied

---

## 4. Connector framework overview

The connector framework is the system layer responsible for integrating external knowledge systems into governed source feeds.

### 4.1 Supported source categories in v1

#### Communication
- Slack
- Email
- Telegram

#### Meeting / conversation tools
- Fireflies
- Granola
- manual transcript upload

#### Project / task systems
- Jira
- Trello
- Asana
- Linear

#### Documentation systems
- Notion
- Google Drive / Docs
- Confluence

#### CRM / support
- Intercom
- HubSpot
- Zendesk

#### Microsoft 365 (Graph)
- Outlook mail, Teams, OneDrive, SharePoint libraries, Outlook calendar

#### Email
- Gmail (family `email`); Outlook mail is ingested via the Microsoft 365 connector (`m365_mail_message`).

### 4.2 Connector framework responsibilities

The framework must support:
- authentication
- source discovery or source mapping
- source feed configuration
- sync execution
- event handling where supported
- raw artifact storage
- content parsing
- metadata extraction
- normalization
- deduplication
- sync health reporting
- retry and backoff behavior
- operational observability

### 4.3 Connector framework non-responsibilities

The framework should not:
- define canonical truth
- decide final lifecycle state of knowledge objects
- bypass access policy assignment
- publish critical outputs directly
- contain ad hoc business logic for unrelated domains

---

## 5. Main connector concepts

### 5.1 Connector
A connector represents the integration logic for one external system or source type.

Examples:
- Telegram connector
- Slack connector
- Email connector
- Fireflies connector
- Jira connector

A connector may support:
- auth flow
- source mapping
- sync modes
- parsing strategies
- event subscriptions
- metadata extraction

### 5.2 Source feed
A source feed is a governed configured input boundary inside the platform.

A source feed includes:
- connector type
- mapped source reference
- owner
- domain
- sensitivity
- allowed jobs
- ingestion mode
- sync mode
- access policy
- operational status

A source feed is the unit of governance for ingestion.

### 5.3 Raw artifact
A raw artifact is the original or near-original payload fetched from a source.

Examples:
- transcript export
- email thread body
- Slack message batch
- Telegram message batch
- Notion page snapshot
- Jira issue snapshot

### 5.4 Normalized record
A normalized record is the structured internal ingestion output derived from raw artifacts.

It is the stable handoff between connector-specific parsing and downstream platform logic.

### 5.5 Ingestion run
An ingestion run is one execution instance that fetches and processes source content.

---

## 6. Required connector contract

Every connector should support a common conceptual contract, even if some capabilities differ by source.

### 6.1 Authentication model

Each connector must define:
- auth type
- token or credential requirements
- credential storage method
- refresh behavior if applicable
- validation behavior
- revocation handling

### 6.2 Source mapping model

Each connector must define how a source is mapped into a source feed.

Examples:
- Telegram chat
- Slack channel
- Email mailbox or label
- Fireflies workspace or meeting source
- Jira project or board
- Notion page tree or database

### 6.3 Sync mode support

Each connector should declare whether it supports:
- full import
- incremental sync
- event-driven ingestion

### 6.4 Event support model

If the source supports events or webhooks, the connector should define:
- supported event types
- delivery reliability assumptions
- replay behavior
- deduplication strategy for repeated events

### 6.5 Parsing model

Each connector must define:
- supported artifact types
- parsing strategy
- handling of partial or malformed payloads
- fallback behavior

### 6.6 Metadata extraction model

Each connector should extract useful source metadata where available, such as:
- source timestamps
- source authors
- participants
- thread identifiers
- channel or chat identifiers
- meeting identifiers
- issue identifiers
- document URLs or external refs

### 6.7 Deduplication hooks

Each connector should provide enough stable identifiers and hashes to support deduplication.

### 6.8 Retry and failure model

Each connector must define:
- retryable errors
- terminal errors
- backoff behavior
- alert thresholds
- degraded mode behavior

### 6.9 Health reporting

Each connector should expose:
- current status
- last successful sync
- last attempted sync
- recent error count
- recent warning count

---

## 7. Source feed model

A source feed is the central operational and governance object for ingestion.

### 7.1 Required source feed fields

Each source feed should include:
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
- `access_policy_id`
- `status`
- `last_sync_at`
- `last_successful_sync_at`
- `health_status`
- `created_at`
- `updated_at`

### 7.2 Required source feed governance fields

Each source feed must also carry:
- policy source
- reviewability expectations where relevant
- source scope description
- reason for connection or business purpose
- optional data retention profile
- optional sanitization profile

### 7.3 Why source feeds matter

Source feeds are where the platform makes ingestion governable.

Without source feeds, the system cannot reliably answer:
- who owns this source
- what domain it belongs to
- how sensitive it is
- which jobs may run on it
- who is allowed to operate on it

---

## 8. Source feed lifecycle

A source feed should move through explicit operational states.

### 8.1 Suggested source feed states

- `draft`
- `configured`
- `active`
- `paused`
- `error`
- `archived`

### 8.2 Source feed lifecycle meaning

#### `draft`
Source feed setup has started but is incomplete.

#### `configured`
Source feed is fully configured but not yet actively syncing.

#### `active`
Source feed is actively eligible for sync and downstream processing.

#### `paused`
Source feed remains configured but no sync is currently running.

#### `error`
Source feed is unhealthy or blocked due to connector or config failure.

#### `archived`
Source feed is no longer operational but remains historically visible for provenance.

### 8.3 Lifecycle rule

A source feed must not enter `active` until all required governance fields are present:
- owner
- domain
- sensitivity
- allowed jobs
- ingestion mode

---

## 9. Ingestion modes

Ingestion mode describes the intended product behavior for how a source is used.

### 9.1 Allowed ingestion modes for v1

Suggested modes:
- `raw_capture_only`
- `governed_processing`
- `governed_processing_with_jobs`

### 9.2 Mode meaning

#### `raw_capture_only`
The platform stores raw artifacts and metadata, but does not automatically generate derived artifacts beyond basic indexing.

Use when:
- source is still being evaluated
- governance is not ready for richer processing
- business wants a conservative first step

#### `governed_processing`
The platform may normalize and index source content and support constrained downstream workflows.

#### `governed_processing_with_jobs`
The source feed may participate in allowed knowledge jobs such as summarization or extraction.

### 9.3 Product rule

A source feed’s ingestion mode must constrain what the rest of the system is allowed to do with its content.

---

## 10. Sync modes

Sync mode describes how source updates enter the platform.

### 10.1 Supported sync modes

- `full_import`
- `incremental_sync`
- `event_driven`

### 10.2 Full import

Used when:
- source needs an initial backfill
- source lacks good incremental support
- operator explicitly requests complete refresh

Tradeoffs:
- more expensive
- higher duplicate risk unless handled carefully
- good for setup or controlled reprocessing

### 10.3 Incremental sync

Used when:
- source supports cursor-based or time-window-based updates
- routine sync is needed
- volume is high enough to avoid repeated full fetches

### 10.4 Event-driven ingestion

Used when:
- source supports reliable events or webhook delivery
- low-latency updates matter
- operational freshness matters

### 10.5 Mixed-mode reality

Some connectors may use:
- initial full import
- then incremental sync
- with event-driven updates where possible

The framework should support this without pretending every source behaves the same way.

---

## 11. Trigger modes for ingestion

An ingestion run may begin because of different triggers.

### 11.1 Supported trigger types

- manual
- scheduled
- event-driven
- retry
- reprocessing
- backfill

### 11.2 Trigger semantics

#### manual
Started by an admin or operator.

#### scheduled
Started on configured cadence.

#### event-driven
Started by source event or webhook.

#### retry
Re-attempt of a failed run or failed subset.

#### reprocessing
Re-run normalization or downstream processing against preserved raw artifacts.

#### backfill
Historical import over a defined older time window or source range.

---

## 12. End-to-end ingestion pipeline

The ingestion pipeline should remain structurally explicit.

### 12.1 Stage 1 — Source feed configuration

An admin or governance operator:
1. selects connector type
2. authenticates connector
3. maps source
4. assigns owner
5. assigns domain
6. assigns sensitivity
7. assigns allowed jobs
8. selects ingestion mode and sync mode
9. saves source feed

The platform validates the configuration before activation.

### 12.2 Stage 2 — Fetch

The connector runtime:
1. resolves source credentials
2. resolves source mapping
3. fetches source content
4. stores raw payloads or snapshots as raw artifacts
5. records ingestion run metadata

### 12.3 Stage 3 — Parse

The connector parser:
1. interprets payloads
2. extracts content body or structured fields
3. extracts source metadata
4. handles malformed payloads
5. emits parsed internal representations

### 12.4 Stage 4 — Normalize

The normalization stage:
1. maps parsed content into normalized records
2. applies stable internal field structures
3. attaches source references
4. records normalization version
5. prepares outputs for indexing or downstream jobs

### 12.5 Stage 5 — Deduplicate

The deduplication stage:
1. checks external IDs
2. checks content hashes
3. checks source-window collisions
4. suppresses or links duplicates
5. records deduplication outcome

### 12.6 Stage 6 — Downstream routing

After normalized records are created, the system may route them to:
- indexing
- retrieval preparation
- job eligibility
- candidate canonicalization flows
- reviewable derived artifact generation
- observability and audit systems

Important rule:
Downstream routing must remain policy-aware.

### 12.7 Mega-files, concatenated transcripts, and session splitting

Exports from chat tools, Slack, or meeting vendors sometimes arrive as **one large file** containing **many sessions or days** in a single blob. Ingesting that as a single raw artifact and normalized record tends to:

- produce **poor chunk boundaries** for retrieval and embeddings
- obscure **per-session provenance** (who said what, when)
- make **review and deduplication** harder

**Product stance:** Prefer **one logical session (or one bounded time window)** per raw artifact where the source format allows it, while still preserving the **original export** as evidence when policy requires ([adr/0006-raw-artifacts-must-be-preserved.md](./adr/0006-raw-artifacts-must-be-preserved.md)).

**Recommended patterns (adapted from common local-memory tooling):**

1. **Pre-split step (optional connector phase):** Before or during **Parse** (§12.3), detect session boundaries (export markers, timestamps, participant headers) and emit **multiple parsed segments**, each becoming its own raw artifact or child record—**without** dropping the parent export if retention policy requires keeping the bundle.
2. **Operator-controlled job:** A **knowledge job** or connector sync option (e.g. “split mega-export”) runs idempotently, writes new raw artifacts with links to the parent hash, and re-runs normalization.
3. **Dry-run / preview:** Admin or operator UI reports how many sessions would be created (reduces accidental fragmentation).

**Scope:** Highest priority for **chat** and **meeting transcript** families ([chat-connector-family.md](./chat-connector-family.md), [meeting-transcript-connector-family.md](./meeting-transcript-connector-family.md)). Not a global default for all connectors.

**Backlog (engineering):**

- Define a **stable internal format** for “session boundary” metadata (start time, end time, source message ids).
- Add connector-specific adapters where exports differ (Claude export vs Slack JSON vs vendor transcript).
- Wire chunk rebuild / embedding enqueue after split so retrieval improves without manual reindex.

---

## 13. Raw artifact model

Raw artifacts are foundational for trust and reprocessing.

### 13.1 What raw artifacts should contain

A raw artifact should retain:
- source feed reference
- external artifact reference where available
- original content or retrievable pointer
- source timestamp
- content hash
- source author or participants where available
- ingestion run reference
- metadata payload

### 13.2 Why raw artifacts are necessary

They support:
- provenance
- reprocessing
- debugging
- audit
- improved normalization later
- future extraction improvements

### 13.3 What raw artifacts are not

Raw artifacts are not:
- canonical knowledge objects
- automatically searchable final truth
- automatically publishable content
- user-facing governed artifacts by default

Raw artifacts are evidence, not authority.

### 13.4 Artifact queue worker audit

After the connector worker processes a queued raw artifact ([`ProcessQueuedRawArtifact`](../apps/api/internal/ingestion_connectors/artifact_worker.go), task `ingestion:process_artifact` in [`cmd/connectorworker`](../apps/api/cmd/connectorworker/main.go)), the platform records an audit event:

- **`ingestion.artifact_processed`** — target type `raw_artifact`, actor **system**, metadata includes `outcome` (`success` | `error`), `artifact_type`, optional `error`, and identifiers (`raw_artifact_id`, `source_feed_id`, `connector_id`).

Schema-oriented detail and event typing live in [connector-framework.md](./connector-framework.md) (ingestion audit trail).

---

## 14. Normalized record model

Normalized records are the stable internal output of ingestion.

### 14.1 Why normalized records exist

They decouple:
- connector-specific quirks
from
- downstream platform logic

Without this layer, the rest of the system becomes polluted with connector-specific assumptions.

### 14.2 What normalized records should contain

A normalized record should include:
- record type
- normalized content body
- extracted metadata
- source references
- timestamp
- author or participant reference where available
- normalization version
- record hash

### 14.3 Example normalized record types

Examples:
- message
- thread
- meeting_transcript
- meeting_summary_source
- issue_snapshot
- document_snapshot
- email_thread
- board_card_snapshot

---

## 15. Deduplication model

Deduplication is required to avoid polluted retrieval and noisy downstream jobs.

### 15.1 Deduplication goals

The system should:
- avoid repeated raw artifacts where possible
- avoid duplicate normalized records
- avoid repeated downstream derived artifacts from the same input window
- preserve provenance even when duplicates are detected

### 15.2 Deduplication signals

Use combinations of:
- external IDs
- source feed IDs
- message or document timestamps
- stable thread identifiers
- content hashes
- normalization hashes
- job input window fingerprints

### 15.3 Deduplication outcomes

Possible outcomes:
- accept as new
- suppress as duplicate
- link as duplicate family
- reprocess because parsing version changed
- mark ambiguous for operator review in rare cases

### 15.4 Important rule

Deduplication should reduce noise, not destroy evidence.
When in doubt, preserve raw source evidence and suppress only downstream duplication.

---

## 16. Reprocessing model

The platform must support reprocessing.

### 16.1 Why reprocessing matters

Reprocessing is needed when:
- parsing improves
- normalization schema changes
- extraction logic improves
- embeddings need regeneration
- indexing changes
- prior runs failed partially

### 16.2 Reprocessing rules

Reprocessing should:
- start from preserved raw artifacts where possible
- record processing version
- avoid silently mutating published canonical artifacts
- create new downstream versions when material changes matter
- remain auditable

### 16.3 Reprocessing scope types

Suggested scope types:
- single raw artifact
- source feed window
- full source feed backfill
- connector-wide replay
- downstream-only reindex or rechunk

---

## 17. Connector-specific guidance for v1

### 17.1 Telegram

Telegram in v1 is supported only as a controlled ingestion source.

Required rules:
- only explicitly connected chats are allowed
- every chat must have an owner
- every chat must have a domain
- every chat must have a sensitivity level
- every chat must define allowed jobs
- Telegram is not a universal output channel
- Telegram is not a free-form assistant interface
- Telegram is not an unrestricted bot surface

Recommended Telegram normalized objects:
- message records
- thread or conversation window records where applicable
- participant references
- source-linked summary inputs

Telegram-specific product stance:
The platform reads Telegram as a governed operational signal source, not as an open conversational product surface.

### 17.2 Slack

Slack should be treated similarly to Telegram in governance structure, but may have richer thread and channel metadata depending on connector depth.

### 17.2a Mattermost (v1)

Mattermost is supported as **ingestion-only** chat sync (same product stance as Telegram for delivery: not a universal assistant surface).

- **Connector type:** `mattermost`
- **Auth:** personal access token in `connector_config_json.mattermost_token`; `mattermost_base_url` (e.g. `https://chat.example.com`, no trailing slash required).
- **Feed `external_ref`:** channel id to read via `GET /api/v4/channels/{channel_id}/posts`.
- **Incremental state:** `mattermost_sync_state.last_post_id` (Mattermost post id cursor).
- **Security:** token is highly sensitive; scope feeds to explicit channels; store only in feed config with least-privilege PAT; classify hosting (self-hosted vs cloud) under your data-processing agreements.

### 17.3 Email

Email ingestion should preserve:
- thread structure where available
- sender and recipient metadata
- subject
- time sequence

Important caution:
Email often carries higher sensitivity and noisier content.
Strong scope control is important.

### 17.4 Fireflies / Granola / manual transcripts

Meeting tools should preserve:
- transcript source
- meeting timestamp
- participants if available
- meeting identifiers
- title
- source link where possible

These sources are especially useful for:
- decision extraction
- summary generation
- meeting object creation
- action item extraction

### 17.5 Jira / Trello

Task system connectors are likely to produce:
- issue or card snapshots
- project references
- status changes
- assignee metadata

Important rule:
These often map better to mirrored-authority objects than platform-canonical truth.

Connector onboarding (minimal admin typing): discovery endpoints under `/integrations/*` use the same credentials as the eventual source feed and require `manage_source_feed` in a granted domain (optional `domain_id` in body; else first granted domain). Implemented pickers include Jira projects, Trello boards, Asana projects, Linear teams, Slack channels, Mattermost channels, Confluence spaces, Notion search results, Google Calendar list, Google Drive child folders, and Zendesk views. See [API_SURFACE_V1.md](./API_SURFACE_V1.md) §5.1 and [work-management-connector-family.md](./work-management-connector-family.md).

### 17.6 Notion / Google Docs

Documentation connectors should preserve:
- document title
- document structure where possible
- external refs
- snapshot timestamps
- folder or parent references where available

Important rule:
A mirrored document is not automatically canonical in platform just because it is imported.

---

## 18. Governance control points in ingestion

Ingestion must include explicit governance gates.

### 18.1 At source feed creation

The system must require:
- owner
- domain
- sensitivity
- allowed jobs
- ingestion mode

### 18.2 At activation

The system should validate:
- connector auth health
- source mapping validity
- required governance fields
- policy consistency

### 18.3 At downstream routing

The system should enforce:
- allowed job participation
- source-derived policy propagation
- sensitivity inheritance
- prohibition of uncontrolled publication

### 18.4 At reprocessing

The system should ensure:
- the action is authorized
- the reason is recorded
- output effects are visible
- previous provenance is preserved

---

## 19. Access and permissions in ingestion

Ingestion itself is permission-sensitive.

### 19.1 Source feed actions should be permissioned separately

At minimum:
- connect source
- edit source feed config
- activate source feed
- pause source feed
- trigger sync
- inspect raw artifacts
- view source-derived normalized records
- allow or deny jobs against the feed

### 19.2 Raw artifact access

Raw source artifacts are often more sensitive than downstream governed outputs.

Therefore:
- raw artifact viewing should be a separate permission
- not all viewers of derived knowledge should automatically see raw source payloads
- audit and review access to raw inputs should be deliberate

### 19.3 Worker execution rule

Workers performing ingestion must run under governed system scope and must not imply universal human-readable visibility.

---

## 20. Error handling and retries

Connectors are operationally messy by nature.
The system should assume this and design accordingly.

### 20.1 Error classes

Suggested error classes:
- auth failure
- source mapping failure
- transient fetch failure
- rate limit
- parse failure
- schema mismatch
- storage failure
- normalization failure
- downstream routing failure

### 20.2 Retry behavior

Retryable cases may include:
- transient network failures
- rate limits
- temporary source unavailability
- temporary storage issues

Non-retryable or operator-visible cases may include:
- invalid credentials
- deleted source
- permanently malformed configuration
- unsupported payload shape without fallback

### 20.3 Partial success model

An ingestion run may succeed partially.

Example:
- fetch succeeds
- some raw artifacts store correctly
- some payloads fail parsing
- the rest proceed to normalization

The system should record partial success rather than flatten everything into pass/fail.

### 20.4 Operator visibility

Operators should be able to see:
- recent failures
- repeated warnings
- last successful sync
- failure category
- suggested recovery path where possible

---

## 21. Scheduling and operational behavior

### 21.1 Scheduling support

The platform should support:
- scheduled sync
- manual sync
- event-driven sync
- backfill jobs
- reprocessing jobs

### 21.2 Operational safeguards

The system should support:
- pause and resume
- concurrency controls
- backoff on repeated failures
- rate limit awareness
- stuck-run detection
- duplicate-run suppression where appropriate

### 21.3 Run identity

Every ingestion run should have:
- stable run ID
- trigger type
- source feed ID
- start time
- end time
- summary counters
- warnings
- errors
- trace reference

---

## 22. Observability and health model

Ingestion must be observable enough to operate like infrastructure.

### 22.1 Source feed health indicators

Each source feed should surface:
- status
- health status
- last sync attempt
- last successful sync
- warning count
- error count
- backlog estimate if relevant

### 22.2 Connector-level metrics

Useful metrics:
- success rate by connector type
- median sync duration
- parse failure rate
- normalization failure rate
- retry rate
- rate limit incidents
- duplicate suppression rate

### 22.3 Governance metrics

Useful governance-focused metrics:
- sources missing owner
- sources missing domain
- sources with ambiguous allowed jobs
- high-sensitivity sources with no recent review
- sources producing unreviewed derived outputs

---

## 23. Data retention and storage posture

The platform should define a clear retention posture, even if v1 keeps it simple.

### 23.1 Retention principles

- preserve enough raw data for provenance and reprocessing
- avoid unnecessary long-term duplication where policy forbids it
- make retention configurable later without redesigning core models
- keep retention behavior visible for admins

### 23.2 Storage separation

Keep separate:
- raw artifacts
- normalized records
- governed entities
- search indexes
- embeddings
- execution traces

This makes the system easier to reason about and safer to evolve.

---

## 24. UX requirements for source operations

The admin and governance surfaces should make ingestion feel controlled and understandable.

### 24.1 Source Feeds UI should support

- connector selection
- source mapping
- owner assignment
- domain assignment
- sensitivity assignment
- allowed jobs selection
- ingestion mode selection
- sync mode selection
- source status visibility
- sync history visibility
- health visibility

### 24.2 Source detail view should support

- governance metadata
- recent ingestion runs
- raw artifact count
- normalized record count
- downstream usage summary
- errors and warnings
- pause/resume actions
- reprocessing actions for authorized users

### 24.3 Important UX rule

The UI should never make a source feel “connected and done.”
It should feel configured, governed, and monitored.

---

## 25. Testing requirements

The ingestion layer requires strong automated testing.

### 25.1 Mandatory test areas

- source feed validation
- missing governance field rejection
- connector auth failure handling
- source mapping correctness
- raw artifact storage behavior
- normalization output shape
- deduplication behavior
- retry behavior
- partial success handling
- reprocessing behavior
- policy propagation from source feed
- Telegram-specific restriction enforcement

### 25.2 Scenario examples

Examples to test:
- source feed missing owner cannot activate
- Telegram source with no allowed jobs is rejected
- duplicate Slack events do not create duplicate normalized records
- parsing version change triggers safe reprocessing
- raw artifact access denied to user with only derived-artifact visibility
- source feed pause prevents scheduled sync
- partial ingestion run still records usable artifacts and warnings

---

## 26. Anti-patterns to avoid

Do not:
- ingest sources without explicit feed configuration
- treat connector auth as enough governance
- publish canonical objects directly from connector code
- drop raw artifacts after normalization
- let source feed state drift from policy state
- let connectors define business truth
- overfit the internal model to one connector’s quirks
- assume every source supports reliable incrementality
- treat all imported docs as canonical
- expose raw source content too broadly
- make Telegram a generic unrestricted bot surface in v1

---

## 27. Open questions

- Which connector should be strongest after Telegram for first real customer value?
- How much source discovery should the platform support versus explicit manual mapping?
- Should some connectors support sampling or preview before full activation?
- Which parsing failures deserve human intervention versus silent suppression?
- How much raw artifact redaction or sanitization is needed in v1?
- Which sources should support event-driven ingestion first?
- How aggressive should reprocessing tooling be in v1?

---

## 28. Web: Source Feed setup wizard

The dashboard route `apps/web/src/app/(dash)/source-feeds/page.tsx` implements a **stepped** setup flow aligned with [SOURCE_FEED_SETUP_FLOW.md](./SOURCE_FEED_SETUP_FLOW.md): connector selection, connection config, display name / source mapping, governance (owner, domain, sensitivity, allowed jobs), ingestion mode, review/readiness, draft creation (`POST /source-feeds`), preview (`POST /source-feeds/:id/preview`), and activate (`POST /source-feeds/:id/activate`). Client-side readiness blocks creation when required fields are missing; API activation still enforces `CanActivate` (owner, domain, non-empty allowed jobs, ingestion mode).

---

## 29. Final ingestion stance

The ingestion layer should behave like controlled infrastructure, not like opportunistic scraping.

For every important piece of ingested knowledge, the system should be able to answer:
- where it came from
- who connected it
- what source it belongs to
- what policy governs it
- what jobs are allowed to use it
- what raw evidence supports it
- what processing steps transformed it

If the platform cannot answer those questions, the ingestion model is not strong enough.