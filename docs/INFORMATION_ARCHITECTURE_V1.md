# Information architecture — implementation map

**Product source of truth:** [INFORMATION_ARCHITECTURE_PRODUCT_V1.md](./INFORMATION_ARCHITECTURE_PRODUCT_V1.md)  
This file tracks **routes**, **Next.js modules**, and **status** while the UI converges on the product spec.

## Canonical web shells and deprecation policy (2026-04)

| Shell | URL prefix | Role | Canonical? |
|-------|------------|------|------------|
| **Main app (dash)** | Root paths: `/`, `/search`, `/ask`, `/governance`, `/entities`, `/knowledge`, … | Implements [INFORMATION_ARCHITECTURE_PRODUCT_V1.md](./INFORMATION_ARCHITECTURE_PRODUCT_V1.md) routes via `(dash)/` and rewrites. | **Yes** — default for product URLs, docs, and smoke tests. |
| **Control plane** | `/control-plane/*` | Canonical **URL** for administration and builders (nav, docs, bookmarks). Several list/detail routes **rewrite** (and middleware) to existing `(dash)/` implementations so behavior stays intact while URLs converge — see [ADMIN_UI_CONSOLIDATION_PLAN.md](./ADMIN_UI_CONSOLIDATION_PLAN.md). CP-only subtrees (e.g. `scenarios/new`, `jobs/new`, governance queues) render from `control-plane/**`. | **Yes** for admin entry and linking; implementation is mixed scaffold + rewired legacy until builders move fully under CP. |
| **Product app (duplicate)** | `/app/*` | Legacy URL segment; **`ProductAppShell` is not mounted** in the default layout (redirects only). | **No** — **deprecated for navigation**; overlapping paths redirect (308) to root equivalents (see `apps/web/next.config.ts`). Remaining `app/**` pages are for redirects or experiments—do not link them as a second product. |

**Rules for contributors**

1. **New user-facing features** ship on **root / `(dash)/` routes** unless a written ADR or IA update promotes `/app` or `/control-plane`.
2. **Admin and builders:** use **`/control-plane/*`** in nav and new links (see [ADMIN_UI_CONSOLIDATION_PLAN.md](./ADMIN_UI_CONSOLIDATION_PLAN.md)); do not add parallel `/admin/*` entrypoints. Legacy `/admin/*` and some flat URLs redirect (308) to the matching `/control-plane/*` path.
3. **`/app/*`** — do not link from in-app nav to `/app/...` for URLs that have a dash equivalent; listed paths 308 to dash/CP; any other `/app/*` falls back to `/` (see `next.config.ts`).
4. **`/access`** — legacy directory URL; **308** to `/control-plane/users` (canonical).

## Legend

| Status | Meaning |
|--------|---------|
| done | Implemented at canonical URL |
| rewrite | Canonical URL is served via Next.js rewrite to an internal route |
| redirect | Legacy URL redirects (308) to canonical |
| partial | UI exists; depth or filters still evolving |

## Route map

| Spec route (canonical) | Next.js / behaviour | API touchpoints | Status |
|----------------------|---------------------|-----------------|--------|
| `/` | `(dash)/page.tsx` | `GET /home/feed`, `GET /recommendations/browse`, `GET /me/follows` (digest follow controls) | done |
| `/search` | `(dash)/search/page.tsx` | `GET /search` | done (filters + taxonomy rail: browse links + “Filter here” type shortcuts per IA product §7) |
| `/ask` | `(dash)/ask/page.tsx` | `POST /ask` (permission-scoped discovery + synthesis) | done (global Ask + link to Search) |
| `/decisions` | rewrite → `/knowledge/decisions`; `/knowledge/decisions` → redirect `/decisions` | `GET /entities?type=decision` | rewrite + redirect |
| `/policies` | same pattern | `type=policy` | rewrite + redirect |
| `/processes` | same pattern | `type=process_sop` | rewrite + redirect |
| `/meetings` | same pattern | `type=meeting_summary` | rewrite + redirect |
| `/insights` | same pattern | `type=insight` | rewrite + redirect |
| `/projects` | same pattern | `type=project` | rewrite + redirect |
| `/knowledge` | `(dash)/knowledge/page.tsx` | — | done (index of browse types) |
| `/knowledge/[type]` | `(dash)/knowledge/[type]/page.tsx` | `GET /entities`, `GET /recommendations/browse`, `GET/POST/DELETE /me/follows` | done |
| `/entities/[id]` | `(dash)/entities/[id]/page.tsx` | `GET /entities/:id/detail`, `GET /entities/:id/related?depth=2`, recommendations, evidence | partial (Explore-from-here + recommendations; taxonomy explorer on `/search` still per IA product §7) |
| `/hubs` | `(dash)/hubs/...` | `GET /content-hubs` | done |
| `/governance` | `(dash)/governance/page.tsx` | `GET /governance/*`, jobs, feeds | done |
| `/reviews` | `(dash)/reviews/page.tsx` | `GET /review-tasks` | partial |
| `/approvals` | `(dash)/approvals/page.tsx` | `GET /governance/approval-queue` | partial |
| `/control-plane/users` | rewrite → `/access` (list); `/control-plane/users/:id` rewrite → `/admin/users/:id` | `GET /users` | partial |
| `/control-plane/jobs` | rewrite → `/jobs` (list); middleware: `/control-plane/jobs/:id` → `/jobs/:id` (except `new` and segment `runs`) | `GET /knowledge-jobs` | partial |
| `/control-plane/jobs/runs/[id]` | `control-plane/jobs/runs/[id]/page.tsx` (`CpScaffold` + `JobRunDetailClient`) | `GET /job-runs/:id` | done |
| `/control-plane/roles` | rewrite → `/admin/roles` | role APIs | partial |
| `/control-plane/scenarios` | rewrite → `/admin/scenarios` (list); middleware: `/control-plane/scenarios/:id` → `/admin/scenarios/:id` (except `new`) | `GET /scenarios` | partial |
| `/control-plane/presets` | rewrite → `/admin/presets` (list); middleware: `/control-plane/presets/:id` → `/admin/presets/:id` (deeper CP paths unchanged) | preset APIs | partial |
| `/control-plane/sources` | native `control-plane/sources/page.tsx` (hub); guided wizard remains at `/source-feeds?from=cp` | `GET /source-feeds`, connectors APIs | done (hub) |
| `/access` | **308** → `/control-plane/users` (canonical directory; internal rewrite still serves list) | `GET /users` | redirect |
| `/control-plane/sources/connectors` | rewrite → `/connectors` | `GET /connectors` | partial |
| `/control-plane/setup/session/:id` | rewrite → `/admin/setup/:id` | setup APIs | partial |
| `/admin/*` (builders, feeds, users) | **308** → matching `/control-plane/*` (see `next.config.ts`) | — | redirect (legacy bookmarks) |
| `/admin/job-runs/[id]` | **308** → `/control-plane/jobs/runs/[id]`; optional legacy `(dash)/admin/job-runs/[id]` implementation | `GET /job-runs/:id` | redirect + CP |
| `/admin/audit` | **308** → `/audit` | `GET /audit-events` | redirect |
| `/admin/settings` | **308** → `/settings` | `GET /settings/instance` | redirect |
| `/admin/ops/answer-diagnostics` | **308** → `/ops/answer-diagnostics` | `GET /ops/answer-diagnostics` | redirect |
| `/admin/ops/search-insights` | **308** → `/ops/search-insights` | `GET /ops/search-insights` | redirect |
| `/login` | `login/page.tsx` | `POST /auth/login` | done |
| `/invite` | `invite/page.tsx` | invitations | done |
| `/bootstrap` | `bootstrap/page.tsx` | `POST /instance/bootstrap` | done |
| `/control-plane/*` | `control-plane/**` + [`ControlPlaneShell`](../apps/web/src/components/ControlPlaneShell.tsx); plus `middleware.ts` + `next.config` rewrites for canonical URLs → dash builders where noted above | governance / setup subtrees + rewired lists | partial |
| `/app/*` | `app/**` (308 redirects to dash for listed paths); [`ProductAppShell`](../apps/web/src/components/ProductAppShell.tsx) exists for parity if remounted later | deprecated; not used as primary shell | redirect + unused shell |

## Shell and navigation

- **App shell:** `(dash)/layout.tsx` + [`AppShellLayout`](../apps/web/src/components/AppShellLayout.tsx) (header + left nav).
- **Control plane shell:** `control-plane/layout.tsx` + [`ControlPlaneShell`](../apps/web/src/components/ControlPlaneShell.tsx); nav [`controlPlaneNav.ts`](../apps/web/src/lib/controlPlaneNav.ts). See [control-plane-ui-ia.md](./control-plane-ui-ia.md).
- **Deprecated `/app` segment:** `app/layout.tsx` passes children only; **`ProductAppShell` is not wired** into the active tree. Nav config [`productAppNav.ts`](../apps/web/src/lib/productAppNav.ts) remains for tests/docs parity—canonical UX uses root [`navigation.ts`](../apps/web/src/lib/navigation.ts). See [user-facing-product-surface.md](./user-facing-product-surface.md).
- **Nav config:** [`navigation.ts`](../apps/web/src/lib/navigation.ts) filtered by `navigation` object on **`GET /auth/me`** (see [API_SURFACE_V1.md](./API_SURFACE_V1.md)).

## Entity type strings

Canonical `entities.type` values for presets and browse remain defined in:

- `apps/web/src/lib/entityTypes.ts`  
- `@knowledgelayer/shared` (if exported)  
- Go: entity type literals in home/search packages  

## Related docs

- [ADMIN_UI_CONSOLIDATION_PLAN.md](./ADMIN_UI_CONSOLIDATION_PLAN.md) — Canonical `/control-plane` admin URLs, redirects, rewrites  
- [control-plane-ui-ia.md](./control-plane-ui-ia.md) — Control plane IA and routes  
- [user-facing-product-surface.md](./user-facing-product-surface.md) — `/app` product surface  
- [design-system-and-page-templates.md](./design-system-and-page-templates.md) — Shared UI foundation  
- [SEARCH_AND_QA_UX.md](./SEARCH_AND_QA_UX.md) — Search and Ask UX contract  
- [SOURCE_FEED_SETUP_FLOW.md](./SOURCE_FEED_SETUP_FLOW.md) — Source Feed setup UX contract  
- [API_SURFACE_V1.md](./API_SURFACE_V1.md)  
- [ADMIN_UI_V1.md](./ADMIN_UI_V1.md)  
- [DOMAIN_MODEL.md](./DOMAIN_MODEL.md)  
- [AI_RETRIEVAL_GOVERNANCE.md](./AI_RETRIEVAL_GOVERNANCE.md)  
