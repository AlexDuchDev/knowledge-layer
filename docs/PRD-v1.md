# PRD-v1.md

## Title

Organizational Memory & Knowledge Operations Platform — Version 1

---

## 1. TL;DR

We are building the first version of a governed organizational memory platform that turns fragmented company context from chats, meetings, documents, and task systems into structured, traceable, permission-aware knowledge.

Version 1 focuses on a specific wedge:
**decision and operational memory for chat-heavy teams**.

The system will:
- connect controlled knowledge sources
- ingest and normalize source content
- create and maintain governed knowledge objects
- run knowledge jobs such as summaries and extraction
- support review and approval where needed
- provide permission-aware search, retrieval, and AI-assisted synthesis with citations
- preserve provenance, version history, and auditability

Version 1 is not a generic AI chatbot over all company data.
Version 1 is not a replacement for collaboration tools.
Version 1 is not a universal enterprise knowledge platform on day one.

Version 1 must prove that governed ingestion + structured memory + permission-aware retrieval creates real operational value.

---

## 2. Problem statement

In growing companies, important context is fragmented across:
- Slack
- Telegram
- Email
- meeting transcripts
- Jira / Trello / planning boards
- Notion / Google Docs
- people’s heads

As a result:
- decisions lose their rationale
- teams repeat the same questions
- managers spend too much time reconstructing context
- onboarding is slow
- knowledge becomes stale
- AI answers become risky when they lack governance, source traceability, and access control

The main problem is not storage.
The main problem is that company knowledge is fragmented, weakly governed, hard to retrieve safely, and expensive to operationalize.

---

## 3. Opportunity

There is a clear opportunity to build a system that:
- ingests knowledge from real operating tools
- turns unstructured inputs into useful knowledge objects
- preserves provenance and source history
- enforces access before retrieval
- makes AI trustworthy through scope, citations, and governance
- supports operational workflows such as summaries, extraction, review, approval, and freshness checks

If successful, the platform becomes the governed memory layer for the company.

---

## 4. Product goal

Build a first production-ready version of a governed organizational memory platform that solves the high-value problem of lost decision and operational context for teams with fragmented communication and documentation.

---

## 5. Goals

### 5.1 Business goals

- Prove that governed organizational memory solves a real and painful workflow problem
- Create a strong wedge around decision and operational memory
- Establish a foundation that can expand into a broader knowledge operations platform
- Differentiate through governance, provenance, and permission-aware retrieval
- Enable future enterprise-leaning use cases without rewriting the core foundation

### 5.2 User goals

- Find confirmed or reviewable decision context quickly
- Reconstruct what happened across chats, meetings, and work systems without manual digging
- Generate structured summaries for planning, status, and review workflows
- Trust AI-assisted answers because they include citations and respect permissions
- Understand whether knowledge is current, approved, stale, canonical, mirrored, or derived

### 5.3 System goals

- Support controlled ingestion from supported sources
- Normalize source content into raw artifacts and structured knowledge objects
- Enforce access control before retrieval and generation
- Preserve provenance, versioning, and audit trail
- Support manual, scheduled, event-driven, and window-based knowledge jobs
- Enable review and approval flows for selected artifacts

### 5.4 Non-goals

Version 1 will not:
- replace Slack, Telegram, Notion, Jira, or Google Docs
- offer unrestricted conversational access to all company data
- support deeply customized ontologies per customer
- support highly complex enterprise compliance permutations on day one
- make object-by-object manual ACL configuration the default operating model
- auto-publish critical materials without review
- optimize for every connector equally before the foundation is mature

---

## 6. Primary v1 wedge

### Decision and operational memory for chat-heavy teams

The first version should be especially strong at:
- ingesting communication-heavy inputs
- extracting decisions, blockers, commitments, and risks
- generating structured weekly and planning summaries
- preserving links back to raw context
- letting users retrieve the relevant answer within permission scope

This is the most important v1 positioning constraint.

If a feature does not improve this wedge, it should be deprioritized.

---

## 7. Target users

### 7.1 Admin / governance operator
Needs to:
- connect source feeds
- assign owners, domains, and sensitivity levels
- configure jobs and output policies
- monitor review queues, sync health, and audit trails

### 7.2 Domain owner
Needs to:
- govern a domain’s source feeds
- review generated outputs
- confirm important decisions or artifacts
- maintain freshness and ownership

### 7.3 Team lead / PM / ops lead / operator
Needs to:
- run summaries and extractions
- inspect outputs
- confirm or refine important artifacts
- retrieve context for planning, reporting, and decision follow-through

### 7.4 Knowledge consumer
Needs to:
- search for process, policy, meeting, project, or decision context
- ask questions and get permission-aware answers
- inspect citations, provenance, and status
- understand what is trusted and what still needs review

---

## 8. User stories

### 8.1 Source setup

As an admin, I want to connect a controlled source feed and assign owner, domain, sensitivity, and allowed jobs, so that ingestion follows governance rules from the start.

### 8.2 Controlled ingestion

As a domain owner, I want source content to be ingested only from explicitly connected sources, so that the platform does not become an uncontrolled surveillance or data sprawl tool.

### 8.3 Knowledge job execution

As an operator, I want to run a weekly summary or extraction job on an approved source scope, so that I can turn fragmented updates into a usable artifact.

### 8.4 Review and approval

As a reviewer, I want to inspect generated artifacts and approve, reject, or revise them before publication, so that important outputs remain trusted.

### 8.5 Decision memory

As a team lead, I want important decisions to be captured with rationale, source references, owner, and status, so that future work does not depend on memory or chat archaeology.

### 8.6 Permission-aware Q&A

As a knowledge consumer, I want to ask a question and receive only the knowledge I am allowed to access, with citations and status indicators, so that I can act with confidence.

### 8.7 Auditability

As an admin, I want to see what was ingested, generated, reviewed, published, or accessed, so that the system is governable and explainable.

### 8.8 Freshness and lifecycle

As a domain owner, I want artifacts to show freshness and review status, so that stale or superseded content does not silently masquerade as current truth.

---

## 9. Scope

### 9.1 Included sources in v1

- Telegram ingestion mode
- Slack
- Email
- Fireflies
- Granola
- Jira
- Trello
- Notion
- Google Drive / Docs

### 9.2 Included knowledge domains in v1

- Decisions
- Projects
- Processes / SOP
- Policies
- Meetings
- Insights

### 9.3 Included system capabilities in v1

- connector framework
- source feed configuration
- source ownership and policy assignment
- controlled ingestion
- raw artifact storage
- normalization pipeline
- knowledge objects
- entity links
- version history
- audit trail
- knowledge jobs engine
- output policy configuration
- review requirement support
- hybrid retrieval
- permission-aware retrieval
- AI-assisted synthesis with citations
- admin and governance UI

### 9.4 Telegram constraint in v1

Telegram is ingestion-only in v1.

It is supported only as:
- explicitly connected chat sources
- with assigned owner
- with assigned domain
- with sensitivity policy
- with defined allowed jobs
- with controlled ingestion mode

Telegram is not supported in v1 as:
- a universal output channel
- a free-form AI bot interface
- an unrestricted chat assistant entry point

---

## 10. Functional requirements

### 10.1 Source management

The system must allow admins to:
- create and manage source feeds
- select connector type
- assign source owner
- assign domain
- assign sensitivity level
- define allowed jobs
- define ingestion mode
- monitor sync state and basic health

### 10.2 Ingestion

The system must:
- ingest content from supported sources
- support full import, incremental sync, and event-driven ingestion where applicable
- preserve raw artifacts and source metadata
- parse and normalize source content
- deduplicate repeated inputs where possible
- link normalized artifacts to source feed and provenance metadata

### 10.3 Knowledge objects

The system must:
- support canonical entity types for v1
- create, store, update, and version entities
- preserve entity provenance
- support explicit relations between entities
- classify each entity as canonical in platform, mirrored authority, or derived artifact

### 10.4 Knowledge jobs

The system must:
- support manual, scheduled, event-driven, and window-based jobs
- allow admins or allowed operators to define source scope
- allow output policy configuration
- allow review requirements
- store execution logs and outputs
- support summarization, extraction, consolidation, monitoring, transformation, and publishing job types

### 10.5 Review and approval

The system must:
- support review tasks for selected outputs
- support approval flows for selected entity types
- expose owner, reviewer, due dates, and status
- prevent critical outputs from auto-publishing when review is required

### 10.6 Search and retrieval

The system must support:
- keyword search
- semantic retrieval
- hybrid retrieval
- filtered retrieval
- permission-aware retrieval
- relation-aware retrieval
- freshness-aware retrieval

### 10.7 AI-assisted synthesis

The system must support:
- summarization
- extraction
- decision extraction
- action item extraction
- suggested links
- duplicate detection
- stale detection
- scoped Q&A with citations

### 10.8 Access control

The system must:
- support domain-based access
- support entity-type access
- support object-level access
- support action-level permissions
- support inheritance and exception rules
- enforce access before retrieval
- prevent LLM context from including disallowed material

### 10.9 Auditability

The system must:
- log sensitive operations
- preserve answer trace for AI-generated answers
- capture provenance for entity creation and updates
- expose audit information in admin/governance surfaces

---

## 11. Non-functional requirements

The system must be:
- governable
- secure
- traceable
- extensible
- resilient enough for operational use
- suitable for enterprise-leaning scenarios
- expandable without rewriting the foundation

Additional quality requirements:
- fail closed on permissions
- preserve explainability
- keep module boundaries explicit
- support background processing for ingestion and jobs
- provide acceptable retrieval performance for operational use
- maintain consistent provenance and audit semantics across modules

---

## 12. Source-of-truth policy

Every entity must be classified as one of:

### 12.1 Canonical in platform
The platform is the source of truth for current state.

Likely examples:
- SOP
- Policy
- Team Handbook
- Role Handbook

### 12.2 Mirrored authority
The authoritative state lives in an external system.

Likely examples:
- Jira-driven project status
- Trello initiative state
- externally owned reference docs

### 12.3 Derived artifact
The object is synthesized from source inputs and requires review or confirmation before being treated as authoritative.

Likely examples:
- weekly digest
- decision candidate extracted from chat
- planning summary
- synthesized status report

This classification must affect:
- UX labels
- lifecycle behavior
- publication rules
- trust messaging
- downstream workflows

---

## 13. Key canonical entity types for v1

- Decision
- Project
- Initiative
- SOP
- Process
- Policy
- Meeting
- Incident
- Experiment
- Insight
- Customer Insight
- Role Handbook
- Team Handbook
- Template
- Reference Document

### 13.1 Decision as a priority entity

Decision is a strategically important entity in v1 and should receive above-average product and design attention.

Minimum fields for Decision:
- id
- title
- summary
- status
- owner
- domain
- sensitivity_level
- rationale
- alternatives_considered
- related_project
- source references
- confirmed_by
- confirmation_method
- created_at
- updated_at
- review_due_at
- provenance metadata

---

## 14. User experience

### 14.1 Admin flow: connect a source feed

1. Admin opens Source Feeds UI
2. Admin selects connector type
3. Admin authenticates connector
4. Admin maps the source
5. Admin assigns:
   - owner
   - domain
   - sensitivity
   - allowed jobs
   - ingestion mode
6. Admin saves the source feed
7. System validates configuration
8. System performs initial sync or schedules ingestion
9. Admin sees sync status, health, and recent activity

Expected result:
The source is governable before it becomes operational.

### 14.2 Operator flow: run a knowledge job

1. Operator opens Knowledge Jobs UI
2. Operator selects existing job or creates new one
3. Operator selects source scope from allowed feeds
4. Operator defines trigger or runs manually
5. Operator reviews output policy and review requirement
6. System runs job
7. System stores execution logs and artifacts
8. Generated output becomes:
   - a draft artifact
   - a review task
   - or a published non-critical artifact depending on policy

Expected result:
Unstructured operational inputs become structured, governed artifacts.

### 14.3 Reviewer flow: review a generated artifact

1. Reviewer opens review queue
2. Reviewer opens generated artifact
3. Reviewer sees:
   - content
   - source citations
   - provenance
   - owner
   - status
   - related objects
4. Reviewer approves, requests changes, or rejects
5. System updates lifecycle state and audit trail

Expected result:
Derived outputs become trustworthy through explicit review.

### 14.4 Knowledge consumer flow: ask a question

1. User opens search or Q&A
2. User asks a question or applies filters
3. System resolves identity and allowed scope
4. Retrieval runs only over allowed objects
5. AI receives only permitted context
6. System returns answer with:
   - citations
   - supporting entities
   - status indicators
   - owner where applicable
   - freshness and trust indicators
7. User opens canonical entity if needed

Expected result:
The user gets a useful answer without violating access or trust boundaries.

### 14.5 Failure / trust UX requirements

The product must clearly communicate when:
- an answer may be partial because of restricted access
- an artifact is stale
- an artifact is derived and not yet approved
- an object is superseded
- a retrieval result is mirrored from an external source
- a job failed or produced low-confidence output

The UI should never imply certainty where certainty is not justified.

---

## 15. Governance model in v1

### 15.1 Ownership

Every important source feed, job, and governed artifact must have an owner.

### 15.2 Review

Selected artifacts and outputs must support:
- assigned reviewer
- due date
- status
- change request outcome
- approval or rejection path

### 15.3 Freshness

The system must support freshness metadata and stale detection for relevant entity types.

### 15.4 Lifecycle

Representative lifecycle states:
- Draft
- In Review
- Approved
- Active
- Stale
- Archived
- Proposed
- Confirmed
- Superseded

### 15.5 Audit

Sensitive events must be logged, including:
- source connection changes
- policy changes
- job runs
- review actions
- approvals
- AI answer traces
- permission-sensitive retrieval events where appropriate

---

## 16. Access and permissions

### 16.1 Access layers

The system must support:
- domain-based access
- entity-type access
- object-level access
- action-level permissions
- inheritance
- exceptions

### 16.2 Access resolution sequence

At request time, the system should evaluate:
1. identity
2. global deny rules
3. object-level ACL
4. domain access
5. entity-type rules
6. action permission
7. sensitivity level

### 16.3 AI access rule

AI must:
1. identify the user
2. resolve the user’s allowed scope
3. retrieve only allowed materials
4. pass only allowed materials to the model
5. store answer trace
6. return citations and supporting references

This is a hard requirement, not a best-effort behavior.

---

## 17. Success metrics

### 17.1 Business / product outcomes

- reduction in time spent reconstructing decision context
- reduction in repeated operational questions
- increase in trusted use of internal retrieval
- adoption of governed summaries and extracted artifacts
- expansion from initial wedge into adjacent domains without foundation rewrite

### 17.2 Operational product metrics

- number of connected governed source feeds
- number of active source owners
- number of knowledge jobs run per week
- percentage of successful job runs
- percentage of outputs reviewed before publication where required
- number of confirmed decision objects created per week
- weekly active users of search / Q&A
- click-through rate from answer to supporting entity

### 17.3 Trust and governance metrics

- percentage of AI answers with valid citations
- zero unauthorized retrieval incidents
- percentage of entities with owner assigned
- percentage of stale critical artifacts flagged
- review SLA completion rate
- audit event coverage for sensitive operations

### 17.4 Quality metrics

- extraction precision for decisions
- extraction precision for action items and risks
- duplicate detection quality
- retrieval relevance quality
- user-rated answer usefulness
- review rejection rate for generated outputs

---

## 18. Milestones and sequencing

### Milestone 1 — Foundation
- users, teams, roles, domains
- access model foundation
- source feeds
- entities CRUD
- basic provenance
- audit log
- Telegram ingestion mode
- basic search

### Milestone 2 — Knowledge operations
- knowledge jobs
- triggers
- execution runs
- output policies
- review requirements
- artifact routing

### Milestone 3 — Retrieval and AI
- chunking
- embeddings
- hybrid retrieval
- permission-aware Q&A
- summarization and extraction
- answer citations and trace

### Milestone 4 — Governance hardening
- lifecycle states
- approvals
- freshness rules
- review center
- policy exceptions
- stronger trust indicators in UX

### Milestone 5 — Connector expansion
- Slack
- Email
- Fireflies
- Granola
- Jira
- Trello
- Notion
- Google Drive / Docs

Important sequencing rule:
Do not expand connectors aggressively before governance, provenance, and permission-aware retrieval are reliable.

---

## 19. Technical considerations

### 19.1 Proposed stack

Frontend:
- Next.js
- React
- TypeScript
- Tailwind
- TanStack Query

Backend:
- Go
- modular monolith
- background workers
- webhook processors

Data:
- PostgreSQL
- pgvector
- Redis
- S3-compatible storage
- OpenSearch

### 19.2 Technical product principles

- modular monolith first
- explicit domain boundaries
- asynchronous processing for ingestion and jobs
- preserve raw artifacts and provenance
- separate ingestion, normalization, canonicalization, retrieval, and publication concerns
- fail closed on permissions
- keep AI orchestration visible and auditable

### 19.3 Technical risks

- access logic becomes fragmented across modules
- ingestion quality varies heavily by connector
- retrieval quality looks good in demos but weak in production
- provenance becomes inconsistent
- review workflows become bolted on instead of native
- excessive ontology complexity slows delivery

---

## 20. Open questions

- Which source should be the default first production connector after Telegram?
- Which entity types should require approval versus review-only?
- Which outputs can safely auto-publish under narrow conditions?
- How visible should restricted / partial answer messaging be in the UX?
- Which retrieval ranking strategy is sufficient for v1 quality?
- How aggressive should stale detection be for different entity types?
- What is the minimum admin configuration required for first value?
- How much editing should reviewers do inside the system versus external tools?

---

## 21. Launch criteria

Version 1 is launch-ready only when:

- a real team can connect controlled source feeds end to end
- ingestion works reliably for at least one production-like workflow
- weekly summary and decision extraction workflows produce usable artifacts
- important outputs can be reviewed before publish
- permission-aware retrieval works correctly
- answers contain citations and supporting entities
- provenance and audit trail are visible for critical actions
- admins can configure sources and jobs without engineering intervention for each change
- users can distinguish canonical, mirrored, and derived artifacts in the interface

---

## 22. Final product stance

Version 1 is successful if it proves that governed memory is meaningfully more useful and trustworthy than:
- scattered chats and notes
- static knowledge bases
- generic AI chat over company documents

The real test is not whether the product can generate summaries.
The real test is whether teams trust it enough to rely on it for operational context and decision memory.