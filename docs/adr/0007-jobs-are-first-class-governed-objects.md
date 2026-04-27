# ADR-0007: Knowledge Jobs Are First-Class Governed Objects

## Status

Accepted

## Context

The platform’s value depends not only on storing and retrieving knowledge, but also on performing repeatable operations over knowledge, such as:
- weekly summaries
- decision extraction
- planning consolidation
- stale detection
- duplicate detection
- publishing approved artifacts

A weak implementation path would treat these operations as:
- hidden prompts
- ad hoc scripts
- untracked background tasks
- ephemeral automations without ownership or policy

That approach would create serious problems:
- unclear ownership
- weak observability
- poor auditability
- inconsistent permissions
- unclear publication behavior
- low trust in outputs
- poor reusability across teams and domains

## Decision

Knowledge jobs will be modeled as first-class governed objects.

A knowledge job must have:
- purpose
- owner
- type
- source scope
- trigger
- allowed operators
- output policy
- review requirement
- publication mode
- execution history

A knowledge job run must also be a first-class record with:
- run identity
- trigger identity
- input scope snapshot
- execution status
- warnings and errors
- outputs
- traceability

## Consequences

### Positive

- stronger operational clarity
- better governance over generated outputs
- reusable job patterns
- easier observability and debugging
- better permission control
- stronger provenance and audit trail
- clearer product UX for operations over knowledge

### Negative

- more upfront modeling work
- additional UI and admin complexity
- higher bar for “quick automations”

## Guardrails

- jobs must not be hidden in prompt-only configuration
- job outputs must inherit domain, sensitivity, provenance, and workflow state
- scheduled jobs must run under governed scope, not broad worker privileges
- job definitions should remain understandable without reading code
- jobs should support pause, activation, and inspection as native product behavior

## Revisit triggers

Revisit if:
- the product later introduces a separate low-stakes automation layer
- some extremely lightweight tasks do not justify full job modeling

Even then, operationally meaningful knowledge work should remain job-based.