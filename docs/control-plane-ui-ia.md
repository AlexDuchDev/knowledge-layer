# Control plane UI information architecture

**Authoritative long-form doc for CP IA.** The companion file [CONTROL_PLANE_UI_IA.md](./CONTROL_PLANE_UI_IA.md) is a short index (policy links only); keep substantive edits in **this** file — see [ADMIN_UI_CONSOLIDATION_PLAN.md](./ADMIN_UI_CONSOLIDATION_PLAN.md) §3.1.

Canonical **URL prefix** for administrators, domain owners, and operators (`/control-plane/*`). Many catalog and builder screens still reuse dash implementations behind internal rewrites — see [ADMIN_UI_CONSOLIDATION_PLAN.md](./ADMIN_UI_CONSOLIDATION_PLAN.md). Implementation: `[apps/web/src/app/control-plane/](../apps/web/src/app/control-plane/)`. Shared UI patterns: [design-system-and-page-templates.md](./design-system-and-page-templates.md).

**Sources hub vs source-feed wizard:** `/control-plane/sources` is the operator **hub** (links to connectors, health, jobs). The guided **source feed creator** is the product-route `**/source-feeds`** (same app; use `?from=cp` when linking from control plane for a subtle return banner). `next.config.ts` no longer rewrites `/control-plane/sources` to `/source-feeds`, so the hub page is always reachable at its canonical URL.

## 1. Mental model

Users think in **setup**, **roles**, **scenarios**, **jobs**, **sources**, **governance**, **presets**, and **users**—not backend packages or internal registries.


| Concept        | Meaning                                                        |
| -------------- | -------------------------------------------------------------- |
| Setup          | First-time launch, templates, sessions, preview, launch result |
| Roles & Access | Role definitions vs assignments vs effective access            |
| Scenarios      | Scenario definitions vs bindings vs outputs in knowledge       |
| Knowledge Jobs | Configured job vs job run (history)                            |
| Sources        | Connector (plugin) vs source feed (configured instance)        |
| Presets        | Catalog/template vs instantiated editable object               |
| Governance     | Review, approval, stale, failures—ongoing ops, not setup       |
| Users          | Directory, assignments, effective visibility                   |


## 2. Top-level navigation

Rendered from `[controlPlaneNav.ts](../apps/web/src/lib/controlPlaneNav.ts)` inside `ControlPlaneShell`: Setup (Getting started, Setup, templates), Configuration (Roles & Access, Scenarios, Knowledge Jobs, Sources hub, source-feed wizard link, Presets), Operations (Governance, Policy exceptions, Users), Product links (Ask, Search, Home). Mobile uses the same groups inside **Menu**; **Product / Control plane** switcher lives in the header next to the user block.

## 3. Screen categories


| Type        | Purpose                              | Examples                           |
| ----------- | ------------------------------------ | ---------------------------------- |
| Catalog     | Browse reusable or live objects      | Roles list, presets catalog        |
| Builder     | Create/edit structured configuration | Role editor, feed editor           |
| Preview     | Effective meaning before save/launch | Role preview, setup launch preview |
| Operational | Live execution and health            | Run history, failed syncs          |
| Binding     | Link roles, scenarios, jobs, sources | Scenario ↔ job bindings            |


## 4. Route structure

Prefix: `/control-plane`.

- **Setup:** `/setup`, `/setup/templates`, `/setup/session/new`, `/setup/session/[id]`, `/setup/wizard`, `/setup/launch-preview`, `/setup/launch-result`
- **Roles:** `/roles`, `/roles/new`, `/roles/[id]`, `/roles/[id]/preview`, `/roles/[id]/assignments`, `/roles/[id]/assign`, `/roles/[id]/clone`
- **Scenarios:** `/scenarios`, `/scenarios/new`, `/scenarios/[id]`, `/scenarios/[id]/preview`, `/scenarios/[id]/clone`, `/scenarios/[id]/bindings/{roles,sources,jobs}`
- **Jobs:** `/jobs`, `/jobs/new`, `/jobs/new/custom`, `/jobs/[id]`, `/jobs/[id]/preview`, `/jobs/[id]/test`, `/jobs/[id]/runs`, `/jobs/[id]/clone`
- **Sources:** `/sources`, `/sources/[id]`, `/sources/connectors`, `/sources/connectors/[id]`, `/sources/feeds/new`, `/sources/feeds/[id]`, `/sources/feeds/[id]/sync`, `/sources/health`
- **Presets:** `/presets`, `/presets/[id]`, `/presets/[id]/instantiate`, `/presets/[id]/related`
- **Governance:** `/governance`, `/governance/reviews`, `/governance/approvals`, `/governance/stale`, `/governance/failed-jobs`, `/governance/failed-syncs`, `/governance/policy-exceptions`
- **Users:** `/users`, `/users/[id]`, `/users/[id]/access`

Index `/control-plane` redirects to `/control-plane/governance`.

## 5. Primary user flows

1. **First-time setup:** Setup template → presets → sources → launch preview → launch result → open created roles/scenarios/jobs.
2. **Add role:** Roles → create/clone → configure → preview → save → assign user.
3. **Add scenario:** Scenarios → preset or new → configure → bind roles/sources/jobs → preview → save.
4. **Add job:** Jobs → preset or custom → scope/trigger/output → preview → test → save.
5. **Add source feed:** Sources → connector → new feed → governance → sync.
6. **Operational health:** Governance dashboard → failed jobs / failed syncs / stale / approvals.

## 6. Cross-navigation rules

Scaffolds use `CpScaffold` **Related** links: preset → instantiate → builder; launch result → catalog lists; job → runs and feeds; feed → jobs and failed syncs; governance queues → jobs/sources; user → roles and legacy detail.

## 7. Dashboard / landing logic

- `**/control-plane`** → redirect to `**/control-plane/governance`** (operator dashboard scaffold).
- **Setup** remains explicit under `/control-plane/setup`; instance bootstrap stays at `**/bootstrap`** until setup sessions are API-backed.
- **Legacy home** (`/`) can surface CTA to control plane when bootstrap complete (see app home page).

## 8. Backend mapping (no table mirroring)


| UI area          | Backend / product concept                             |
| ---------------- | ----------------------------------------------------- |
| Role Builder     | Role definitions, assignments, effective access       |
| Scenario Builder | Scenario definitions, bindings                        |
| Job Builder      | Knowledge job definitions, runs                       |
| Preset catalog   | Preset definitions, relationships, instantiate        |
| Setup flow       | Onboarding templates, sessions, launches (future API) |
| Sources          | Connectors, source feeds, sync runs                   |
| Governance       | Review tasks, approvals, stale/failure states         |


## 9. Visibility

Nav items use the same `NavigationVisibility` flags as the legacy shell where applicable (`may_publish`, `has_domain_grant`, etc.).