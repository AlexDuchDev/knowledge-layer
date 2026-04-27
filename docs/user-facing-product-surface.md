# User-facing product surface (canonical: root dash)

End-user usage layer: Ask, Find, Explore, generated outputs, and governance queues. **Canonical implementation:** root `(dash)/` routes (`/`, `/search`, `/ask`, `/governance`, …) with [`AppShellLayout`](../apps/web/src/components/AppShellLayout.tsx). **Getting started** for mixed operator/product flows: [`/help/getting-started`](../apps/web/src/app/(dash)/help/getting-started/page.tsx). **Source feed wizard** (governed connector setup): `/source-feeds` (opens in the product shell; use `?from=cp` when linked from control plane). The historical `/app/*` segment **redirects** to those URLs; [`ProductAppShell`](../apps/web/src/components/ProductAppShell.tsx) is not mounted in the default layout. Legacy files under [`apps/web/src/app/app/`](../apps/web/src/app/app/) exist for redirects and experiments only. Design system: [design-system-and-page-templates.md](./design-system-and-page-templates.md).

## 1. Product mental model

| Mode | User intent |
|------|-------------|
| Ask | Get an answer with citations from allowed scope |
| Find | Search/filter to specific knowledge |
| Explore | Understand a context (project, domain, type, recency) |
| Consume outputs | Read digests, briefs, summaries |
| Review | Process governed queues when the role allows |

## 2. Main surfaces (canonical URLs)

- **Ask** (`/ask`) — question-led; shares retrieval backbone with search
- **Search** (`/search`) — keyword + filters; not the same UX as Explorer
- **Explorer** (`/entities`, browse by type under `/knowledge`, `/decisions`, …) — structured browse hub
- **Projects** (`/projects`, `/projects/[id]`) — project memory page
- **Decisions** (`/decisions`, `/decisions/[id]`) — decision list and detail
- **Digests** (`/insights`, …) — generated artifacts viewer (legacy `/app/digests` → `/insights`)
- **Governance** (`/governance`, `/reviews`, `/approvals`, …) — reviews and approvals for allowed users

## 3. Route structure

- **`/app/*`** — **Deprecated.** `next.config.ts` issues **308 redirects** to the root routes above (e.g. `/app/search` → `/search`). Do not add new product features under `/app`.

Canonical mirrors (use these in docs and nav):

- `/ask`, `/search`, `/entities`, `/projects`, `/decisions`, `/insights`, `/governance`, `/reviews`, `/approvals`

## 4. Permission-aware behavior

All data loading must use APIs that enforce scope server-side. UI copy reminds users that results reflect their grants; the browser must not pretend to enforce policy.

## 5. Project memory

A dedicated hub per project: summary, linked decisions, documents, meetings, work items, digests, recent activity, risks/blockers when available.

## 6. Decision explorer

Decisions are first-class: list filters, detail with active vs superseded, provenance, links to meetings/docs/issues and related outputs.

## 7. Digest viewer

Digests and summaries are not “just another entity list”: filters by type/date/domain, detail with citations and links to projects/teams/domains and review state when applicable.

## 8. Governance queues

Parallel to [control-plane-ui-ia.md](./control-plane-ui-ia.md) governance: operators use control plane; reviewers use **`/governance`**, **`/reviews`**, and **`/approvals`** (canonical product URLs under the root dash shell). Deprecated **`/app/*`** paths redirect to these routes — do not document `/app/governance` as a separate product surface.

## 9. Integration points

- **Retrieval / AI:** Ask, Search, citations (`POST /ask`, `POST /entities/:id/ask`, optional `scenario_code` on both; `GET /search` with optional `scenario_code` query param)
- **Knowledge core:** entities, links, provenance, lifecycle
- **Jobs / scenarios:** generated digests and summaries as entities or artifacts
- **Governance:** review tasks, approvals, stale flags from APIs
