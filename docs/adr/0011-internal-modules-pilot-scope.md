# ADR-0011: Scope of the `internal/modules` hexagonal pilot

## Status

Accepted

## Context

The API is a **modular monolith** ([ADR-0001](./0001-modular-monolith.md)) with bounded contexts under `apps/api/internal/<context>/`. A parallel layout exists under `apps/api/internal/modules/` (e.g. `audit_ops` with extracted transport), described as a **hexagonal-style pilot** in [MODULE_BOUNDARIES.md](../MODULE_BOUNDARIES.md).

That duality creates risk:

- contributors are unsure whether new domains belong in `internal/` or `internal/modules/`
- duplicated patterns (two “right ways”) increase review friction and import-graph surprises

## Decision

1. **Default:** New bounded-context code continues under `apps/api/internal/<context>/` following [MODULE_BOUNDARIES.md](../MODULE_BOUNDARIES.md) (transport → application → domain → repository).
2. **Pilot scope:** `internal/modules/*` is **only** for experiments already started there (today: **audit_ops** transport extraction and closely related seams). Do **not** add new top-level domains under `modules/` without a **new ADR** that revisits this decision.
3. **Expansion trigger:** A second context may move to `internal/modules/` only if there is a concrete need (e.g. shared transport generation, strict port/adapter testing) that cannot be satisfied cleanly under flat `internal/*` — document the trigger and import rules in a follow-up ADR.
4. **Consolidation option:** If the pilot stops delivering value, fold `modules/audit_ops` back into `internal/audit` (or equivalent) in one migration PR rather than leaving both trees indefinitely.

## Consequences

- **Positive:** Clear default path for most PRs; pilot remains visible and contained.
- **Negative:** Two folder conventions remain until consolidation or a deliberate second pilot; MODULE_BOUNDARIES must stay explicit about allowed edges.

## Related

- [MODULE_BOUNDARIES.md](../MODULE_BOUNDARIES.md) §1 (audit_ops pilot), §9
- [backend-architecture.md](../backend-architecture.md) if it references `modules/*`
