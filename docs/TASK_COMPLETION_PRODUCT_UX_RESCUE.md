# Task completion: product UX rescue (2026-04-12)

Filled per AGENTS.md documentation contract. Copy sections into PR description as needed.

## Summary

Consolidated primary navigation around the root dash shell; demoted audit and ops to **Advanced**; aligned control-plane and onboarding banners with canonical `/control-plane/*` URLs; replaced API-first chrome with honest capability copy on Home, Search, Jobs, Connectors, and Presets; documented the local golden path across README and user docs.

## Code files changed

- `apps/web/src/lib/navigation.ts`, `controlPlaneNav.ts`, `productAppNav.ts`, `docConcepts.ts`
- `apps/web/src/lib/controlPlaneNav.test.ts`, `productAppNav.test.ts`
- `apps/web/src/components/RoleOnboardingBanner.tsx`, `WorkflowNextSteps.tsx`, `ProductAppShell.tsx`, `control-plane/CpScaffold.tsx`
- `apps/web/src/app/(dash)/page.tsx`, `search/page.tsx`, `jobs/page.tsx`, `connectors/page.tsx`, `admin/presets/page.tsx`
- `apps/web/src/app/control-plane/governance/page.tsx`, `setup/page.tsx`

## Documentation files changed

- `README.md`, `docs/README.md`, `docs/USER_GUIDE.md`, `docs/USER_GUIDE_V1.md`, `docs/CONTROL_PLANE_OVERVIEW.md`, `docs/INFORMATION_ARCHITECTURE_V1.md`, `docs/LIMITATIONS.md`, `docs/EXTERNAL_DEV_QUICKSTART.md`, `docs/RELEASE_READINESS_AUDIT.md`, `docs/user-facing-product-surface.md`, `docs/TASK_COMPLETION_PRODUCT_UX_RESCUE.md` (this file)

## In-product guidance changed

- Home, Search, Jobs, Connectors, Presets, CP setup/governance hubs, CpScaffold related section, banners, workflow next steps.

## Checklist reference

- [x] DOCS_IMPACT_MAP.md reviewed for web + docs paths
- [x] DOCS_MAINTENANCE_POLICY.md
