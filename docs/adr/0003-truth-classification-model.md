# ADR-0003: Use an Explicit Truth Classification Model

## Status

Accepted

## Context

The platform ingests knowledge from many systems and also generates derived outputs through jobs and AI workflows.

Without an explicit truth model, users and the system will confuse:
- imported source snapshots
- platform-authored canonical artifacts
- AI-generated summaries
- mirrored external state
- reviewed versus unreviewed material

This creates major trust problems:
- derived artifacts may look authoritative
- external source mirrors may be mistaken for platform truth
- governance rules become inconsistent
- retrieval may rank low-authority materials too aggressively

## Decision

Every important entity must carry an explicit truth classification.

Allowed values:
- `canonical_in_platform`
- `mirrored_authority`
- `derived_artifact`

## Meaning

### Canonical in platform
The platform is authoritative for current state.

Typical examples:
- SOP
- Policy
- Team Handbook

### Mirrored authority
The source of truth lives in an external system.

Typical examples:
- Jira-driven project state
- Trello-driven initiative state
- mirrored reference docs

### Derived artifact
The object is synthesized or extracted from source inputs and may require review.

Typical examples:
- weekly digest
- extracted decision candidate
- planning summary
- synthesized meeting artifact

## Consequences

### Positive

- stronger trust semantics
- clearer workflow and publication rules
- better retrieval ranking behavior
- better UX clarity
- easier provenance interpretation

### Negative

- additional modeling and UI complexity
- more care needed when creating new entity types
- some borderline objects will require explicit product judgment

## Guardrails

- truth mode must be visible in UX
- derived artifacts should not look canonical by default
- mirrored objects should preserve external references
- jobs should default outputs to `derived_artifact` unless a stronger governed transition exists
- truth mode changes should be auditable

## Revisit triggers

Revisit if:
- entity model changes significantly
- a new trust mode is clearly needed and justified

Do not weaken the explicit classification requirement.