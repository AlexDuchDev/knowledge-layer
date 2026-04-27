# @knowledgelayer/shared

TypeScript package with **constants and types shared between `apps/web`, future external client SDKs, and any custom tooling that talks to the Knowledge Layer API**.

The package is intentionally tiny — it stays at the level of *contract* (entity types, lifecycle states, truth modes) rather than *helpers*. UI logic lives in `apps/web/src/lib`; API logic lives in `apps/api/internal`.

## What's in here

| Symbol | Type | Purpose |
|---|---|---|
| `API_VERSION` | `"v1"` | Canonical API surface tag — bump in lockstep with [docs/API_STABILITY.md](../../docs/API_STABILITY.md) when breaking changes ship. |
| `ENTITY_TYPES` | `readonly string[]` | All canonical entity types accepted by the API (`decision`, `policy`, `process_sop`, `meeting_summary`, `insight`, `project`, …). UI selects, browse routes, and external integrations should reference this rather than hard-coding strings. |
| `LIFECYCLE_PUBLISHED` | `"published"` | Lifecycle state constant — keeps UI / external code from drifting if the value is ever renamed. |
| `APPROVAL_APPROVED` | `"approved"` | Approval state constant — same rationale. |
| `TruthMode` | type | `"canonical" \| "mirrored" \| "derived"` — see [ADR-0003](../../docs/adr/0003-truth-classification-model.md). |
| `AccessDecisionResult` | type | `"allow" \| "deny"` — narrow type for the 9-step `AccessEvaluator` decision (see [PERMISSION_SYSTEM.md](../../docs/PERMISSION_SYSTEM.md)). |

## Use from `apps/web`

```ts
import { ENTITY_TYPES, LIFECYCLE_PUBLISHED } from "@knowledgelayer/shared";

const isPublished = (e: Entity) => e.lifecycle_state === LIFECYCLE_PUBLISHED;
```

## Use from an external integration

External tooling should depend on this package directly to avoid string-matching drift. Add it via `npm install @knowledgelayer/shared` once published, or import via path/workspace alias when consuming the monorepo.

## What does NOT belong here

- HTTP client code (lives in `apps/web/src/lib/api.ts` — each consumer wires their own).
- React components (lives in `apps/web/src/components`).
- Anything that imports a Node-only or browser-only API (the package must build to a portable `dist/`).
- Anything that talks to the database, Redis, or LLMs (those belong in `apps/api/internal`).

## Build

```bash
cd packages/shared
npm run build       # tsc → dist/
npm run typecheck   # noEmit check, also wired into root `make lint`
```

## When to bump

Add an export here when **two or more code-bases** would otherwise duplicate the same string constant or type alias. A single consumer is not enough — keep it local until it spreads.
