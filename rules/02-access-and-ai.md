# Access and AI Rules

This project must remain permission-aware and trust-oriented.

## Hard access rules

- Deny by default
- Access is evaluated before retrieval
- Never retrieve broadly and filter later
- Never send disallowed context to the model
- Relation expansion must re-check access on each related object
- Cached answers or contexts must not bypass current permission checks
- Admin does not automatically mean unrestricted content visibility

## Inheritance rules

Most access should inherit from:
- domain default policy
- source feed policy
- entity type policy
- job output policy

Object-level overrides are exceptions.
Do not build new flows that depend on manual per-object ACLs by default.

## Sensitivity rules

Respect sensitivity independently from domain.
Higher sensitivity may narrow access even inside the same domain.

## AI truth rules

AI is not an authority layer.

AI outputs must:
- stay within allowed scope
- preserve citations or supporting entities where relevant
- preserve provenance
- carry owner, domain, sensitivity, workflow state, and truth mode if reusable
- default to derived artifacts unless an explicit governed transition exists

## Review rules

High-authority generated outputs require review before publication or authoritative use.

Examples:
- decision extraction outputs
- planning summaries used for execution
- SOP drafts
- policy drafts
- sensitive cross-domain summaries

## Trust UX rules

Do not hide important trust signals.
Users should be able to tell:
- canonical vs mirrored vs derived
- draft vs approved vs stale vs superseded
- partial answer due to scope limits where applicable

## Testing priorities

Always prioritize tests for:
- access resolution
- scope enforcement
- AI context scoping
- citation attachment
- review gating
- no-leak behavior