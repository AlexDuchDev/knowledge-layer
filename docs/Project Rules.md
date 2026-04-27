# Project Rules

## Commands

- `make dev` — run local development environment
- `make test` — run all tests
- `make lint` — run linting
- `make typecheck` — run frontend type checks
- `make test-backend` — run backend tests
- `make test-frontend` — run frontend tests

If these commands do not exist yet, create them before adding more workflow complexity.

## Repo expectations

- Frontend: `apps/web`
- Backend API: `apps/api`
- Workers: `apps/workers`
- Shared contracts/types: `packages/shared`
- Docs: `docs`
- ADRs: `docs/adr`

Prefer this structure unless a better repo shape is explicitly documented.

## Workflow

For any feature larger than a tiny fix:
- start with a short plan
- read the relevant docs in `docs/`
- implement in small steps
- run the narrowest useful test first
- then run broader validation
- update docs if behavior changed

If the conversation becomes noisy or spans multiple unrelated tasks, start a fresh thread and reference prior work.

## Product guardrails

This product is a governed organizational memory platform.

Always preserve:
- permission-aware retrieval
- source-of-truth distinction
- provenance
- review / approval controls
- auditability

Never optimize for convenience by weakening access control or governance.

## Architecture guardrails

- Prefer modular monolith boundaries
- Keep domain modules explicit
- Avoid premature extraction to services
- Keep ingestion, canonicalization, retrieval, and publication separable
- Use asynchronous workers for long-running ingestion and knowledge jobs
- Fail closed on policy evaluation

## Data and domain guardrails

Canonical entity types initially include:
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

Do not introduce many new entity types without updating domain docs and explaining why.

## Access model guardrails

Enforce access in this order:
1. identity
2. global deny rules
3. object-level ACL
4. domain access
5. entity-type rules
6. action permission
7. sensitivity level

Do not fetch outside allowed scope.
Do not send disallowed context to AI.

## AI behavior

AI is an assistant layer, not an authority layer.

AI outputs must:
- be traceable
- include supporting citations or linked entities
- respect retrieval scope
- avoid automatic publication for critical materials without review

## Testing priorities

Write or extend tests first for:
- permission resolution
- inheritance rules
- source feed policy propagation
- job output policy behavior
- retrieval scoping
- audit event generation

## Documentation pointers

Use these files as the primary project context:
- `docs/PRODUCT.md`
- `docs/PRD-v1.md`
- `docs/ARCHITECTURE.md`
- `docs/DOMAIN_MODEL.md`
- `docs/ACCESS_MODEL.md`
- `docs/INGESTION_AND_CONNECTORS.md`
- `docs/KNOWLEDGE_JOBS.md`
- `docs/AI_RETRIEVAL_GOVERNANCE.md`

Keep rules short. Put detailed product and technical decisions in docs, not here.