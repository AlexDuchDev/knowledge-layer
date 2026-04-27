# AGENTS.md

## Mission

Build an Organizational Memory & Knowledge Operations Platform for controlled ingestion, governed knowledge objects, permission-aware retrieval, and AI-assisted synthesis.

The system is not a generic chat-with-docs tool.
The system is not a document dump.
The system is not a collaboration workspace replacement.

The system is:
- organizational memory infrastructure
- knowledge operations engine
- governed retrieval and synthesis layer
- controlled connector-based ingestion system

## Product priorities

When making tradeoffs, optimize in this order:

1. Correct access control and policy enforcement
2. Traceability and auditability
3. Clear domain model and canonical entities
4. Operational simplicity
5. Extensibility
6. UX polish
7. Speed of adding new connectors

Never ship AI convenience at the expense of governance.

## Core product principles

- Knowledge objects matter more than files
- Raw context may be stored, but source of truth is defined separately
- AI is not an authority layer
- Access control must be enforced before retrieval
- Materials inherit rules by default
- Manual overrides are exceptions, not the default model
- Start with a small canonical model and expand carefully
- Prefer adult control surfaces over flashy AI behavior

## Delivery principles

- Build foundation first
- Favor modular monolith boundaries over premature services
- Keep domain seams explicit
- Write docs as code evolves
- Every important feature should be testable and explainable
- Prefer boring, reliable infrastructure choices
- Do not introduce hidden magic behavior

## Working style for agents

For any non-trivial task:
1. Read the relevant product and architecture docs first
2. Write or update a short plan
3. State assumptions explicitly
4. Make the smallest correct change
5. Validate with tests, typecheck, and lint where applicable
6. Update documentation if behavior or architecture changed

Do not make large speculative refactors unless asked.

## Scope discipline

Before implementing, classify the request into one of these buckets:
- Foundation
- Governance
- Ingestion
- Knowledge Core
- Retrieval & AI
- Operations
- UX / Admin

If the task cuts across multiple buckets, call that out and reduce the implementation to the smallest safe increment.

## Source of truth policy

When creating or modifying entities, treat them as one of:
- Canonical in platform
- Mirrored authority from external system
- Derived artifact requiring review

Do not blur these categories in code or UI.

## Access and AI rules

- Never retrieve unauthorized data and then filter later
- Retrieval must run only within allowed scope
- LLM context must contain only permitted material
- Answers must carry supporting citations or linked entities
- Critical generated outputs must not auto-publish without review
- Preserve answer trace and provenance

## Documentation contract

**Procedure:** Use [docs/DOCS_MAINTENANCE_POLICY.md](docs/DOCS_MAINTENANCE_POLICY.md) (rules, reporting) and [docs/DOCS_IMPACT_MAP.md](docs/DOCS_IMPACT_MAP.md) (path-oriented file mapping). End tasks with the completion report in [docs/templates/TASK_AND_PR_DOC_IMPACT.md](docs/templates/TASK_AND_PR_DOC_IMPACT.md). UI copy and deploy/runtime changes are in scope, not only the list below.

**Quick reference —** if you change behavior in any of these areas, update the corresponding doc:
- domain model -> DOMAIN_MODEL.md
- access logic -> ACCESS_MODEL.md
- ingestion behavior -> INGESTION_AND_CONNECTORS.md
- jobs or triggers -> KNOWLEDGE_JOBS.md
- retrieval / AI behavior -> AI_RETRIEVAL_GOVERNANCE.md
- architecture or module boundaries -> ARCHITECTURE.md / MODULE_BOUNDARIES.md

## Coding expectations

- Prefer explicit types
- Prefer clear names over short names
- Avoid clever abstractions early
- Keep modules cohesive
- Keep side effects visible
- Add tests for policy logic, inheritance logic, and retrieval scoping
- Add audit events for sensitive operations
- Fail closed on permission checks

## What to avoid

Do not:
- add feature scope that weakens governance
- add per-object manual permission workflows as the default path
- create AI features that imply authority without provenance
- hide critical business rules in prompts only
- merge ingestion, canonicalization, and publication into one opaque step
- over-model the ontology too early
- introduce microservices without a concrete bottleneck

## Preferred implementation sequence

- Identity & Access foundation
- Knowledge Core foundation
- Source feeds and Telegram ingestion mode
- Audit trail
- Knowledge jobs
- Retrieval layer
- AI synthesis with citations
- Governance hardening
- Connector expansion

## Definition of done

A task is done only when:
- the behavior matches the product intent
- tests cover critical logic
- docs are updated
- access implications are considered
- audit implications are considered
- the change is understandable by the next engineer and the next agent