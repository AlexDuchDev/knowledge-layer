# Knowledge Layer Web (Next.js 15, App Router, Tailwind)

Two cohabiting surfaces in one app:

- **Product surface** at `/`, `/search`, `/ask`, `/knowledge/*`, `/entities/[id]`, `/governance`, `/meeting-tasks`, … — what end-users see.
- **Control plane** under `/control-plane/*` — operator/admin views (Roles, Scenarios, Jobs, Presets, Setup wizard, Governance queues, Effective-access).

## Quick start (local)

```bash
# 1. From repo root, start the API + infra
make db-up
cd apps/api && go run ./cmd/api    # in one terminal

# 2. In another terminal
cd apps/web
npm install
npm run dev                        # Next dev server on :3000
```

Open http://localhost:3000 — admin user is `30000000-0000-0000-0000-000000000001` (seed).

## Layout

```
apps/web/
  src/
    app/                       # App Router
      (dash)/                  # product surface (search, ask, entities, …)
      control-plane/           # operator / admin (CP-native pages — Phase 2.1.x)
      bootstrap/, login/, invite/
    components/
      control-plane/           # CP-native client components (Roles, Scenarios, Setup, …)
      guidance/                # DocHelpCallout, EmptyStateGuidance, FieldHint
      shared/                  # PageHeader, SectionHeader, AppShell
      templates/               # Catalog/Builder/Detail/Preview/Queue/Setup page templates
      …                        # AskPanel, EntityDetailView, KnowledgeCard, etc.
    lib/
      api.ts                   # apiBase + apiJson — single fetch helper
      navigation.ts            # primary nav (filters by /auth/me capability flags)
      controlPlaneNav.ts       # CP nav
      docConcepts.ts           # in-product help slug → docs URL
      knowledgeRoutes.ts       # entity-type → browse path
      …
    middleware.ts              # remaining minimal CP rewrite (Jobs detail only)
  next.config.ts               # redirects + a small set of beforeFiles rewrites
```

## Environment variables

| Var | Default | What it does |
|---|---|---|
| `NEXT_PUBLIC_API_URL` | `http://localhost:8080` | API base used by `apiJson`. With repo's docker compose mapping use `http://localhost:18080`. |
| `NEXT_PUBLIC_USE_DEV_HEADER` | `false` | When `true`, sends `X-Principal-User-ID` for the local dev pilot. Set `false` for session-cookie auth (production). |
| `NEXT_PUBLIC_PRINCIPAL_USER_ID` | seeded admin UUID | Used only when `NEXT_PUBLIC_USE_DEV_HEADER=true`. |
| `NEXT_PUBLIC_DOCS_BASE_URL` | unset | Optional GitHub `.../blob/<branch>/docs` URL — when set, `DocHelpCallout` links jump to your fork's docs instead of dead-end. |

## Flows

1. **Auth** — session cookie or `X-Principal-User-ID` (dev only) → every `apiJson` call inherits credentials.
2. **Capability-aware nav** — `/auth/me` returns flags (`has_domain_grant`, `may_publish`, `may_approve`, …); `lib/navigation.ts` filters items the user shouldn't see in the sidebar.
3. **Ask / Search** — `/search` (filters + scope summary + `KnowledgeCard`s) and `/ask` (multi-modal Q&A with citations and feedback loop).
4. **Control plane** — native CP pages call `/api/...` for builders and `/governance/*` / `/ops/*` / `/knowledge-jobs/runs` for triage.

## Conventions

- **Use `apiJson<T>(path, init?)`** for every API call — it threads dev-header / session credentials and surfaces a friendly error for cross-host misconfig.
- **Wrap CP pages in `<CpScaffold>`** with a `screenType` ("Catalog" / "Builder" / "Detail" / "Operational" / "Setup") and a `guidanceSlug` from [`lib/docConcepts.ts`](src/lib/docConcepts.ts) so help links resolve.
- **Use page templates** (`components/templates/*`) for new screens — `CatalogPageTemplate`, `BuilderPageTemplate`, `QueuePageTemplate`, `SetupWizardTemplate`. They keep spacing/headers consistent.
- **Suspense-wrap any page that calls `useSearchParams()`** — Next.js 15 prerender requires it (see `app/(dash)/source-feeds/page.tsx` for the canonical pattern).

## Build / lint / typecheck

```bash
npm run dev          # Next dev server with HMR
npm run lint         # ESLint, --max-warnings 0
npm run typecheck    # tsc --noEmit
npm run build        # Next production build (also exercised in CI before merge)
```

`npm run build` should always pass — Phase 2 verification gates the merge on it.

## Boundaries

- **Do not call the API from the server component layer** unless the request is the page render itself; everything dynamic should live in a client component invoked from the server page (consistent with the App Router data-fetching story).
- **Do not duplicate UI between `(dash)/admin/*` and `/control-plane/*`** — Roles, Scenarios, Presets, Setup are now native under `/control-plane/*`. Jobs detail is the one intentional shared view via middleware rewrite.
- **Do not hard-code entity-type strings** — import `ENTITY_TYPES` from `@knowledgelayer/shared`.

## Docs

- App layout map: [docs/apps:web.md](../../docs/apps:web.md)
- Information architecture: [docs/INFORMATION_ARCHITECTURE_V1.md](../../docs/INFORMATION_ARCHITECTURE_V1.md), [docs/CONTROL_PLANE_UI_IA.md](../../docs/CONTROL_PLANE_UI_IA.md)
- Glossary (canonical terms for UI copy): [docs/GLOSSARY.md](../../docs/GLOSSARY.md)
- Search & QA UX spec: [docs/SEARCH_AND_QA_UX.md](../../docs/SEARCH_AND_QA_UX.md)
