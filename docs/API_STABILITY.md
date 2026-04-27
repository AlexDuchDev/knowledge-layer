# API stability policy

This document defines what external integrators may assume about Knowledge Layer's HTTP API across releases. Read it before building third-party tooling that depends on `apps/api`.

## TL;DR

- **v0.x (current):** breaking changes are allowed between minor releases. **Do not** build production integrations against v0.x without pinning the exact server version.
- **v1.0 (target):** semver. The HTTP surface declared stable in [API_SURFACE_V1.md](./API_SURFACE_V1.md) follows the deprecation cycle below.
- **v2 and beyond:** require a new major release.

## Versioning model

| Phase | Server tag | API surface | Promise |
|-------|------------|-------------|---------|
| **v0.x** | `v0.MAJOR.MINOR` (current OSS preview) | Endpoints under unversioned paths (`/ask`, `/search`, `/entities`, …) | None. Endpoints, request/response shapes, and headers may change between any two minor versions. CHANGELOG.md flags breaking changes per release. |
| **v1.x** | `v1.MAJOR.MINOR` | Stable endpoints documented in [API_SURFACE_V1.md](./API_SURFACE_V1.md). New endpoints may be introduced; existing ones may be **extended** (new optional fields) but not changed in incompatible ways within the same major. | Semver. Backward-compatible additions in minors. Deprecations honour the cycle below. |
| **v2.x** | `v2.MAJOR.MINOR` | Future major; documented when planned. | Reserved for breaking redesign (e.g. multi-tenant, new auth model). |

### Endpoint path convention

- **v0.x:** unversioned paths only (`/ask`, `/control-plane/*`, …).
- **v1.0:** new endpoints introduced under `/v1/...` are part of the stable contract; legacy unversioned paths remain available but may be marked deprecated and removed in v2.0.
- Webhooks and operator endpoints (`/ops/*`, `/health`, `/metrics`) follow the same versioning.

## Deprecation cycle (v1+)

1. **Deprecation announcement** — flagged in CHANGELOG.md and the endpoint response (`X-Deprecation: <iso-date>` header) for **at least 2 minor releases**.
2. **Removal** — only in the next **major** release.

A deprecated endpoint must continue to work (with the deprecation header) for the entire deprecation window.

## What is NOT covered by this policy

- Internal Go packages under `apps/api/internal/**` — no stability promise. They are private to the binary and may be refactored at will.
- Database schema (`apps/api/internal/db/migrations/`) — migrations are forward-only and applied automatically; downgrade requires backup/restore. Schema is **not** a public API.
- Web UI routes (`apps/web/src/app/**`) — covered by [INFORMATION_ARCHITECTURE_V1.md](./INFORMATION_ARCHITECTURE_V1.md) (canonical URLs are stable; rewrites may move).
- CLI flags, env variables: covered by [CONFIG_ENV.md](./CONFIG_ENV.md) and the version-log section there.

## Practical guidance for integrators

- Pin to an exact server tag (`v0.x.y`) and re-test before upgrading minor versions until v1.0.
- Watch CHANGELOG.md and the [release notes](./RELEASING.md) for `BREAKING:` markers.
- For production integrations, wait for v1.0 or accept the upgrade burden documented above.
- Use `GET /health` as the only contract you can rely on across all versions.

## Related

- [RELEASING.md](./RELEASING.md) — release cadence and tag conventions.
- [API_SURFACE_V1.md](./API_SURFACE_V1.md) — endpoint surface that v1.0 freezes.
- [CHANGELOG.md](../CHANGELOG.md) — per-release change log; breaking-change marker.
