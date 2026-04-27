# OSS v1 scope (single page)

This document is the **release contract** for what the open-source Knowledge Layer repo aims to support in a first serious OSS cut. It aggregates [LIMITATIONS.md](./LIMITATIONS.md), deployment notes, and UI canon. When behavior changes from stub → supported (or the reverse), update **this file** and [LIMITATIONS.md](./LIMITATIONS.md) together.

## Product surface (web)

- **Canonical user shell:** root `(dash)/` routes (`/`, `/search`, `/ask`, `/governance`, `/entities`, …) per [INFORMATION_ARCHITECTURE_V1.md](./INFORMATION_ARCHITECTURE_V1.md).
- **Canonical operator shell:** `/control-plane/*` per [ADMIN_UI_CONSOLIDATION_PLAN.md](./ADMIN_UI_CONSOLIDATION_PLAN.md). Legacy `/admin/*` and `/access` redirect to CP equivalents; `/app/*` is deprecated (see `apps/web/next.config.ts`).
- **Guided source-feed wizard** remains at `/source-feeds?from=cp`, linked from the CP sources hub.

## Knowledge jobs (v1)

- **Wired processors** (orchestrator / worker): see table in [LIMITATIONS.md](./LIMITATIONS.md) — includes `weekly_digest`, `decision_extraction`, `planning_summary`, `stale_scan`, `support_trends_extraction` and their expectations.
- **Other `job_type` values:** fail-closed (reject or explicit run error); UI must not imply full coverage beyond this set.

## Connectors and ingestion

- **Per-family sync vs normalization depth:** single source of truth — [CONNECTOR_CAPABILITY_MATRIX.md](./CONNECTOR_CAPABILITY_MATRIX.md).
- **Artifact worker:** supports a defined set of `artifact_type` normalizers; unknown types no-op to avoid retry storms ([LIMITATIONS.md](./LIMITATIONS.md)).

## AI, retrieval, and privacy

- **Ask and HTTP AI helpers** (`POST /ask`, `POST /ai/*`): `PrivacyGateway` path — [AI_PRIVACY_FLOW.md](./AI_PRIVACY_FLOW.md).
- **Embeddings and other non-chat OpenAI calls:** **outside** `PrivacyGateway` by ADR — [adr/0013-embeddings-privacy-boundary.md](./adr/0013-embeddings-privacy-boundary.md); trust boundary = ingestion + chunk ACL + retrieval filters.
- **Degraded stacks** (no OpenSearch / no LLM key): [SELF_HOSTED.md](./SELF_HOSTED.md) capability matrix.

## Optional modules (feature-flag gated)

These ship in the OSS tree but are **disabled by default** to keep the core deployment focused. Enable per-module by setting the listed env vars; without them the module is a no-op (no records, no scheduled work, no UI side effects).

| Module | Code | Enable flag(s) | Purpose | Reference |
|--------|------|----------------|---------|-----------|
| **Second Brain** (pre-meeting briefs, extracted meeting tasks, Telegram/Mattermost outbound) | `internal/secondbrain/`, `/meeting-tasks` UI | `SECOND_BRAIN_PREBRIEF_TICK=1` (jobworker tick) plus `TELEGRAM_BOT_TOKEN` and/or `MATTERMOST_OUTGOING_WEBHOOK_TOKEN` for outbound channels; `SECOND_BRAIN_WEBHOOK_SECRET` for inbound webhooks | Meeting prep, action extraction, chat-channel delivery | [adr/0012](./adr/0012-second-brain-pilot-scope-defaults.md), [SECOND_BRAIN_OVERLAY_SIZING.md](./SECOND_BRAIN_OVERLAY_SIZING.md) |
| **GraphRAG co-mention expansion** | `internal/graphrag/`, `internal/retrieval_intelligence/graphExpandContextPieces` | `NEO4J_URL` (plus `NEO4J_USER`/`NEO4J_PASSWORD`) | Permission-aware graph expansion of retrieval context (canView filter applied to expanded chunks — Phase 1.1.1 hardening) | [AI_RETRIEVAL_GOVERNANCE.md](./AI_RETRIEVAL_GOVERNANCE.md) |

Optional modules are second-class for evaluation: a fork that ignores the flags must still pass `make test` and `scripts/smoke-local.sh` end-to-end against the core.

## Storage and ops defaults

- **Blob / raw payload:** default in-process `blobstore.Nop` — not durable object storage; production retention needs an S3/R2-style adapter ([LIMITATIONS.md](./LIMITATIONS.md)).
- **OpenSearch in local compose:** security disabled — **dev only**; production requires TLS + auth ([LIMITATIONS.md](./LIMITATIONS.md)).
- **Metrics:** `GET /metrics` with `OPS_AUTH_TOKEN` outside local ([LIMITATIONS.md](./LIMITATIONS.md)).

## What “v1 OSS complete” means here

1. Docs above match runtime behavior (no silent “enterprise-only” features in OSS nav).
2. Canonical URLs in IA are the only **documented** entrypoints for smoke tests and external guides.
3. Limitations table stays honest when adding connectors, jobs, or AI surfaces.

## Related

- [GETTING_STARTED.md](./GETTING_STARTED.md), [EXTERNAL_DEV_QUICKSTART.md](./EXTERNAL_DEV_QUICKSTART.md)
- [RELEASE_READINESS_AUDIT.md](./RELEASE_READINESS_AUDIT.md) (if present and maintained)
- [PRODUCTION_GO_LIVE_CHECKLIST.md](./PRODUCTION_GO_LIVE_CHECKLIST.md), [INFRA_PRODUCTION_REFERENCE.md](./INFRA_PRODUCTION_REFERENCE.md), [PRODUCTION_CUTOVER_QUICKREF.md](./PRODUCTION_CUTOVER_QUICKREF.md)
- [STAGING_SMOKE_TEST.md](./STAGING_SMOKE_TEST.md) (includes session smoke via [`scripts/smoke-session.sh`](../scripts/smoke-session.sh))
- [RELEASING.md](./RELEASING.md), [`scripts/repo-sanity-check.sh`](../scripts/repo-sanity-check.sh)
- [POST_V1_HARDENING.md](./POST_V1_HARDENING.md) — after v1: retention, cost, SLOs, scale.

### Version log

- **2026-04-21** — Linked production/OSS operator docs, session smoke script, releasing and repo sanity checks (docs-only contract refresh).
- **2026-04-24** — Phase 1 alignment: added "Optional modules" section (Second Brain promoted from Provisional ADR-0012 to Accepted; GraphRAG documented as feature-flag-gated). Documented vault fail-closed in production hardening.
