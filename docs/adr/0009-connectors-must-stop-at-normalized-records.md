# ADR-0009: Connectors Must Stop at Normalized Records

## Status

Accepted

## Context

External connectors vary widely in structure and behavior:
- chat systems
- meeting tools
- email
- documentation systems
- task systems

Without a clear boundary, connector code can begin to absorb too much product logic, such as:
- deciding what becomes a canonical entity
- assigning publication states
- embedding domain-specific truth rules
- performing downstream business interpretation

This would create serious long-term problems:
- connector-specific logic leaking into core product semantics
- inconsistent behavior across source types
- brittle scaling of new connectors
- difficulty debugging provenance and publication behavior
- erosion of domain boundaries inside the architecture

## Decision

Connector responsibility stops at normalized records and raw artifacts.

Connectors may:
- authenticate
- map sources
- fetch content
- store raw artifacts
- parse payloads
- extract source metadata
- create normalized records
- report sync health and execution details

Connectors must not:
- directly publish canonical knowledge objects
- decide final truth mode for downstream artifacts beyond source metadata context
- bypass workflow and governance layers
- perform opaque connector-specific business interpretation that belongs in downstream modules

## Consequences

### Positive

- cleaner architecture
- stronger domain boundaries
- easier connector expansion
- more consistent provenance
- better downstream governance
- clearer debugging and reprocessing

### Negative

- downstream pipelines need stronger modeling
- less temptation-friendly “fast path” automation
- some teams may want connector-specific shortcuts that now need explicit design

## Guardrails

- normalized records must be the stable handoff boundary
- canonicalization belongs in downstream platform logic
- publication belongs in workflow/governance logic
- connector code should remain source-oriented, not business-authority-oriented
- exceptions require explicit ADR-level discussion

## Revisit triggers

Revisit only if:
- a source type clearly requires a richer ingestion boundary
- normalized records prove insufficient for important source semantics

Even then, connector code should still avoid owning truth and publication logic.