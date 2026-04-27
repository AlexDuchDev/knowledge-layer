# IMPLEMENTATION_PLAN.md

## 1. Purpose

This document defines the practical implementation plan for v1 of the Organizational Memory & Knowledge Operations Platform.

It translates the product and architecture documents into:
- build order
- milestones
- dependencies
- delivery slices
- risks
- validation checkpoints

This is not a roadmap for marketing.
It is the execution plan for building the first usable version correctly.

---

## 2. Implementation strategy

We will build the system in vertical slices, but with strong foundation-first discipline.

The build order should optimize for:
1. access correctness
2. domain clarity
3. provenance and auditability
4. operational reliability
5. retrieval usefulness
6. AI usefulness
7. connector expansion

We should not optimize first for:
- flashy AI demos
- broad connector coverage
- highly customized workflows
- complex UI polish before core control surfaces are stable

---

## 3. Execution principles

### 3.1 Foundation before convenience
Do not add “magic” behavior before permissions, provenance, and governance are solid.

### 3.2 Narrow wedge first
The first product win is:
**decision and operational memory for chat-heavy teams**

### 3.3 Build with explainability
Every major flow should be understandable in logs, UI, and data model.

### 3.4 Prefer controlled defaults
Default to:
- derived artifacts
- review-required on important generated outputs
- inheritance over manual ACL
- explicit ownership over loose collaboration

### 3.5 Keep modules boring
Do not over-engineer abstractions before they carry real load.

---

## 4. Recommended build phases

## Phase 1 — Repo and platform foundation

### Goal
Create the base repo shape, runtime boundaries, and engineering loop.

### Deliverables
- monorepo layout or repo layout established
- `apps/web`
- `apps/api`
- `apps/workers`
- `packages/shared`
- docs and ADR folders wired in
- local dev commands
- CI basics
- linting
- formatting
- type checking for frontend
- backend test harness
- migrations framework
- env/config management

### Why first
Without this, every later step will be slower and more fragile.

### Exit criteria
- repo boots locally
- CI runs basic checks
- migrations can be created and applied
- frontend and backend apps compile
- workers can run basic jobs

---

## Phase 2 — Identity & Access foundation

### Goal
Implement the minimum control plane for users, teams, roles, domains, grants, and access evaluation.

### Deliverables
- users
- teams
- roles
- role bindings
- domains
- domain grants
- access policies
- policy overrides
- action permissions
- sensitivity levels
- access decision service
- structured access decision output
- tests for deny-by-default and inheritance basics

### Why now
Everything else depends on correct scoping.

### Exit criteria
- users can be assigned roles and domain access
- access decision service returns explainable decisions
- deny-by-default is enforced
- sensitivity checks exist
- overrides work in a controlled way
- tests cover core access paths

---

## Phase 3 — Knowledge Core foundation

### Goal
Implement the canonical entity foundation and supporting metadata model.

### Deliverables
- entity base model
- entity types for v1
- entity metadata envelope
- truth mode support
- lifecycle state support
- owner/domain/policy linkage
- provenance records
- versioning foundation
- explicit entity relations
- basic CRUD APIs for core entities

### Priority entity types
Start with:
- Decision
- Project
- Meeting
- SOP
- Policy
- Insight
- ReferenceDocument

### Exit criteria
- entities can be created and queried
- provenance can be attached
- relations can be created
- version snapshots work
- truth mode is visible and persisted

---

## Phase 4 — Source feeds and ingestion foundation

### Goal
Build the controlled entry point for external knowledge.

### Deliverables
- connector registry model
- source feed model
- source feed admin flows
- source feed validation
- ingestion run model
- raw artifact storage
- normalized record model
- source health status
- pause/resume support
- initial connector abstraction

### First connector
Telegram ingestion mode

### Why Telegram first
It is part of the wedge and creates a sharp governed-ingestion use case.

### Exit criteria
- admin can configure Telegram source feed
- required governance fields are enforced
- ingestion run stores raw artifacts
- normalized records are produced
- source feed health is visible
- source feed policy is attached

---

## Phase 5 — Audit and observability foundation

### Goal
Make the system inspectable before complexity grows.

### Deliverables
- audit event model
- audit event emission for sensitive actions
- ingestion run logs
- job run logs foundation
- basic system health metrics
- admin-facing audit list
- trace references for important flows

### Exit criteria
- access-related changes produce audit events
- source feed changes produce audit events
- entity creation/update path can emit traceable events
- basic operational visibility exists

---

## Phase 6 — Knowledge jobs engine

### Goal
Turn the platform into an operational system over knowledge.

### Deliverables
- job definitions
- job triggers
- job runs
- source scope resolution
- execution pipeline
- output routing
- review-required publication mode
- example job templates
- operator permissions for jobs

### First jobs to build
- weekly digest
- planning summary
- decision extraction

### Exit criteria
- a job can be defined and run manually
- job run trace exists
- outputs receive owner/domain/sensitivity/truth mode/provenance
- outputs can enter review queue
- operators are permission-checked

---

## Phase 7 — Workflow & Governance foundation

### Goal
Implement review, approval, lifecycle, and freshness behavior.

### Deliverables
- review tasks
- basic approval flow support
- lifecycle transitions
- freshness rules
- stale detection foundation
- governance queues
- reviewer assignment model

### Exit criteria
- review tasks can be created and completed
- selected entity types can require review
- lifecycle state transitions are enforced
- stale status can be computed and surfaced

---

## Phase 8 — Search and retrieval foundation

### Goal
Make the knowledge layer findable and useful.

### Deliverables
- indexing pipeline
- chunking
- keyword search
- filtered search
- semantic retrieval foundation
- hybrid retrieval
- relation-aware retrieval foundation
- permission-aware retrieval scoping
- search result trust metadata

### Exit criteria
- users can search core entity types
- results are permission-scoped
- trust indicators appear in results
- relation expansion respects access rules

---

## Phase 9 — AI-assisted synthesis

### Goal
Add bounded AI value on top of governed retrieval.

### Deliverables
- summarization task pipeline
- extraction task pipeline
- scoped Q&A
- citation model
- supporting entity model in answers
- answer trace model
- derived artifact routing rules
- review-required behavior for important outputs

### Exit criteria
- answers only use allowed context
- answers include citations
- answer traces are stored
- generated artifacts default to derived
- high-authority outputs can route to review

---

## Phase 10 — Governance hardening

### Goal
Improve trust, maturity, and operational discipline.

### Deliverables
- policy exception center
- stale-content center
- approval queues
- stronger audit surfaces
- stronger trust badges
- partial-view messaging
- source-of-truth visibility improvements
- missing-owner and missing-policy monitoring

### Exit criteria
- governance center is usable
- stale and exception states are visible
- admins can explain policy source and trust state
- review backlog and policy risks are visible

---

## Phase 11 — Connector expansion

### Goal
Expand ingestion breadth only after core control systems are stable.

### Connector order suggestion
1. Slack
2. Fireflies / Granola
3. Email
4. Jira
5. Notion
6. Google Drive / Docs
7. Trello

### Why this order
It follows the wedge:
- communication and meeting context first
- planning and task-state context second
- broader document coverage after foundation is stable

### Exit criteria
- each connector supports governed source feed setup
- raw artifact preservation works
- normalization output is stable
- connector health is observable
- downstream policy propagation is correct

---

## 5. Suggested first end-to-end vertical slice

Build this first real use case:

### Slice: Weekly governed digest from Telegram

Flow:
1. admin creates Telegram source feed
2. source feed has owner/domain/sensitivity/allowed jobs
3. ingestion stores raw artifacts and normalized message records
4. operator runs weekly digest job
5. system summarizes progress/blockers/risks/decisions
6. output becomes derived artifact
7. review task is created
8. reviewer approves or requests changes
9. approved digest becomes retrievable
10. user can ask scoped question and get cited answer from it and linked sources

### Why this slice is strong
It exercises:
- access
- source feeds
- ingestion
- provenance
- jobs
- review
- retrieval
- AI
- trust indicators

This is a real product slice, not just a demo.

---

## 6. Dependency map

### Must exist early
- users / roles / domains
- access evaluation
- entity metadata envelope
- source feeds
- raw artifact storage
- audit events

### Depends on source feeds
- ingestion
- source policy inheritance
- job source scopes

### Depends on entities
- retrieval
- review flows
- lifecycle
- citations to governed objects

### Depends on retrieval
- scoped Q&A
- good AI summaries with evidence
- relation-aware context reconstruction

### Depends on workflow
- trustworthy publication
- high-authority AI output safety
- lifecycle-aware trust UX

---

## 7. Team sequencing guidance

If one small team is building this:
- do not over-parallelize too early
- finish control-plane foundation before spreading thin
- prioritize one end-to-end slice over many half-built subsystems

If multiple engineers are available:
- one owns access + identity
- one owns knowledge core + governance
- one owns ingestion + workers
- one owns frontend admin/review/search surfaces
- one supports retrieval + AI once foundation is ready

---

## 8. Risks during implementation

### 8.1 Premature connector sprawl
Risk:
too many connectors before policy and provenance are solid.

Response:
freeze connector expansion until the first end-to-end slice works well.

### 8.2 Weak access modeling
Risk:
retrofit permissions later.

Response:
build access decision service before retrieval and AI.

### 8.3 Overbuilding ontology
Risk:
too many entity types too early.

Response:
start with a smaller core set and expand carefully.

### 8.4 Hidden AI logic
Risk:
business logic drifts into prompt text.

Response:
keep output routing, policy, truth mode, and review logic in explicit code.

### 8.5 Review fatigue
Risk:
too many weak outputs flood reviewers.

Response:
be selective with early jobs and keep outputs high-signal.

---

## 9. Validation checkpoints

After each phase, ask:

### Foundation phases
- is the system more controllable, or just more complex?

### Entity and ingestion phases
- can we explain where every important object came from?

### Job and workflow phases
- can a human inspect and govern what was generated?

### Retrieval and AI phases
- can we explain why this answer was shown and why it is trustworthy?

### Expansion phases
- are we adding breadth without weakening the foundation?

---

## 10. Definition of implementation success

Implementation is successful when:
- one real team can use the first end-to-end slice reliably
- permissions work correctly
- generated outputs are governable
- provenance is inspectable
- retrieval is useful and trust-aware
- AI is helpful without becoming unsafe
- admins can operate the system without engineering intervention for every change

---

## 11. Final implementation stance

We are not building a demo-first AI product.
We are building operational knowledge infrastructure.

That means the right implementation order is:
- control
- traceability
- structure
- workflows
- retrieval
- AI
- scale

If the order flips, the system will look impressive before it becomes trustworthy.