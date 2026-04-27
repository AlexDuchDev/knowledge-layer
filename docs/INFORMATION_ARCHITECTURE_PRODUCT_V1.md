# Information Architecture — Product specification (v1)

This document is the **canonical product definition** for how the Knowledge Layer web application is structured: primary surfaces, navigation groups, URL conventions, role visibility, and screen depth. Implementation status is tracked in [INFORMATION_ARCHITECTURE_V1.md](./INFORMATION_ARCHITECTURE_V1.md).

---

## 1. Purpose

- Separate **everyday knowledge work** (find, read, ask, browse) from **administration and governance** (access, sources, jobs, audit) while keeping both first-class.
- Use a **persistent primary navigation** so users always know where they are and how to reach governed workflows.
- Enforce **navigation visibility from server-derived capabilities** (not only by hiding links opportunistically on the client).
- Keep URLs **readable, bookmarkable, and aligned with the domain model** (knowledge types, governance queues, admin objects).

---

## 2. Product surfaces

| Surface | Audience | Intent |
|--------|----------|--------|
| **Knowledge App** | All granted users | Home, search, browse by type, entity detail, hubs, Ask (scoped and global entry). |
| **Workflows** | Reviewers, editors, owners | Reviews, approvals, governance hub, editorial queues. |
| **Administration** | Identity admins, operators | Users, access, source feeds, connectors, jobs, job runs, audit, instance settings. |

---

## 3. Global layout rules

- **Header**: product name, environment hint if non-prod, user menu (profile, sign out), optional notifications entry.
- **Left primary navigation**: grouped into **Knowledge**, **Workflows**, **Administration** (labels may be shortened in UI but semantics stay).
- **Content**: max-width reading column for long text; full width for tables and operational screens.
- **Breadcrumbs**: semantic trail `Knowledge > {section} > {item}`; administration paths start with `Administration > …`.

---

## 4. Primary navigation map (by group)

### 4.1 Knowledge

| Label | Route | Notes |
|-------|-------|--------|
| Home | `/` | Domain-aware feed. |
| Search | `/search` | Governed search; filters for trust and lifecycle. |
| Ask | `/ask` | Entry to governed Q&A; may deep-link to entity-scoped Ask. |
| Decisions | `/decisions` | Browse `decision` entities. |
| Policies | `/policies` | Browse `policy`. |
| Processes / SOPs | `/processes` | Browse `process_sop`; supports sub-focus (SOP vs process vs policy handbook) in UI. |
| Meetings | `/meetings` | Browse `meeting_summary`. |
| Insights | `/insights` | Browse `insight`. |
| Topic hubs | `/hubs` | Curated hubs. |

### 4.2 Workflows

| Label | Route | Notes |
|-------|-------|--------|
| Governance | `/governance` | Queues, exceptions, owners, feedback, operational hooks. |
| Reviews | `/reviews` | Review task queue (`/review-tasks`). |
| Approvals | `/approvals` | Approval-oriented queue (`/governance/approval-queue` API). |

### 4.3 Administration

**Canonical URLs and implementation map:** [INFORMATION_ARCHITECTURE_V1.md](./INFORMATION_ARCHITECTURE_V1.md) (route table, rewrites, redirects). Product labels below use **control-plane** paths for bookmarks and nav; legacy `/admin/*` may still redirect or rewrite during consolidation ([ADMIN_UI_CONSOLIDATION_PLAN.md](./ADMIN_UI_CONSOLIDATION_PLAN.md)).

| Label | Route | Notes |
|-------|-------|--------|
| Users | `/control-plane/users` | Directory; detail `/control-plane/users/[id]` (often served via rewrite to existing dash module). |
| Access & trust | `/control-plane/access` | Grants, roles, invitations, trust-related identity ops (or equivalent CP path; see IA V1). |
| Source feeds | `/control-plane/source-feeds` | List + detail `/control-plane/source-feeds/[id]`. |
| Connectors | `/control-plane/connectors` | Connector registry. |
| Knowledge jobs | `/control-plane/jobs` | Job definitions and runs. |
| Job run detail | `/control-plane/jobs/runs/[id]` | Single run inspection (`GET /job-runs/:id`). |
| Audit | `/control-plane/audit` | Audit events. |
| Instance settings | `/control-plane/settings` | Mail test, build info, env reference. |
| Ops diagnostics | `/control-plane/ops/answer-diagnostics`, `/control-plane/ops/search-insights` | Requires governance-style permission in API. |

Public or pre-auth routes stay outside the shell: `/login`, `/invite`, `/bootstrap`.

---

## 5. Entity detail and Ask

- **Entity detail** lives at `/entities/[id]` with trust, provenance, evidence, and **Ask** at `/entities/[id]/ask` (or embedded panel).
- **Related / explore:** entity detail surfaces **governed related entities** (linked objects) and an optional **Explore from here** pattern: bounded traversal with the same permission and trust rules as Search relation expansion ([SEARCH_AND_QA_UX.md](./SEARCH_AND_QA_UX.md) §4.12).
- **Global Ask** at `/ask` must not bypass access control: it only starts a governed flow (e.g. search-first or entity picker), never silent wide retrieval.

---

## 6. URL and routing principles

- **Canonical paths** are those listed in §4. Legacy paths (e.g. `/knowledge/decisions`, `/source-feeds`) **redirect** to canonical URLs where applicable.
- **Implementation detail** may use internal rewrites so one page module serves both legacy and canonical paths during migration.
- **Deep links** to admin objects use stable IDs in the path.

---

## 7. Search and list behaviour

- Search supports filters aligned with API: domain, type, truth mode, lifecycle, freshness, **approval status** (when backed by storage), relation expansion.
- **Taxonomy explorer:** the Search experience should offer a **browse-by-taxonomy** affordance (domain → type → facets) alongside the search box so users can narrow scope before or instead of free-text query ([SEARCH_AND_QA_UX.md](./SEARCH_AND_QA_UX.md) §4.11). Implementation may be a left rail, collapsible panel, or tight integration with `/knowledge` deep-links that pre-fill Search filters.
- Browse lists respect **domain grants** and show empty states when nothing is visible.

---

## 8. Governance and review stance

- **Reviews** and **approvals** are distinct product entry points even when they share backend machinery; both must be reachable from Workflows when the user has capability.
- Editorial and publishing queues remain under **Governance** and link from the governance hub.

---

## 9. Access & trust (admin)

The Access & trust area should expose, over time:

- Domains and default policies  
- Role bindings and coarse grants  
- Invitations lifecycle  
- API keys / automation principals (when productized)  

v1 may ship subsets as long as the **information architecture** reserves the structure (tabs or sub-routes).

---

## 10. Drill-down requirements

Administration objects should have **detail pages** where the API exposes a single resource (canonical **control-plane** paths; see [INFORMATION_ARCHITECTURE_V1.md](./INFORMATION_ARCHITECTURE_V1.md)):

- User: `/control-plane/users/[id]`  
- Source feed: `/control-plane/source-feeds/[id]`  
- Job run: `/control-plane/jobs/runs/[id]`  

List pages link forward; breadcrumbs link back to the parent list.

---

## 11. Empty states and errors

- **403**: explain missing capability; link to home or request access path.  
- **404**: object missing or not visible in scope.  
- **Empty queue**: describe what creates work (e.g. submissions, jobs).

---

## 12. Role and capability matrix (navigation)

High-level mapping (exact checks are implemented in API `navigation` on `GET /auth/me`):

| Capability flag (API) | Typical nav visible |
|------------------------|---------------------|
| `has_domain_grant` | Knowledge group (browse/search within grant). |
| `may_approve` | Reviews, Approvals (workflows). |
| `may_publish` | Governance hub, ops diagnostics, most administration. |
| `may_manage_source_feed` | Emphasize source feeds / ingestion (often overlaps publish). |
| `may_run_job` | Job run / operator affordances. |

**Rule:** Hide primary nav items the user cannot use; do not rely on 403 as the first signal.

---

## 13. Implementation priority

1. Canonical routes + redirects + app shell + nav config.  
2. `GET /auth/me` navigation visibility + client filtering.  
3. First-class `/reviews`, `/approvals`, `/ask`.  
4. Admin drill-down pages.  
5. Search and browse depth (approval filters, process sub-focus, access sub-structure).  
6. Taxonomy explorer on Search + bounded **explore from entity** (related links and optional traverse API) aligned with [AI_RETRIEVAL_GOVERNANCE.md](./AI_RETRIEVAL_GOVERNANCE.md).

---

## 14. Final stance

- **Knowledge objects** and **governance** drive IA, not file folders.  
- **Administration** is not a hidden settings screen; it is a professional operations surface.  
- **AI** remains non-authoritative: Ask and search are retrieval and synthesis **within policy**.  

---

## Related engineering docs

- [INFORMATION_ARCHITECTURE_V1.md](./INFORMATION_ARCHITECTURE_V1.md) — route map and implementation status  
- [SEARCH_AND_QA_UX.md](./SEARCH_AND_QA_UX.md) — Search and scoped Q&A UX contract  
- [SOURCE_FEED_SETUP_FLOW.md](./SOURCE_FEED_SETUP_FLOW.md) — Source Feed setup flow UX contract  
- [API_SURFACE_V1.md](./API_SURFACE_V1.md) — HTTP contract  
- [ACCESS_MODEL.md](./ACCESS_MODEL.md) — access evaluation  
- [DOMAIN_MODEL.md](./DOMAIN_MODEL.md) — entities and lifecycle  
