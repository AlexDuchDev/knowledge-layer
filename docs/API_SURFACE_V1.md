# API_SURFACE_V1.md

## 1. Purpose

This document outlines the initial API surface for v1.

It is not a final OpenAPI spec.
It is the product and engineering map of:

- which API capabilities should exist
- which resources are first-class
- which actions matter
- how API grouping should reflect domain boundaries

The API should be boring, explicit, and trust-aware.

---

## 2. API design principles

- keep handlers thin
- map endpoints to domain concepts
- make permission-sensitive actions explicit
- avoid generic “do everything” endpoints
- expose trust state, not just content
- preserve pagination and filtering early
- support admin explainability where important

---

## 3. Authentication and session

## 3.1 Auth endpoints

- `POST /auth/login`
- `POST /auth/logout`
- `GET /auth/me`

### `GET /auth/me`

Returns:

- user identity
- teams
- roles
- domain grants
- effective high-level permissions for UI bootstrapping

---

## 4. Users and access

## 4.1 Users

- `GET /users`
- `GET /users/:id`
- `POST /users`
- `PATCH /users/:id`

## 4.2 Teams

- `GET /teams`
- `GET /teams/:id`
- `POST /teams`
- `PATCH /teams/:id`

## 4.3 Roles

- `GET /roles`
- `GET /roles/:id`

## 4.4 User role bindings

- `POST /user-role-bindings`
- `PATCH /user-role-bindings/:id`
- `DELETE /user-role-bindings/:id`

## 4.5 Domains

- `GET /domains`
- `GET /domains/:id`
- `POST /domains`
- `PATCH /domains/:id`

## 4.6 Domain grants

- `POST /domain-grants`
- `PATCH /domain-grants/:id`
- `DELETE /domain-grants/:id`

## 4.7 Access policies

- `GET /access-policies`
- `GET /access-policies/:id`
- `POST /access-policies`
- `PATCH /access-policies/:id`

## 4.8 Policy overrides

- `GET /policy-overrides`
- `POST /policy-overrides`
- `PATCH /policy-overrides/:id`
- `DELETE /policy-overrides/:id`

## 4.9 Access evaluation

- `POST /access/evaluate`

### `POST /access/evaluate`

Purpose:
structured effective access decision for admin or system use

Input example:

- principal
- action
- resource_type
- resource_id

Response example:

- allow or deny
- matched policies
- matched overrides
- reason
- sensitivity evaluation

Important:
Requires authentication. Body must include `principal_id`; it must match the caller unless the caller has identity-admin capability (`publish` on a granted domain). See [ACCESS_MODEL.md](./ACCESS_MODEL.md).

---

## 5. Source feeds and ingestion

## 5.1 Connectors

- `GET /connectors`
- `GET /connectors/:id`
- Onboarding discovery (same permission model as Jira: optional `domain_id`, else first granted domain; `manage_source_feed` at sensitivity 1):
  - `POST /integrations/jira/list-projects` — Jira Cloud projects (`jira_site_base_url`, `jira_email`, `jira_api_token`).
  - `POST /integrations/trello/list-boards` — Trello boards (`trello_api_key`, `trello_token`).
  - `POST /integrations/asana/list-projects` — Asana projects (`asana_personal_access_token`).
  - `POST /integrations/linear/list-teams` — Linear teams (`linear_api_key`).
  - `POST /integrations/slack/list-channels` — Slack channels (`bot_token`).
  - `POST /integrations/mattermost/list-channels` — Mattermost channels (`mattermost_base_url`, `mattermost_token`).
  - `POST /integrations/confluence/list-spaces` — Confluence spaces (`confluence_base_url`, `confluence_auth`).
  - `POST /integrations/notion/search` — Notion pages/databases (`notion_integration_token`).
  - `POST /integrations/google-calendar/list-calendars` — Calendar list (`service_account` object).
  - `POST /integrations/google-drive/list-folders` — Child folders (`service_account`, optional `parent_folder_id`, default root).
  - `POST /integrations/zendesk/list-views` — Zendesk saved views (`zendesk_subdomain`, `zendesk_email`, `zendesk_api_token`).
  - OAuth onboarding (Gmail + Microsoft 365; requires `manage_source_feed` at sensitivity 1; one-time handoff via `oauth_sid`):
    - `POST /integrations/oauth/gmail/authorize-url` — Returns `{ "authorize_url": "..." }`.
    - `POST /integrations/oauth/microsoft/authorize-url` — Returns `{ "authorize_url": "..." }`.
    - `GET /integrations/oauth/gmail/callback` — Provider redirect target (`redirect_uri`).
    - `GET /integrations/oauth/microsoft/callback` — Provider redirect target (`redirect_uri`).
    - `POST /integrations/oauth/consume` — Body `{ "oauth_sid": "..." }` → `{ "connector_config_patch": { ... } }` (one-time; 410/404 when expired).

## 5.2 Source feeds

- `GET /source-feeds`
- `GET /source-feeds/:id`
- `POST /source-feeds`
- `PATCH /source-feeds/:id`
- `POST /source-feeds/:id/activate`
- `POST /source-feeds/:id/pause`
- `POST /source-feeds/:id/resume`

Notes (implemented):

- `GET /source-feeds` requires `X-Principal-User-ID` and returns feeds scoped to granted domains; connector config is omitted.
- `GET /source-feeds/:id` requires `manage_source_feed` in the feed domain and may include connector config.

## 5.3 Ingestion runs

- `GET /source-feeds/:id/ingestion-runs`
- `GET /ingestion-runs/:id`
- `POST /source-feeds/:id/sync`
- `POST /source-feeds/:id/reprocess`

## 5.4 Raw artifacts

- `GET /source-feeds/:id/raw-artifacts`
- `GET /raw-artifacts/:id`

Important:
raw artifact access should be separately permissioned.

## 5.5 Normalized records

- `GET /source-feeds/:id/normalized-records`
- `GET /normalized-records/:id`

---

## 6. Entities

## 6.1 Entity listing and detail

- `GET /entities`
- `GET /entities/:id`

Filters should support:

- type
- domain
- owner
- lifecycle_state
- truth_mode
- freshness_status
- approval_status
- linked source feed where appropriate

## 6.2 Entity creation and update

- `POST /entities`
- `PATCH /entities/:id`
- `POST /entities/:id/archive`

## 6.3 Entity relations

- `GET /entities/:id/links`
- `POST /entities/:id/links`
- `DELETE /entity-links/:id`

## 6.4 Entity versions

- `GET /entities/:id/versions`
- `GET /entity-versions/:id`

## 6.5 Entity provenance

- `GET /entities/:id/provenance`

Important:
entity detail responses should include trust and workflow metadata, not just content fields.

## 6.6 Entity Ask (LLM, evidence-scoped)

- `POST /entities/:id/ask`

Input (JSON):

- `question` (required unless `images` or `audio_base64` is provided; used with evidence text for the model)
- `include_related` (optional)
- `answer_strategy` (optional; `standard` | `best_trusted`)
- `scenario_code` (optional) — when non-empty, same fail-closed scenario binding check as `POST /ask` and `GET /search` (`403` if the principal has no matching `role_scenario_bindings` entry via active `user_role_bindings`). Omitted or empty string skips the scenario gate; entity `view` and per-entity retrieval rules still apply.
- `images` (optional) — array of `{ "url": "https://..." | "data:image/..." }` or `{ "data_base64": "...", "media_type": "image/png" }` (max 8), appended as vision parts after sanitized evidence text.
- `audio_base64`, `audio_format` (optional) — short clip transcribed server-side, transcript merged into `question` before synthesis.

Output (shape):
`"trace_id": "uuid", "answer": "string", "citations": [...], "supporting_entities": [...]`

Notes (implemented):

- Requires `X-Principal-User-ID`.
- Access is enforced **before** retrieval; evidence set includes the entity plus optional 1-hop linked entities, each individually checked for `view`.
- The LLM is only given permitted evidence blocks; it must answer with citations referencing those blocks.
- `trace_id` can be used with `POST /answer-feedback`.
- LLM config is via env: `OPENROUTER_API_KEY` (preferred when using OpenRouter) or `OPENAI_API_KEY` with optional `OPENAI_MODEL` / `OPENAI_BASE_URL`. For tests/dev you can set `OPENAI_MOCK=1`.

## 6.7 Global Ask (permission-scoped discovery + synthesis)

- `POST /ask`

Input (JSON): `question` (required for keyword unless `images` / `audio_base64` is provided; used as search keyword for discovery), optional `domain_id`, `type`, `truth_mode`, `lifecycle_state`, `freshness_status`, `approval_status`, `include_related`, `answer_strategy` (`standard` | `best_trusted`), optional `scenario_code`, and the same optional `**images` / `audio_base64` / `audio_format`** multimodal fields as `POST /entities/:id/ask`.

Behavior:

- Discovery uses the same permission model as `GET /search` (domain grants). When OpenSearch is configured, keyword search uses it; otherwise `q` matches `entity_search_projection.title` (ILIKE) within granted domains.
- Top search hits (up to 12 seeds) are loaded; each entity is checked for `view` before entering evidence.
- Optional `include_related` adds 1-hop linked entities from the **first** permitted seed (same cap as entity-scoped Ask).
- Output shape matches `POST /entities/:id/ask` (`trace_id`, `answer`, `citations`, `supporting_entities`, `scope`). `scope.ask_mode` is `global`.
- Answer traces are stored with `entity_id` set to the anchor seed (first permitted hit).

---

## 7. Review and governance

## 7.1 Review tasks

- `GET /review-tasks`
- `GET /review-tasks/:id`
- `POST /review-tasks/:id/start`
- `POST /review-tasks/:id/approve`
- `POST /review-tasks/:id/request-changes`
- `POST /review-tasks/:id/reject`

## 7.2 Approval flows

- `GET /approval-flows`
- `POST /approval-flows`
- `PATCH /approval-flows/:id`

## 7.3 Approval records

- `GET /approval-records`
- `POST /approval-records`

## 7.4 Freshness rules

- `GET /freshness-rules`
- `POST /freshness-rules`
- `PATCH /freshness-rules/:id`

## 7.5 Governance dashboards

- `GET /governance/review-queue`
- `GET /governance/reviews/overdue`
- `GET /governance/approval-queue`
- `GET /governance/stale-content`
- `GET /governance/policy-exceptions`
- `GET /governance/policy-exceptions/:id`

Notes (implemented):

- governance list endpoints support `?limit=` with safe caps.
- overdue/approval queues are scoped to domains where the principal can `publish` (fail-closed).
- `GET /governance/upkeep-suggestions` — heuristic candidates (stale/review_due, thin summary, missing links) for publishers/stewards; no auto-write.

---

## 7.6 Preset catalog

Unified browse + instantiate over role/scenario/job presets (delegates to existing builders). Requires control-plane identity gate (same family as Role/Scenario builder admin routes).

- `GET /api/presets` — query: `type`, `category_axis`, `category_code`
- `GET /api/presets/:id` — detail + preview payload
- `GET /api/presets/:id/related` — outgoing relationships
- `POST /api/presets/:id/instantiate` — body: `name`, `code`, optional `description`, `purpose`, `overrides`

See [preset-catalog.md](./preset-catalog.md).

## 7.7 Onboarding / setup sessions

Resumable setup wizard per `created_by` user.

- `GET /api/onboarding/templates`
- `POST /api/onboarding/sessions`
- `GET /api/onboarding/sessions` — list for current user (`limit` query param, max 100)
- `GET /api/onboarding/sessions/:id`
- `PATCH /api/onboarding/sessions/:id`
- `POST /api/onboarding/sessions/:id/select-template` — body: `template_code`
- `POST /api/onboarding/sessions/:id/preview`
- `POST /api/onboarding/sessions/:id/launch` — 409 if session already launched

See [onboarding-setup-flow.md](./onboarding-setup-flow.md).

---

## 8. Knowledge jobs

## 8.1 Job definitions

- `GET /knowledge-jobs`
- `GET /knowledge-jobs/:id`
- `POST /knowledge-jobs`
- `PATCH /knowledge-jobs/:id`
- `POST /knowledge-jobs/:id/activate`
- `POST /knowledge-jobs/:id/pause`
- `POST /knowledge-jobs/:id/archive`

## 8.2 Job triggers

- `GET /knowledge-jobs/:id/triggers`
- `POST /knowledge-jobs/:id/triggers`
- `PATCH /job-triggers/:id`
- `DELETE /job-triggers/:id`

## 8.3 Job runs

- `GET /knowledge-jobs/:id/runs`
- `GET /job-runs/:id`
- `POST /knowledge-jobs/:id/run`
- `POST /job-runs/:id/cancel`
- `POST /job-runs/:id/retry`

## 8.4 Job outputs

- `GET /job-runs/:id/outputs`
- `GET /job-outputs/:id`

Important:
job run detail should include input scope snapshot and output routing state.

---

## 9. Search and retrieval

## 9.1 Search

- `GET /search`

Supported query params (implemented):

- `q`
- `type`
- `domain_id`
- `owner_id`
- `truth_mode`
- `freshness_status`
- `expand_relations` (when `1` or `true`, includes 1-hop `entity_links` expansion within granted domains only)
- `approval_status`
- `lifecycle_state`
- `scenario_code` (optional) — when non-empty, same fail-closed scenario binding check as `POST /ask` and `POST /entities/:id/ask` (`403` if the principal has no matching `role_scenario_bindings` entry via active `user_role_bindings`).

Response:

- `hits`: each hit includes `entity_id`, `domain_id`, optional `domain_name`, optional `owner_id` / `owner_name`, type/title, trust fields, optional `snippet`, optional `relation_expansion`.
- Snippets are populated when OpenSearch supplies them; trust metadata is always present on hits.

Future: explicit pagination in the JSON contract.

## 9.2 Related entities

- `GET /entities/:id/related`

Query params (implemented):

- `limit` — 1–12, default 6; caps total items including 2-hop rows.
- `depth` — `1` (default) or `2`. When `2`, expands from up to **four** 1-hop neighbors that the principal can `view`, follows `entity_links` one additional step, and **re-checks `view` on every candidate** (no relation bypass). Reasons: `linked:{relation_type}` for 1-hop; `linked_2hop:{relation}:via:{seed_entity_id}` for 2-hop.

Important:
must re-check access on related objects.

## 9.3 Explainable recommendations (implemented)

- `GET /entities/:id/recommendations` — ranked candidates with explicit `reason` strings; each candidate is `view`-checked like `/related`.
- `GET /recommendations/browse` — browse-scoped suggestions (`type`, optional `domain_id`, `limit`).

## 9.4 Home feed (implemented)

- `GET /home/feed` — dashboard sections (pending reviews, followed scopes, recent work, digests, recommended reads); all entity lists are permission-gated.

## 9.5 User scope follows — surfacing only (implemented)

- `GET /me/follows`
- `POST /me/follows` — body: `scope_type` (`domain` | `content_hub` | `knowledge_topic` | `digest_stream`), `ref_id`, optional `entity_type` for `knowledge_topic`.
- `DELETE /me/follows` — same fields to remove a row.

Follow rows **do not grant** domain or entity access; they only influence Home/notifications surfacing.

## 9.6 Domain setup kits (implemented)

- `GET /onboarding/domain-kits` — list built-in kit metadata (admin / publish gate).
- `POST /domains/:id/apply-setup-kit` — idempotent apply stub + audit (v1 does not silently mutate domain content).

---

## 10. Q&A and AI synthesis

## 10.1 Scoped Q&A

- `POST /entities/:id/ask` (entity-scoped; see §6.6)
- `POST /ask` (global discovery + synthesis within search scope; see §6.7)

Legacy / planned name in older drafts:

- `POST /qa/ask` (not mounted; use `/ask` or `/entities/:id/ask`)

Input:

- question
- optional filters
- optional source/entity constraints if allowed

Response:

- answer
- citations
- supporting_entities
- trust_indicators
- partial_view_flag if applicable
- answer_trace_id

## 10.2 AI summaries and extraction previews

**Implemented (mounted on API):**

- `POST /ai/summarize` — admin-gated summarization of pasted text; no persistence of the summary as an entity (see `[routes_register.go](../apps/api/internal/httpserver/routes_register.go)`).
- `POST /ai/draft-suggestions` — **draft entities only** (`lifecycle_state=draft`, not published); requires `view` + `edit`; returns structured suggestions for the UI; **no DB write**; audited as `ai.draft_suggestions`.

**Planned / not mounted as HTTP in v1 (use knowledge jobs or future routes instead):**

- `POST /ai/extract-decisions` — decision extraction is implemented as the `**decision_extraction`** knowledge job processor, not as this REST shortcut.
- `POST /ai/extract-insights` — not mounted; any similar flow should go through governed job outputs.

These endpoints should usually back job or review workflows, not become uncontrolled side doors.

Important:
if these create reusable artifacts, they must route through governed output creation.

## 10.3 Answer traces

- `GET /answer-traces/:id`

This endpoint should be tightly permissioned.

---

## 11. Audit and operations

## 11.1 Audit events

- `GET /audit-events`
- `GET /audit-events/:id`

Filters:

- actor
- event_type
- target_type
- target_id
- date range

## 11.2 Notifications

- `GET /notifications`
- `POST /notifications/:id/read`

## 11.3 Health and operations

- `GET /ops/health`
- `GET /ops/queues`
- `GET /ops/connectors`
- `GET /ops/job-failures`

These may be admin-only.

---

## 12. Response design guidance

Important responses should include:

- object identity
- trust metadata
- workflow metadata
- ownership
- timestamps
- policy-relevant status where appropriate

For example, entity responses should include:

- `truth_mode`
- `lifecycle_state`
- `approval_status`
- `freshness_status`
- `owner`
- `domain`
- `review_due_at`

Do not make the frontend reconstruct trust semantics from many hidden calls.

---

## 13. API grouping recommendation

Keep route groups aligned to product concepts:

- auth
- users/access
- source-feeds/ingestion
- entities
- governance
- knowledge-jobs
- search/qa
- audit/ops

Avoid vague generic buckets like:

- `/core`
- `/system`
- `/data`
- `/assistant`

Those names become junk drawers.

---

## 14. First APIs to build

Build in this order:

### First

- `GET /auth/me`
- `GET /domains`
- `GET /users`
- `POST /source-feeds`
- `GET /source-feeds`
- `POST /source-feeds/:id/activate`
- `POST /source-feeds/:id/sync`

### Then

- `GET /entities`
- `GET /entities/:id`
- `POST /knowledge-jobs`
- `POST /knowledge-jobs/:id/run`
- `GET /review-tasks`

### Then

- `GET /search`
- `POST /qa/ask`
- `GET /audit-events`

This order follows the first end-to-end slice.

---

## 14.1 Second Brain overlay (extracted tasks, chat links, metrics, bots)

Domain grant required where `:domainId` appears (same pattern as other domain APIs).

**Extracted meeting tasks**

- `GET /domains/:domainId/extracted-meeting-tasks` — optional query `review_status`, `limit`
- `POST /domains/:domainId/extracted-meeting-tasks` — create draft task (+ review event + product event)
- `GET /domains/:domainId/second-brain-metrics` — counts by `review_status` and key review events
- `GET /domains/:domainId/second-brain-product-events` — recent rows from `second_brain_product_events` for BI / in-app dashboards (`limit`)
- `GET /extracted-meeting-tasks/:id`
- `PATCH /extracted-meeting-tasks/:id` — while `review_status=draft`
- `POST /extracted-meeting-tasks/:id/confirm-no-edit` | `confirm-after-edit` | `reject`

**Chat identity mapping**

- `GET /me/chat-links` — `telegram_chat_id`, `mattermost_user_id` (for outbound delivery and webhook user resolution)
- `PUT /me/chat-links` — JSON body with optional `telegram_chat_id` / `mattermost_user_id` (omitted fields are left unchanged)

**Inbound webhooks** (no cookie / dev header; shared secret in path — see [SECOND_BRAIN_BOTS.md](./SECOND_BRAIN_BOTS.md), [CONFIG_ENV.md](./CONFIG_ENV.md))

- `POST /webhooks/second-brain/<SECOND_BRAIN_WEBHOOK_SECRET>/telegram` — Telegram `Update` JSON; optional `/ask` prefix; uses same retrieval stack as `POST /ask` when `TELEGRAM_BOT_TOKEN` is set for replies
- `POST /webhooks/second-brain/<SECRET>/mattermost` — `application/x-www-form-urlencoded`; when `MATTERMOST_OUTGOING_WEBHOOK_TOKEN` is set, `token` form field must match; returns `{"text":"..."}` for Mattermost outgoing webhooks

**Workers:** with Redis enabled, `secondbrain:outbound` tasks deliver queued brief text. When `SECOND_BRAIN_PREBRIEF_TICK=1`, `cmd/jobworker` runs pre-meeting delivery after each knowledge scheduled tick (`ProcessPreBriefTick`).

---

## 15. Anti-patterns to avoid

Do not:

- create one giant `/ai` endpoint that does everything
- hide permission decisions behind generic 403s with no structured internal reason
- expose raw artifacts through the same semantics as approved entities
- create endpoints that mix job definition and job execution loosely
- make trust metadata optional in important responses
- use uncontrolled “chat” endpoints that bypass retrieval and policy logic

---

## 16. Final API stance

The API should reflect the product honestly.

A caller should be able to tell:

- what object is being acted on
- what kind of truth it represents
- what workflow state it is in
- what action is being requested
- whether the action is permission-sensitive
- how to inspect provenance and audit when needed

If the API feels generic, the product behavior will drift toward generic too.