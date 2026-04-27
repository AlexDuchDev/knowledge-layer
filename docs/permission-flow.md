# Permission flow

How authorization is enforced in the Knowledge Layer API and how retrieval/AI stay within scope.

**Canonical model:** [permission-system.md](./permission-system.md).

## Central resolver

- **Implementation:** `identity_access.AccessEvaluator` ([`access.go`](../apps/api/internal/identity_access/access.go)).
- **Platform façade:** `platform/permissions.Resolver` ([`permissions/resolver.go`](../apps/api/internal/platform/permissions/resolver.go)) — use this from new code so grant/domain logic stays in one place.

## Evaluation inputs

- **Principal:** authenticated user UUID (session or dev header).
- **Action:** e.g. `view`, `edit`, `search`, `run_job`, `manage_source_feed`, `view_raw`.
- **Resource:** type + optional id (entity, domain, source_feed, etc.).
- **Domain scope:** required for most checks; ties to `domain_grants` and `access_level`.
- **Sensitivity:** compared to grant `sensitivity_cap` and domain defaults.
- **Entity type:** for entities, matched against `access_policies.entity_type_scope` when set.
- **Overrides:** `policy_overrides` may allow or deny early.
- **entity_acl:** explicit **deny** rows for user/team principals on an entity.

## Where checks apply

| Area | Mechanism |
|------|-----------|
| Entity read/write | `requireEntityAction` / `entityViewOK` in [`routes_helpers.go`](../apps/api/internal/httpserver/routes_helpers.go) calling `AccessEvaluator.Evaluate` |
| Source feeds | [`source_feed_access.go`](../apps/api/internal/httpserver/source_feed_access.go) |
| Knowledge jobs | Owner or `knowledge_job_operators`; routes use job service + identity gates |
| Search | Domains from `DomainIDsWithGrant`, then **per-hit** `Evaluate(view)` (`filterHitsByEntityView`) |
| Ask / retrieval | Orchestration in `retrieval_intelligence` + entity-level checks before evidence is used |

## Deny and inheritance

- Default posture is **deny** if domain scope or grants are missing.
- **entity_acl** deny is evaluated for entity resources after overrides, before grant expansion.
- Materials inherit domain/policy defaults; explicit overrides are exceptions (see [ACCESS_MODEL.md](./ACCESS_MODEL.md)).

## Retrieval and AI constraints

- Keyword search must not return hits outside granted domains (post-filter aligned with resolver).
- Vector/hybrid search (when enabled) must apply the **same** scope **before** returning neighbors to the client or LLM.
- LLM context is built only from entities the principal may **view**; traces recorded in `answer_traces`.

## New code guidelines

1. Call `deps.Permissions.Evaluate` (or `DomainIDsWithGrant`) instead of duplicating SQL.
2. Do not “fetch then filter” sensitive fields in a separate layer without an upstream scope decision.
3. Document new sensitive routes in this file when adding endpoints.
