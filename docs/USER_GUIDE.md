# User guide

End-user oriented scenarios and navigation. **Canonical detailed guide:** [USER_GUIDE_V1.md](USER_GUIDE_V1.md).

## Start here

1. **Local golden path** — [USER_GUIDE_V1.md](USER_GUIDE_V1.md#local-golden-path-evaluators) (bootstrap → Home → Search → entity → Ask) and [EXTERNAL_DEV_QUICKSTART.md](EXTERNAL_DEV_QUICKSTART.md).
2. **Concepts** — [PRODUCT_CONCEPTS.md](PRODUCT_CONCEPTS.md), [GLOSSARY.md](GLOSSARY.md).
3. **Product surface** — [USER_FACING_PRODUCT_SURFACE.md](USER_FACING_PRODUCT_SURFACE.md) (canonical: [user-facing-product-surface.md](user-facing-product-surface.md)); canonical routes are **root** URLs, not `/app/*`.
4. **Search & Ask** — [SEARCH_AND_QA_UX.md](SEARCH_AND_QA_UX.md) — how scoped search and citations work.
5. **Admin tasks** — [ADMIN_GUIDE_V1.md](ADMIN_GUIDE_V1.md).
6. **Operator setup** — control-plane setup now uses real onboarding sessions/templates/preview/launch for the supported preset combinations; see [CONTROL_PLANE_OVERVIEW.md](CONTROL_PLANE_OVERVIEW.md) and [LIMITATIONS.md](LIMITATIONS.md).

## In-app help

The web UI links to these docs when `NEXT_PUBLIC_DOCS_BASE_URL` is set (see [.env.example](../.env.example)). Concepts like Role, Scenario, and Source Feed are summarized on control-plane screens.

## Maintenance

If user-visible flows change, update **USER_GUIDE_V1.md** and in-product guidance together ([DOCS_MAINTENANCE_POLICY.md](DOCS_MAINTENANCE_POLICY.md)).
