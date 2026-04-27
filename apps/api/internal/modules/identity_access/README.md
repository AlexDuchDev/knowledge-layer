# Module: identity_access

**Purpose:** HTTP transport and application wiring for users, domains, grants, teams, and sessions.

**Domain:** `domain/` — use-case rules; `infra/` — persistence; `transport/` — handlers.

**Flows:** Login/session, invitation, domain listing scoped to principal, identity-admin gates.

**Integrates with:** `identity_access` core package, `platform/permissions.Resolver`.

**Anti-pattern:** Do not expose identity-admin routes without `requireCanManageIdentity` (see `httpserver`).

**Docs:** [docs/PERMISSION_SYSTEM.md](../../../../../docs/PERMISSION_SYSTEM.md), [docs/ACCESS_MODEL.md](../../../../../docs/ACCESS_MODEL.md).
