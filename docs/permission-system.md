# Permission system (organizational memory)

This document is the canonical reference for **backend-enforced** access control. UI hiding is not a security boundary.

## Overview

- **Evaluator**: `internal/identity_access.AccessEvaluator.Evaluate` implements the ordered pipeline.
- **Facade**: `internal/platform/permissions.Resolver` is the preferred entry point for application code.
- **Shared DTOs**: `internal/shared/permissions` provides `ResolutionResult` and input builders for cross-module use.
- **Sensitivity constants**: `internal/identity_access/sensitivity.go` (`public_internal` … `strictly_confidential` as ints 0–4).

## Access dimensions

Decisions combine:

1. User identity (`principal_id`)
2. Team membership (via `entity_acl` team principals)
3. Role assignments (`user_role_bindings` + `role_action_permissions` + `action_permissions`)
4. Domain access (`domain_grants`: `access_level`, `sensitivity_cap`)
5. Entity-type rules (`access_policies.entity_type_scope` per domain)
6. Object-level ACL (`entity_acl` allow/deny)
7. Requested action (normalized; see **Actions**)
8. Resource sensitivity vs grant cap
9. Source feed scope (connector registration + `Evaluate` on `source_feed`)
10. Knowledge job scope (`knowledge_job_sources` strict set, with JSON fallback when empty)
11. Output / publication (enforced via entity and governance routes using `view` / `publish` / `approve`)

## Resolution order (explicit)

The pipeline in `Evaluate` runs in this order:

1. **Authenticated principal** — `PrincipalID == uuid.Nil` → deny (`DENY_MISSING_PRINCIPAL`).
2. **Global deny** — `global_access_denials` row for user → deny (`DENY_GLOBAL_BLOCK`).
3. **Policy overrides** — `policy_overrides` on target; first **deny** stops (`DENY_POLICY_OVERRIDE`).
4. **Entity ACL** — `entity_acl` **deny** stops (`DENY_ENTITY_ACL`). An **allow** row enables bypass of entity-type policy for **`view` / `search`** only (see below).
5. **Domain grant** — `domain_id` required; must have non-expired `domain_grants` (`DENY_NO_DOMAIN_GRANT`).
6. **Entity-type policy** — `access_policies.entity_type_scope` for the domain; if blocked, **allow** ACL row + `view`/`search` may bypass (`DENY_ENTITY_TYPE_POLICY`).
6b. **Role entity-type bindings** — for `resource_type = entity` with `EntityType` set: if applicable roles (same match as step 7) have any `role_entity_type_bindings`, the type must appear in the **union** of those bindings (`DENY_ROLE_ENTITY_TYPE_SCOPE`). If none of those roles declare bindings, this step is a no-op.
7. **Action permission** — role must be **active**, include the action, satisfy optional **`role_domain_bindings`** (non-empty list ⇒ current domain must be listed), and `access_level` must allow the action (`DENY_ROLE_ACTION`, `DENY_ACCESS_LEVEL`).
8. **Sensitivity** — resource `sensitivity_level` must be `<= sensitivity_cap` (`DENY_SENSITIVITY_CAP`).
9. **Allow** — attach `matchedPolicies`; set `ReasonCode = ALLOW_OK`.

Internal phase notes are appended to `AccessDecision.Trace` (not for end-user responses).

### Mapping to specification §7 (object → domain → type → action → sensitivity)

Product specs often list **object-level** controls, then **domain**, **entity-type**, **action**, **sensitivity**. The code uses the same substance; numbering differs only because **auth** and **global deny** are explicit prep steps:

| Spec §7 layer | Code steps | Notes |
|---------------|------------|--------|
| *(pre)* Authenticated user | 1 | `DENY_MISSING_PRINCIPAL` |
| *(pre)* Org-wide block | 2 | `global_access_denials` |
| **Object-level** | **3–4** | **3** = `policy_overrides`; **4** = `entity_acl` (deny wins; allow can narrow type policy for read) |
| **Domain** | 5 | `domain_grants` + `sensitivity_cap` on the grant |
| **Entity-type** | 6, 6b | **6** = domain policy; **6b** = Role Builder `role_entity_type_bindings` union |
| **Action** | 7 | active role, optional `role_domain_bindings`, `role_action_permissions` + `access_level` vs action |
| **Sensitivity** | 8 | resource level vs grant cap; `AccessDecision.SensitivityResult` / `MatchedRuleCode` carry audit-friendly summaries |

Optional `EvaluateInput` fields **`ResourceOwnerID`** and **`AccessPolicyID`** are reserved for future ABAC-style checks; the pipeline does not require them today.

### Chunk and hybrid retrieval (pattern)

There is no separate public “chunk-by-id” HTTP surface yet; search is entity-centric. **When** chunk or hybrid endpoints return text tied to a knowledge object:

1. Resolve the chunk’s **`entity_id`** (or equivalent parent).
2. Load the entity and run **`Evaluate` with action `view`** (same inputs as entity GET).
3. Return chunk content **only** if allowed—never fetch body/snippet first and filter afterward.

Implement this as a single helper (e.g. `CanViewChunk` / `requireEntityViewForChunk`) next to the route handler.

### Source feeds and Telegram ingestion

- **HTTP** paths for source feeds use `Evaluate` on `source_feed` (see `httpserver/source_feed_access.go` and `requireManageSourceFeed`).
- **Workers** must only read feeds that are **registered** for the connector and pass the same declaration checks as `JobAllowsSourceFeed` where a job is involved; avoid ad-hoc “raw feed id” reads without the connector + domain context.

### Job outputs (`GET /job-outputs/:id`)

Today this route is gated with **`requireCanManageIdentity`** (platform admin-style), not `Evaluate` on a linked entity. Published outputs that surface as **entities** use normal entity routes (`view` / `publish` / governance). If first-class **`job_output`** rows become a general API resource, add `ResourceType: "job_output"` (or resolve `output_entity_id` → `Evaluate(view)`) instead of relying on identity-admin only.

## Inheritance (defaults)

- **Entities** inherit domain default policy context through `domain_id` and `access_policies` linked to the domain; materials should be created with explicit `sensitivity_level` and domain.
- **Job outputs** inherit `output_domain_id`, `output_sensitivity_level`, `publication_mode`, and review requirements from the job; provenance links tie outputs to runs.
- **Source feeds** carry `domain_id`, owner, sensitivity, and connector policy; **Telegram** (v1) is ingestion-only: artifacts do not become a public corpus—access still flows through domain grants + entity checks.
- **Object ACL** is an exception layer on top of domain policy (deny wins over broad allow; allow can narrow type restrictions for read as above).

## Deny behavior

- Explicit **deny** in `entity_acl` or `policy_overrides` overrides inherited allow paths.
- **ReasonCode** and **Reasons** explain the failing phase for audit and support tooling (avoid echoing internal trace to clients).

## Actions

Canonical codes include at least:

`view`, `search`, `view_raw`, `edit`, `create`, `approve`, `archive`, `export`, `run_job`, `manage_jobs`, `manage_sources` (legacy HTTP may still say `manage_source_feed`; it normalizes to `manage_sources`), `manage_permissions`, `manage_policies`, `review`, `publish`.

`view` does **not** imply `edit` or `approve`; each is checked via roles + `levelAllowsAction`.

## Retrieval and AI

- **Search** (`internal/search.Service.Search`): domain allow-list **plus** `filterHitsByEntityView` runs full `Evaluate(view)` per hit so snippets/titles cannot leak past ACL, type, or sensitivity.
- **Ask** (`internal/qa`, `internal/retrieval_intelligence`): evidence entities are loaded and passed through `canView` / `Evaluate` **before** LLM context is built (global ask uses search → filtered hits → permitted entities only).

## Schema additions

| Migration | Purpose |
|-----------|---------|
| `000017_permission_actions.up.sql` | Extra `action_permissions` rows; role links for admin/analyst |
| `000018_global_access_denials.up.sql` | `global_access_denials(user_id, reason)` |

## Examples

### Domain isolation

Finance user with grant on domain F can `view` an entity in F; marketing user without grant on F gets `DENY_NO_DOMAIN_GRANT` even if they guess the UUID (HTTP returns 403).

### Sensitivity

User with `sensitivity_cap = 2` cannot `view` an entity at level `leadership_restricted` (3): `DENY_SENSITIVITY_CAP`.

### Job source declaration

`knowledge_job_sources` rows, when present, form the **only** allowed feeds. Jobs populate this table from `source_scope_json` on create when `source_feed_id` is present. The run path uses `runOrchestrator` → `JobAllowsSourceFeed` before the digest reads normalized records.

### Connector admin

`manage_source_feed` in handlers normalizes to `manage_sources` so role assignments align with the expanded catalog.

## HTTP enforcement by route group

| Route group | File / helper | Actions evaluated |
|-------------|---------------|-------------------|
| `POST /entities` | `routes_register.go` | `create` on proposed domain, type, sensitivity |
| `GET` / `PATCH` entity, links, provenance, … | `requireEntityAction`, `entityViewOK` | `view`, `edit`, etc. per handler |
| `POST /entities/:id/promote-canonical` | `routes_register.go` | domain policy with **entity type** from target |
| `GET /review-tasks` | `FilterReviewTasksForPrincipal` | per task: `view` on target entity when `target_type == entity` |
| `GET /review-tasks/:id` | `requireReviewTaskEntityActions` | `view` |
| `POST .../start`, `request-changes`, `reject` | `requireReviewTaskEntityActions` | `view` + `review` |
| `POST .../approve` | `requireReviewTaskEntityActions` | `view` + `approve` |
| `GET /knowledge-jobs`, `GET /knowledge-jobs/:id` | `principalCanViewKnowledgeJob` | `view` or `manage_jobs` on job `output_domain_id`, or job owner; fail closed if domain nil and not owner |
| `GET /knowledge-jobs?expand=scenarios` | same as list | optional scenario binding summary per row |
| `GET /job-builder/presets` | authenticated | no domain object; principal required |
| `POST /knowledge-jobs` | `requireCreateKnowledgeJobCapability` | `manage_jobs` on `output_domain_id` when set; else owner-only draft definitions |
| `PATCH /knowledge-jobs/:id` | `principalCanManageKnowledgeJob` | owner or `manage_jobs` on output domain; full definition patch |
| `POST /knowledge-jobs/:id/clone`, `POST .../scenario-bindings`, `POST .../operators` | `principalCanManageKnowledgeJob` (via `requireManageKnowledgeJobByID` where used) | Job Builder control plane |
| `GET /knowledge-jobs/:id/preview` | `principalCanViewKnowledgeJob` | effective config + validation messages |
| `POST /knowledge-jobs/:id/test-run` | `principalMayRunKnowledgeJob` | `dry_run` skips enqueue; else same as run |
| `GET` / `POST /knowledge-jobs/:id/triggers`, `PATCH` / `DELETE /job-triggers/:id` | `requireManageKnowledgeJobByID` | **Not** platform identity-admin only; same manage rule as job definition |
| `GET /knowledge-jobs/:id/runs` | `principalCanViewKnowledgeJob` | run history |
| `GET /job-runs/:id` | parent job view | same visibility as job detail |
| `POST /knowledge-jobs/:id/run` | `principalMayRunKnowledgeJob` then `Run` | owner or `knowledge_job_operators` always; if `allow_domain_run_job` is true, also domain `run_job` / `manage_jobs`; if false, **no** domain-wide run — sources still enforced in orchestrator |
| `GET /job-outputs/:id` | `requireCanManageIdentity` | admin-style until tied to entities (see above) |
| Search | `search.Service.filterHitsByEntityView` | `view` per hit |
| Ask | `qa` / `retrieval_intelligence` | `view` / `Evaluate` before LLM context |

## Related code paths

| Area | Enforcement |
|------|-------------|
| Entity CRUD / promote | `httpserver.requireEntityAction`, `POST /entities` → `create` |
| Review / governance | `FilterReviewTasksForPrincipal`, `requireReviewTaskEntityActions` |
| Source feeds | `httpserver.requireManageSourceFeed`, `source_feed_access` |
| Jobs | `JobService.Run`, `principalMayRunKnowledgeJob`, `JobAllowsSourceFeed`, `runOrchestrator` |
| Search | `search.Service.filterHitsByEntityView` |
| Ask | `qa` + `retrieval_intelligence` with `canView` |

## Remaining gaps (intentional next steps)

- Row-level **allow** on `entity_acl` that grants **without** any domain grant is not implemented (bypass is type-policy only for view/search).
- **Global deny** is user-scoped only (no role-wide block in v1).
- **Chunk / hybrid** HTTP paths: add evaluator at load time per **Chunk and hybrid retrieval** above.
- **`GET /job-outputs/:id`**: consider entity-linked `Evaluate(view)` when outputs are exposed beyond identity admins.
- **Policy engine** external to SQL is out of scope; rules stay in Postgres for auditability.
