# ADR-0006: Raw Artifacts Must Be Preserved

## Status

Accepted

## Context

The platform ingests content from external systems such as:
- Telegram
- Slack
- Email
- meeting tools
- Jira / Trello
- Notion / Google Docs

This content is parsed, normalized, indexed, and sometimes transformed into governed knowledge objects or AI-assisted outputs.

A tempting shortcut would be to:
- keep only normalized records
- keep only extracted entities
- discard raw source material after parsing
- treat search indexes or derived objects as sufficient long-term evidence

That would weaken the system materially.

Without preserved raw artifacts:
- provenance becomes fragile
- reprocessing becomes expensive or impossible
- parsing improvements cannot be applied safely
- auditability weakens
- review quality drops because source evidence is missing
- AI outputs become harder to validate
- source disputes become harder to resolve

## Decision

The system will preserve raw artifacts for ingested source material.

Raw artifacts must be stored separately from:
- normalized records
- canonical entities
- search indexes
- embeddings
- generated outputs

Raw artifacts are first-class evidence objects, but they are not themselves canonical knowledge objects.

## Meaning

Raw artifacts should preserve, where applicable:
- source feed reference
- external artifact reference
- original payload or durable storage pointer
- source timestamps
- content hash
- source author or participant metadata
- ingestion run reference
- relevant source metadata

## Consequences

### Positive

- stronger provenance
- support for reprocessing
- better auditability
- safer review workflows
- better debugging of ingestion problems
- easier future parser upgrades
- clearer evidence base for generated artifacts

### Negative

- higher storage cost
- retention policy becomes important
- more sensitive raw content requires stronger access controls
- operational tooling must account for raw artifact lifecycle

## Guardrails

- raw artifacts must not be confused with canonical knowledge objects
- raw artifact access should be separately permissioned
- normalized records should reference raw artifacts
- generated artifacts should preserve provenance links back to raw artifacts where relevant
- retention handling must be explicit and policy-aware
- search indexes and embeddings must remain rebuildable from durable sources where possible

## Revisit triggers

Revisit only if:
- source-specific retention constraints require alternative handling
- storage model changes materially
- some source category cannot legally or operationally preserve raw payloads in full

Even then, the principle of preserving sufficient source evidence should remain.