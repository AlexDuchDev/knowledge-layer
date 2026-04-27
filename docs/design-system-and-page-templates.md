# Design system and page templates

Operational UI foundation for the Knowledge Layer web app. Complements [control-plane-ui-ia.md](./control-plane-ui-ia.md) and [user-facing-product-surface.md](./user-facing-product-surface.md). See also [ADMIN_UI_V1.md](./ADMIN_UI_V1.md).

## 1. Principles

- **One product:** Control plane and end-user surfaces share primitives and semantics; shells differ in navigation emphasis.
- **Reuse patterns:** Prefer templates + shared components over one-off page layout.
- **Clarity over novelty:** Dense, governance-heavy screens prioritize readable hierarchy and state visibility.
- **Composition:** Pages assemble templates; templates assemble shared + UI primitives.
- **Policy in the API:** Badges and labels reflect server-provided state; the browser does not infer permissions.

## 2. Folder structure (`apps/web/src`)

Paths are relative to [`apps/web`](../apps/web).

| Layer | Path | Role |
|-------|------|------|
| Primitives | `components/ui/` | Buttons, inputs, toggles — minimal styling |
| Shared product | `components/shared/` | PageHeader, StatusBadge, FilterBar, states |
| Templates | `components/templates/` | Catalog, builder, detail, queue, setup, ask/search, preview |
| Semantics | `lib/objectState.ts` | Labels + badge classes for lifecycle/operational/kind states |
| Tokens | `app/globals.css` `:root` | Semantic UI tokens (surface/text/border/focus/status) + spacing |

Token usage rules (ADS-inspired):

- Prefer **semantic classes** (`bg-surface-panel`, `text-text-secondary`, `border-border-default`, `ring-status-dangerBorder`) over direct Tailwind palette colors.
- `lib/objectState.ts` is the **only place** that maps domain states → badge styling; pages should never embed status colors.
- All interactive primitives must have a visible focus state (defaults come from global focus styles + component-level rings).

## 3. Shared state semantics

Defined in `lib/objectState.ts` and rendered via `StatusBadge`:

- **Governance lifecycle:** draft, in_review, approved, active, stale, archived, superseded
- **Operational:** active, inactive, failed, running, pending, completed
- **Provenance:** preset vs instantiated
- **Job surface:** configured_job vs job_run
- **Source surface:** connector vs source_feed
- **Visibility hints:** standard, restricted, leadership_only

Do not duplicate color/label maps on individual pages; extend `objectState.ts` when adding a new enum.

## 4. Page template families

| Template | Use for |
|----------|---------|
| `CatalogPageTemplate` | Lists: roles, scenarios, jobs, presets, feeds, users |
| `BuilderPageTemplate` | Multi-section editors with optional summary sidebar |
| `DetailPageTemplate` | Entity / feed / user detail with metadata + linked sections |
| `PreviewPageTemplate` | Effective preview before save or launch |
| `QueuePageTemplate` | Reviews, approvals, stale, failed jobs/syncs |
| `SetupWizardTemplate` | Onboarding steps, progress, resume |
| `AskSearchPageTemplate` | Ask and Search modes (query + filters + results) |

## 5. Layout rules

- **Page width:** Templates default to `max-w-6xl` for operational screens; setup/ask may use `max-w-3xl` / `max-w-4xl`.
- **Section spacing:** Use `space-y-6` between major blocks inside templates.
- **Screen type chip:** `PageHeader` optional `screenType` documents Catalog | Builder | Preview | Operational | Binding | Detail | Setup | Ask | Search for scaffold clarity.

## 6. Form rules

- Use `components/ui/*` for controls; group with `SectionHeader` and consistent `gap-4` / `space-y-3`.
- Validation messages: use `ErrorState` or inline `text-status-danger` under fields (avoid raw `text-red-*`).

## 7. List / detail / queue rules

- **Lists:** `FilterBar` + table or card list + `EmptyState` / `LoadingState`.
- **Detail:** `MetadataPanel` for key-value rows; below, linked sections with `SectionHeader`.
- **Queues:** `QueuePageTemplate` + list of items (future: shared `QueueItemCard`).

## 8. Usage guidance

- New screens should pick a template first, then add domain-specific panels under `components/` or route-local modules.
- When a pattern repeats three times, promote it to `shared/` or `templates/`.
- Import paths: `@/components/...`, `@/lib/objectState`.
