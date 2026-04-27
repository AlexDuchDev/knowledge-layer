# Competitive readiness — v1 hardening summary

This note captures the **086–100** backlog outcomes: what shipped, what remains intentionally narrow, and how the product stays governance-first versus generic “chat with docs” tools.

## Discovery and trust

- **Explainable recommendations** (`GET /entities/:id/recommendations`, `GET /recommendations/browse`) rank with explicit `reason` strings; every candidate is permission-checked like `/related`.
- **Home feed** combines pending reviews, followed scopes, recent work, digests, and recommended reads without widening retrieval beyond grants.
- **Search / Ask** copy and **PartialViewNotice** on Home, Search, and Knowledge browse explain that empty or short lists reflect **access and sensitivity**, not missing product data.
- **KnowledgeCard** always shows a **TrustLine** when `truth_mode` is present; missing lifecycle/freshness render as placeholders for consistent badges across search and lists.
- **Share internally** (`ShareTrustCard`) copies link + trust metadata for teammates who already have access.

## Workflow and delivery

- **Follow / digest surfacing** (`user_scope_follows`) drives Home and notifications; documented as **surfacing-only** (not grants).
- **Workflow next-step strip** on Home, Search, Ask, Notifications, Reviews, Approvals, and Jobs reduces dead-ends outside the main nav.
- **Role onboarding banner** uses `GET /auth/me` navigation flags and a dismissible localStorage key.
- **Governance upkeep suggestions** surface heuristic maintenance candidates without auto-writing entities.
- **Domain setup kits** expose `GET /onboarding/domain-kits` and `POST /domains/:id/apply-setup-kit` (audit-first v1).
- **Preset catalog** (`/api/presets`) and **session-based setup** (`/api/onboarding/...`) compose seeded templates, preview validation, and launch via catalog instantiation; see [onboarding-setup-flow.md](./onboarding-setup-flow.md) and [preset-catalog.md](./preset-catalog.md).

## AI boundaries

- **Draft suggestions** (`POST /ai/draft-suggestions`) are draft-only, require `view`+`edit`, return UI-applied JSON only, and emit audit events.
- No path adds **retrieve-then-filter** behavior: context assembly stays after successful permission checks (see [AI_RETRIEVAL_GOVERNANCE.md](./AI_RETRIEVAL_GOVERNANCE.md)).

## Gaps / non-goals (v1)

- Follow rows do not trigger email or external delivery; digest preferences are **in-product surfacing** only.
- Setup kit apply is idempotent documentation/audit-oriented; deep automation (roles, feeds, jobs) remains incremental.
- Workflow CTAs are intentionally shallow links; fine-grained hiding per permission would require `/auth/me` on each page (future enhancement).

## References

- [API_SURFACE_V1.md](./API_SURFACE_V1.md) — new endpoints indexed under §9.3–9.6 and §10.2.
- [DOMAIN_MODEL.md](./DOMAIN_MODEL.md) — `user_scope_follows`.
- [ACCESS_MODEL.md](./ACCESS_MODEL.md) — surfacing vs grants.
- [INFORMATION_ARCHITECTURE_V1.md](./INFORMATION_ARCHITECTURE_V1.md) — route map updates.
