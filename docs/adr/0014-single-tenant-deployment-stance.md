# ADR-0014: Single-tenant deployment stance

## Status

**Accepted (2026-04-25).** Supersedes any implicit assumption in earlier ADRs about multi-tenant scope.

## Context

A recurring question from external evaluators is "can one Knowledge Layer instance host multiple unrelated organizations behind tenant isolation?". The answer needs to be explicit so:

- Operators know what they're building when they self-host.
- Contributors know which design choices are off the table.
- Security reviewers can audit against a fixed boundary instead of guessing.

The platform was built around the **9-step `AccessEvaluator`** ([identity_access/access.go](../../apps/api/internal/identity_access/access.go), [PERMISSION_SYSTEM.md](../PERMISSION_SYSTEM.md)) and **domain-scoped permissions** as the access boundary. Domains are an *intra-organization* partition (engineering vs HR), not a tenant-isolation primitive. Adding cross-tenant isolation on top without a schema rewrite would create a soft boundary on top of a hard-domain model — exactly the kind of governance ambiguity AGENTS.md says to avoid.

## Decision

**Knowledge Layer is single-tenant: one instance per organization.** This is a **stable contract**, not a placeholder for "multi-tenant later".

Concretely:

1. **No `tenant_id` column.** The schema does not carry a tenant discriminator on entities, source feeds, jobs, audit events, or any other surface. Adding one would be a major schema migration; we are explicitly choosing not to.
2. **No tenant routing.** The HTTP layer does not parse `X-Tenant-ID` or sub-domain tenancy. Every request is implicitly scoped to "this instance".
3. **No noisy-neighbour controls.** Rate limiting, cost attribution, and quota enforcement are not built around tenant boundaries because there are no tenants.
4. **One Postgres / one Redis / one OpenSearch per instance.** Sharing storage between instances is an operator choice, not a product affordance — sharing means one organization can table-scan another's data.
5. **Bootstrap creates ONE first admin and ONE first domain.** [`instancebootstrap`](../../apps/api/internal/instancebootstrap/) reflects this; multi-tenant would need a separate "tenant onboarding" flow that doesn't exist.
6. **Optional modules respect this.** Second Brain, GraphRAG, and any future optional module assume one tenant — Telegram bots, vector indices, and chat outbound delivery are wired per-instance, not per-tenant.

If you need multi-tenant deployment, run **multiple Knowledge Layer instances**, each in its own Postgres + Redis. Container orchestration (Kubernetes namespaces, ECS services, separate Compose stacks) is the right place for multi-tenant separation — *not* the application layer.

## Consequences

### Positive

- **Simpler access model.** The 9-step evaluator stays focused on the documented intra-organization concerns (roles, scenarios, sensitivity caps, domain grants). No tenant filter needs to wrap every query.
- **Clear security boundary.** Operators auditing the deployment can answer "is org A's data isolated from org B's data?" with "yes — they are different Postgres databases" instead of "yes — there's a `WHERE tenant_id = ?` on every query that we hope nobody forgets to add".
- **Predictable performance.** No per-tenant quota machinery means no noisy-neighbour at the application layer — the only contention is at the infrastructure layer, which the operator already controls.
- **Honest open-source positioning.** README, [OSS_V1_SCOPE.md](../OSS_V1_SCOPE.md), and [SELF_HOSTED.md](../SELF_HOSTED.md) all already say "one organization runs its own instance"; this ADR makes the claim contractual.

### Negative

- **No SaaS path.** A managed multi-tenant SaaS over Knowledge Layer is impossible without a substantial fork. This is a deliberate trade.
- **No "trial" tenancy.** Organizations evaluating Knowledge Layer must stand up their own instance; we do not offer a shared sandbox where multiple evaluators coexist.
- **Migration cost if reversed.** A future major version that wanted multi-tenancy would need a `tenant_id` column on dozens of tables, a migration tool to backfill it on existing rows, and a rewrite of every domain query. The cost of changing this decision later is high — which is precisely why we want the decision to be explicit now rather than emergent.

### Neutral

- **Operators still get organisational hierarchy via domains.** Engineering vs HR vs Legal as separate scopes inside one organization is the supported structure (see [DOMAIN_MODEL.md](../DOMAIN_MODEL.md), [ACCESS_MODEL.md](../ACCESS_MODEL.md)). The single-tenant decision does not constrain how rich the intra-organization access graph can be.
- **Container orchestration is the multi-tenant lever.** Operators wanting "shared compute, isolated data" run multiple Knowledge Layer pods or Compose stacks pointing at separate Postgres databases. This is well-trodden territory — both Kubernetes and Compose handle it natively.

## Alternatives considered

### A) Soft multi-tenancy via domain reuse

Pretend each tenant is a domain. Rejected because **domains are inside the access graph** — one user can hold roles in multiple domains, sensitivity caps span domains, scenarios bind across domains. Treating two unrelated organizations as "two domains" would reuse a primitive that already has cross-cutting semantics, creating exactly the kind of leak the access model is supposed to prevent.

### B) `tenant_id` column on every table

The textbook SaaS approach. Rejected because:
- Every existing query becomes vulnerable to "forgot the WHERE clause" leaks; we'd need either a query interceptor (fragile) or a wholesale rewrite (expensive).
- The 9-step `AccessEvaluator` would need a 10th step ("same tenant?") at the start, and every code path that bypasses the evaluator (admin tooling, migrations, debug scripts) would need a parallel guard.
- The audit value of "one organization = one Postgres database" is much higher than "one organization = a `WHERE tenant_id` clause".

### C) Schema-per-tenant in one Postgres

A middle ground: one cluster, separate Postgres schemas. Rejected because:
- Connection pooling becomes complicated (search-path per request).
- Migrations must run per schema; the auto-migrate-on-startup model breaks.
- The operational savings (one cluster) are usually marginal vs running one container per tenant.

If an operator really wants this, they can do it externally with Postgres roles + search-path setup; Knowledge Layer does not need to know.

## Documentation updates

This ADR is referenced from:

- [README.md](../../README.md) — "Open source and deployment model" section already says self-hosted single-tenant; no change needed.
- [OSS_V1_SCOPE.md](../OSS_V1_SCOPE.md) — already aligned.
- [SELF_HOSTED.md](../SELF_HOSTED.md) — already says "each organization runs its own instance"; aligned.
- [LIMITATIONS.md](../LIMITATIONS.md) — add a row stating multi-tenant is out of scope and pointing to this ADR.

## Revisiting

This decision is not load-bearing on the rest of the architecture; revisiting would be a v2 conversation:

- A new ADR proposing multi-tenancy must include a worked schema migration plan, a per-step access-pipeline review, and a quota/noisy-neighbour design.
- Until that ADR exists and is Accepted, treat any "tenant" suggestion as a request for a separate Knowledge Layer instance.

## Related

- [adr/0001-modular-monolith.md](./0001-modular-monolith.md) — same spirit: prefer simpler architecture until a real bottleneck exists.
- [adr/0002-access-before-retrieval.md](./0002-access-before-retrieval.md) — the access invariant this ADR keeps stable.
- [PERMISSION_SYSTEM.md](../PERMISSION_SYSTEM.md) — what the 9-step evaluator covers, and what it deliberately doesn't.
