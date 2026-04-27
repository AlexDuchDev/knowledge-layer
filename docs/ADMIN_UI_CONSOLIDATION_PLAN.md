# Admin UI consolidation plan

**Status:** Active source of truth for admin route canonicalization.  
**Related:** [INFORMATION_ARCHITECTURE_V1.md](./INFORMATION_ARCHITECTURE_V1.md), [control-plane-ui-ia.md](./control-plane-ui-ia.md), [CONTROL_PLANE_UI_IA.md](./CONTROL_PLANE_UI_IA.md).

## 1. Goals

- **One canonical admin shell:** `/control-plane/*` (`ControlPlaneShell`) for builders, sources, presets, setup, and operator governance queues.
- **One canonical user shell:** root `(dash)/` routes (`/search`, `/ask`, `/governance`, `/entities`, …) per product IA.
- **No competing “official” admin entrypoints** in primary navigation (`/admin/*` and flat `/jobs` become redirects).
- **OSS clarity:** external docs and nav point to `/control-plane` for administration.

## 2. Canonical target structure

| Concern | Canonical URL prefix | Shell |
|---------|----------------------|--------|
| End-user product | `/`, `/search`, `/ask`, `/knowledge`, `/entities`, `/reviews`, `/approvals`, `/governance`, `/hubs`, … | `(dash)/` + `AppShellLayout` |
| Administration & builders | `/control-plane/*` | `ControlPlaneShell` |
| Instance / global ops (exception) | `/audit`, `/settings`, `/ops/*` | `(dash)/` (no control-plane equivalent yet) |
| Job run detail by run id | `/control-plane/jobs/runs/[id]` | `ControlPlaneShell` + shared `JobRunDetailClient`; `/admin/job-runs/:id` **308** to this URL |

## 3. No-new-duplicates rule

1. **Do not** add new public admin or builder routes outside `apps/web/src/app/control-plane/` (except documented exceptions with a removal date).
2. **Do not** link to `/admin/*`, `/app/*`, or flat `/jobs`/`/connectors`/`/source-feeds`/`/access` for **new** features—use `/control-plane/...` or canonical product URLs.
3. Update `navigation.ts`, `controlPlaneNav.ts`, `productAppNav.ts`, and in-app breadcrumbs to match this policy.
4. Exceptions today: **`/audit`**, **`/settings`**, **`/ops/*`** — legacy **`/admin/job-runs/:id`** redirects to **`/control-plane/jobs/runs/:id`** (physical dash page may remain for deep links until fully removed).

### 3.1 Control-plane IA: stub vs long-form (deduped)

- **Long-form IA (edit here):** [control-plane-ui-ia.md](./control-plane-ui-ia.md) — navigation, catalogs vs builders, CP vs dash.
- **Short index only:** [CONTROL_PLANE_UI_IA.md](./CONTROL_PLANE_UI_IA.md) — links into the long-form doc and policy docs; **do not** copy multi-section content into both files.

### 3.2 Rewrite freeze until native CP pages

- **Do not** add new `next.config.ts` / `middleware.ts` rewrites from `/control-plane/*` to dash routes without updating **this document** and the inventory in §4.2 / parity matrix §5.
- **Exit criterion:** Phase 4 (§7) — remove dash-backed rewrites for a surface only after the matching **native** page under `apps/web/src/app/control-plane/` ships and primary nav no longer depends on the rewrite for that path.
- **Tracking:** file issues per row in §5 where parity is still “dash deeper” or “split”; closing rewrites is **blocked** until those issues are resolved (see §7 Phase 4).

## 4. Route inventory

### 4.1 `(dash)/admin/*` (legacy admin)

| File route | Public URL |
|------------|------------|
| `admin/roles` | `/admin/roles` |
| `admin/scenarios` | `/admin/scenarios` |
| `admin/scenarios/[id]` | `/admin/scenarios/:id` |
| `admin/presets` | `/admin/presets` |
| `admin/presets/[id]` | `/admin/presets/:id` |
| `admin/setup` | `/admin/setup` |
| `admin/setup/[sessionId]` | `/admin/setup/:sessionId` |
| `admin/source-feeds/[id]` | `/admin/source-feeds/:id` |
| `admin/users/[id]` | `/admin/users/:id` |
| `admin/job-runs/[id]` | `/admin/job-runs/:id` |

### 4.2 Flat dash routes (implementation behind some `/control-plane/*` URLs)

These URLs remain valid; primary nav prefers `/control-plane/...`, which may **rewrite** internally to the row below.

| Public URL | Physical page |
|------------|----------------|
| `/access` | `(dash)/access` |
| `/connectors` | `(dash)/connectors` |
| `/jobs`, `/jobs/[id]` | `(dash)/jobs/...` |
| `/source-feeds` | `(dash)/source-feeds` |
| `/audit` | `(dash)/audit` |
| `/settings` | `(dash)/settings` |
| `/ops/answer-diagnostics` | `(dash)/ops/...` |
| `/ops/search-insights` | `(dash)/ops/...` |

### 4.3 `/control-plane/*`

See [control-plane-ui-ia.md](./control-plane-ui-ia.md) §4 (setup, roles, scenarios, jobs, sources, presets, governance, users, including `new`, `[id]`, bindings, preview, clone, sync). Index `/control-plane` redirects to `/control-plane/governance`.

### 4.4 `/app/*` (mirrors)

Already redirected to canonical product URLs: `/app`, `/app/search`, `/app/ask`, `/app/explorer`, `/app/projects`, `/app/decisions`, `/app/governance` (except stale), reviews, approvals, **`/app/digests` → `/insights`**. **Remaining:** none for `/app/*` mirrors beyond maintenance of this list.

## 5. Parity matrix: `/admin` vs `/control-plane` (Phase 1)

| Area | Dash `/admin` or flat | `/control-plane` | Parity | Source of truth going forward |
|------|------------------------|------------------|--------|-------------------------------|
| Roles | `(dash)/admin/roles` — list + API detail/preview/assign | `/control-plane/roles/*` — full builder, clone, preview, assign | **CP deeper** (multi-step flows) | **Control plane** |
| Scenarios | `(dash)/admin/scenarios` — thinner | `/control-plane/scenarios/*` + bindings | **CP deeper** | **Control plane** |
| Presets | `(dash)/admin/presets` | `/control-plane/presets/*` + instantiate/related | **CP deeper** | **Control plane** |
| Setup | `(dash)/admin/setup` | `/control-plane/setup/*` wizard/templates/sessions | **CP deeper** | **Control plane** |
| Jobs | Flat `/jobs` | `/control-plane/jobs/*` + test/runs/clone | **CP deeper** | **Control plane** |
| Source feeds | Flat `/source-feeds`, `/admin/source-feeds/[id]` | `/control-plane/sources/*`, feeds, sync | **CP deeper** | **Control plane** |
| Connectors | Flat `/connectors` | `/control-plane/sources/connectors` | **CP** | **Control plane** |
| Users / access | Flat `/access`, `/admin/users/[id]` | `/control-plane/users/*` | **Split** (access page vs CP users) | **Control plane** for directory; `/access` redirects to `/control-plane/users` |
| Job run **by run id** | `/admin/job-runs/[id]` (redirects) | `/control-plane/jobs/runs/[id]` | **CP** | **`/control-plane/jobs/runs/:id`** |
| Audit | `/audit` | None | **Dash only** | **`/audit`** |
| Settings | `/settings` | None | **Dash only** | **`/settings`** |
| Ops diagnostics | `/ops/*` | None | **Dash only** | **`/ops/*`** |

## 6. Migration table (redirect targets)

| From | To | Notes |
|------|-----|--------|
| `/admin/roles` | `/control-plane/roles` | 308 |
| `/admin/scenarios`, `/admin/scenarios/:path*` | `/control-plane/scenarios`, `/control-plane/scenarios/:path*` | 308 |
| `/admin/presets`, `/admin/presets/:path*` | `/control-plane/presets`, `/control-plane/presets/:path*` | 308 |
| `/admin/setup` | `/control-plane/setup` | 308 |
| `/admin/setup/:sessionId` | `/control-plane/setup/session/:sessionId` | 308 (maps legacy session URL to CP path; rewrites to dash wizard) |
| `/admin/source-feeds` | `/control-plane/sources` | 308 |
| `/admin/source-feeds/:id` | `/control-plane/sources/feeds/:id` | 308 (feed id) |
| `/admin/connectors` | `/control-plane/sources/connectors` | 308 |
| `/admin/jobs`, `/admin/jobs/:path*` | `/control-plane/jobs`, `/control-plane/jobs/:path*` | 308 |
| `/admin/job-runs/:id` | `/control-plane/jobs/runs/:id` | 308 |
| `/admin/users`, `/admin/users/:path*` | `/control-plane/users`, `/control-plane/users/:path*` | 308 |
| `/admin/audit` | `/audit` | 308 (canonical instance) |
| `/admin/settings` | `/settings` | 308 |
| `/admin/ops/:path*` | `/ops/:path*` | 308 |
| `/app/governance/stale` | `/control-plane/governance/stale` | 308 |
| `/access` | `/control-plane/users` | 308 — canonical operator directory URL (see §5 Users/access) |

**Flat dash URLs (optional bookmarks):** `/jobs`, `/source-feeds`, `/connectors`, and `/access` remain valid **without** forcing a 308 to `/control-plane/*` so existing links keep a single hop. Primary nav uses `/control-plane/*`.

**Rewrites + middleware:** `next.config.ts` `beforeFiles` rewrites map canonical `/control-plane` list (and selected) paths to dash implementations (`/jobs`, `/admin/roles`, `/connectors`, `/access`, …). **`/control-plane/sources`** is served by the native CP hub page (no rewrite to `/source-feeds`); the guided wizard remains at **`/source-feeds`**. `middleware.ts` rewrites depth-2 `/control-plane/{scenarios,jobs,presets}/:id` to legacy builders (excluding CP-only segments such as `scenarios/new`, `jobs/new`, and **`jobs/runs/*`** which is a native CP job-run detail page). See [INFORMATION_ARCHITECTURE_V1.md](./INFORMATION_ARCHITECTURE_V1.md) route table.

## 7. Migration phases

| Phase | Work |
|-------|------|
| **0** | This document committed. |
| **1** | Parity matrix reviewed (§5); any CP gaps filed as issues. |
| **2** | Nav updated to `/control-plane/*` + exceptions (`/audit`, `/settings`, `/ops/*`). |
| **3** | Redirects in `next.config.ts`; CP→dash rewrites + `middleware.ts` for canonical URLs (done); drop legacy pages when CP fully implements builders. |
| **4** | Delete or thin `(dash)/admin/*` pages after traffic verified — **blocked** until CP list/builder routes no longer depend on `next.config` rewrites to `/admin/*` and flat `/jobs` (see §5). Do not remove rewrites in the same release as CP parity gaps. |
| **5** | Docs sync (IA, README, impact map). |

## 8. Doc update checklist

- [x] `docs/ADMIN_UI_CONSOLIDATION_PLAN.md` (this file)
- [x] `docs/INFORMATION_ARCHITECTURE_V1.md` — shells table
- [x] `docs/CONTROL_PLANE_UI_IA.md` / `control-plane-ui-ia.md`
- [x] `README.md`, `docs/README.md`
- [x] `docs/DOCS_IMPACT_MAP.md`
- [x] `docs/EXTERNAL_DEV_QUICKSTART.md` — admin entry

## 9. Implementation reference

- Redirects + rewrites: [apps/web/next.config.ts](../apps/web/next.config.ts)
- CP detail rewrites: [apps/web/src/middleware.ts](../apps/web/src/middleware.ts)
- Main nav: [apps/web/src/lib/navigation.ts](../apps/web/src/lib/navigation.ts)
- CP nav: [apps/web/src/lib/controlPlaneNav.ts](../apps/web/src/lib/controlPlaneNav.ts)
- Product app nav: [apps/web/src/lib/productAppNav.ts](../apps/web/src/lib/productAppNav.ts)
