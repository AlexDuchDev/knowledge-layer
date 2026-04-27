# Connector framework (governed ingestion)

Technical reference for the **connector layer** in [`apps/api`](../apps/api). Product-level contract: [INGESTION_AND_CONNECTORS.md](./INGESTION_AND_CONNECTORS.md). Extended framework spec (families, rollout, agent guidance): [Connector Framework Specification.md](./Connector%20Framework%20Specification.md).

## 1. Three concepts (do not merge)

| Concept | Table / type | Role |
|---------|----------------|------|
| **Connector** | `connectors` | Integration *kind* or registered integration account template (`type`, `auth_mode`, `capabilities_json`, `config_json`, `auth_config_ref` for secret references — not raw secrets). |
| **Source feed** | `source_feeds` | One *governed* source instance: `domain_id`, `sensitivity_level`, `knowledge_scope`, `sync_mode`, `allowed_job_types_json`, `owner_id` / `owner_team_id`, `external_ref`, `connector_config_json`, lifecycle `status`. |
| **Raw artifact** | `raw_artifacts` | Immutable (deduped) payload reference for one import batch or message; `metadata_json` carries provenance; **not** canonical knowledge. |

**Sync run:** `ingestion_runs` tracks each sync attempt (`trigger_type`, counts, status).

**Normalized staging:** `normalized_records` links to `raw_artifacts` for downstream entity/material creation (still not auto-canonical without review rules elsewhere).

## 2. Connector categories (roadmap)

The registry and schema support these families without hardcoding each product:

- **Communication / chat:** Telegram, Slack — see [chat-connector-family.md](./chat-connector-family.md)  
- **Email:** Gmail — [email-connector-family.md](./email-connector-family.md)  
- **Meetings / transcripts / calendar context:** Fireflies, Google Calendar — [meeting-transcript-connector-family.md](./meeting-transcript-connector-family.md)  
- **Work management:** Jira, Trello, Asana, Linear — [work-management-connector-family.md](./work-management-connector-family.md)  
- **Documentation / wiki:** Notion, Google Drive / Docs, Confluence — [docs-wiki-connector-family.md](./docs-wiki-connector-family.md)  
- **Microsoft 365:** Outlook mail, Teams, OneDrive, SharePoint files, Outlook calendar — [microsoft-365-connector-family.md](./microsoft-365-connector-family.md)  
- **CRM / support:** Intercom, HubSpot, Zendesk — [crm-support-connector-family.md](./crm-support-connector-family.md)  

**Sync implementations in code:** Telegram, Slack, Notion, Jira, Trello, Fireflies, Google Calendar, Microsoft 365 (Outlook/Teams/OneDrive/SharePoint/calendar), Gmail, Intercom, Google Drive, Confluence, Asana, Linear, HubSpot, Zendesk. Stub connector rows: `000024_connector_family_stubs`; wave-2 types (Confluence, HubSpot, Zendesk, Asana, Linear): `000025_connector_wave2` (draft).

## 2.1 Connector family packages (shared DTOs + validation)

| Family | Go package | Doc |
|--------|------------|-----|
| Chat | [`families/chat`](../apps/api/internal/ingestion_connectors/families/chat) | [chat-connector-family.md](./chat-connector-family.md) |
| Docs / wiki | [`families/docs_wiki`](../apps/api/internal/ingestion_connectors/families/docs_wiki) | [docs-wiki-connector-family.md](./docs-wiki-connector-family.md) |
| Work management | [`families/work_mgmt`](../apps/api/internal/ingestion_connectors/families/work_mgmt) | [work-management-connector-family.md](./work-management-connector-family.md) |
| Meeting / transcript | [`families/meeting`](../apps/api/internal/ingestion_connectors/families/meeting) | [meeting-transcript-connector-family.md](./meeting-transcript-connector-family.md) |
| Microsoft 365 | [`families/microsoft365`](../apps/api/internal/ingestion_connectors/families/microsoft365) | [microsoft-365-connector-family.md](./microsoft-365-connector-family.md) |
| Email | [`families/email`](../apps/api/internal/ingestion_connectors/families/email) | [email-connector-family.md](./email-connector-family.md) |
| CRM / support | [`families/crm_support`](../apps/api/internal/ingestion_connectors/families/crm_support) | [crm-support-connector-family.md](./crm-support-connector-family.md) |

Adapters stay in `adapters/<vendor>/` and compose with these packages; orchestration remains in `ingestion_connectors`.

## 3. Sync modes

Declared on `source_feeds.sync_mode` (validated in code):

- `full_import`
- `incremental`
- `event_driven`
- `manual` (default)

Connectors interpret the mode per technology; the framework stores and surfaces it for scheduling and UX.

## 4. Adapter contract (Go)

**Interface:** [`internal/ingestion_connectors/adapter.go`](../apps/api/internal/ingestion_connectors/adapter.go) — `ConnectorAdapter`:

- `ValidateConnectorConfig` — non-secret connector row / `config_json`
- `ValidateSourceFeedConfig` — feed `connector_config_json` + governance readiness (e.g. Telegram allowlist before activation)
- `ListAvailableFeeds` — optional discovery; return `ErrListAvailableFeedsNotSupported` when not applicable
- `SyncFeed` — connector-specific fetch (persistence stays in `Service` / worker for a single code path)
- `MapArtifactMetadata` — normalize connector payload fragments before merging policy metadata

**Optional webhook extension:** `WebhookHandler` with `HandleWebhook(ctx, WebhookRequest) (*WebhookResult, error)` — see [`adapter.go`](../apps/api/internal/ingestion_connectors/adapter.go). HTTP route `POST /connectors/webhook/:adapter_kind/:source_feed_id` is wired in [`routes_register.go`](../apps/api/internal/httpserver/routes_register.go) and accepts deliveries for any adapter that implements `WebhookHandler`. Auth is intentionally NONE at the route layer — the adapter is the single source of truth for "is this delivery authentic?" via per-feed signature verification (HMAC, per the connector's own protocol). The route caps body size (1 MiB), copies the body before invocation (so the adapter's HMAC hashes the exact bytes), and persists results via `Service.IngestWebhookResult`. Slack is the reference implementation (Phase 2.2.3) — see [`adapters/slack/webhook.go`](../apps/api/internal/ingestion_connectors/adapters/slack/webhook.go) and the test fixture in [`webhook_test.go`](../apps/api/internal/ingestion_connectors/adapters/slack/webhook_test.go) for a curl-able example with mocked signing secret.

**Registry:** `NewRegistry(...).AdapterForConnectorType(connectors.type)`; webhook lookup via `Registry.WebhookHandlerForType(type)`.

**Telegram adapter:** [`adapters/telegram/telegram.go`](../apps/api/internal/ingestion_connectors/adapters/telegram/telegram.go)

**Slack adapter:** [`adapters/slack/slack.go`](../apps/api/internal/ingestion_connectors/adapters/slack/slack.go) (sync) + [`adapters/slack/webhook.go`](../apps/api/internal/ingestion_connectors/adapters/slack/webhook.go) (Events API push)

## 5. Application services (named ports)

- **`ingestion_connectors.Service`** — Postgres-backed connector + feed CRUD, `SyncSourceFeed` (dispatches by `connectors.type`), plus named sync entrypoints (`SyncTelegram`, `SyncSlack`, `SyncNotion`, `SyncJira`, `SyncTrello`, `SyncFireflies`, `SyncGoogleCalendar`, `SyncMicrosoft365`, `SyncGmail`, `SyncIntercom`, `SyncGoogleDrive`, `SyncConfluence`, `SyncAsana`, `SyncLinear`, `SyncHubSpot`, `SyncZendesk`), raw artifact reads.
- **SQL layout:** `connector_repo.go`, `raw_artifact_repo.go`, `ingestion_run_repo.go` (methods on `Service`); `source_feed_scan.go` for feed row scans; **`sync_pipeline.go`** — `startIngestionRun`, `finalizeIngestionRun`, `completeSourceFeedSync`, `buildRawArtifactMetadataJSON`, `appendFeedGovernanceToRawJSON`, `insertRawArtifactRow`.
- **`SourceFeedGovernanceService`** — `Service.SourceFeedGovernance().ValidateCreateInput` wraps `ValidateCreateSourceFeedInput`.
- **`ingestion_connectors/app.SyncOrchestrator`** — `RunSync` / `RunManualSync` → `SyncSourceFeed`; optional `ValidateAdapterConfig`.

## 6. Sync orchestration flow

1. Manual or scheduled request (HTTP or worker).  
2. Load `source_feeds` row; require **active** for Telegram sync.  
3. Resolve `connectors` row by `connector_id`.  
4. Resolve adapter (when registered) for validation; sync implementation switches on `connector.type`.  
5. Create `ingestion_runs` row (`running`).  
6. Set `source_feeds.sync_status = syncing`.  
7. Fetch external data (Bot API, Drive, …).  
8. **Persist `raw_artifacts`** with `content_hash` dedup; Telegram uses `buildRawArtifactMetadataJSON` (adapter `MapArtifactMetadata` + `MergeFeedPolicyMetadata`); Google Drive merges governance via `appendFeedGovernanceToRawJSON` on connector-built JSON.  
9. Optionally write `normalized_records`.  
10. Complete run; set `sync_status` to `idle` or `error`; audit from HTTP when sync was synchronous.

**Async:** `platform/queue.Publisher.EnqueueConnectorSourceSync` → `cmd/connectorworker` → `SyncOrchestrator.RunSync` → `SyncSourceFeed`.

## 7. Telegram v1 constraints

- **Ingestion only** (`ingestion_mode` enforced at product level).  
- **`source_feeds.external_ref`** must be the **primary numeric chat id**; **`allowed_chat_ids`** must include that id (explicit allowlist).  
- Ingestion keeps updates **only for the primary chat** (`filterTelegramUpdatesForFeed`); other allowlisted ids do not create rows on that feed.  
- **`sync_state.last_update_id`** in `connector_config_json` drives incremental `getUpdates` (`offset = last_id + 1`); cursor advances from the **full** Telegram batch so filtered updates are still acknowledged.  
- Adapter `ListAvailableFeeds` returns **not supported** — no “discover all chats” API in v1.  
- Details: [TELEGRAM_CONNECTOR_V1.md](./TELEGRAM_CONNECTOR_V1.md).

## 8. Policy inheritance

[`MergeFeedPolicyMetadata`](../apps/api/internal/ingestion_connectors/policy_inheritance.go) injects a `governance` object into `raw_artifacts.metadata_json`:

- `domain_id`, `sensitivity_level`, `source_feed_id`, `owner_id`, optional `owner_team_id`, `connector_id`, `ingestion_mode`, `sync_mode`, `knowledge_scope`, `external_ref`, `allowed_job_types`

Downstream normalization and entity creation must treat these as inherited defaults unless a stricter policy applies.

## 9. Permission integration

- Creating/updating feeds: `manage_source_feed` / `manage_sources` (see `httpserver` + `identity_access`).  
- Listing feeds: domain-scoped list without `connector_config_json` for non-admin callers.  
- Detail, activate, sync, raw artifacts: `requireManageSourceFeed` or `requireViewRaw` as implemented in [`routes_register.go`](../apps/api/internal/httpserver/routes_register.go).  
- **Unauthorized sync** is rejected with **403** (see integration test `viewer_cannot_trigger_source_sync`).

## 10. Schema additions

Migration [`000020_connector_framework_governance.up.sql`](../apps/api/internal/db/migrations/000020_connector_framework_governance.up.sql):

- `connectors.auth_config_ref`, `connectors.config_json`  
- `source_feeds.external_ref`, `knowledge_scope`, `owner_team_id`, `notes`, `sync_status`  

Base ingestion tables remain from `000003_ingestion.up.sql`.

Migration [`000023_connector_ingestion_indexes.up.sql`](../apps/api/internal/db/migrations/000023_connector_ingestion_indexes.up.sql): B-tree indexes on `source_feeds(connector_id)`, `raw_artifacts(source_feed_id, created_at DESC)`, `ingestion_runs(source_feed_id, started_at DESC)`.

### Spec column crosswalk (names in [Connector Framework Specification](./Connector%20Framework%20Specification.md))

| Spec | Database |
|------|----------|
| `sync_runs` / `sync_jobs` | `ingestion_runs` |
| `raw_storage_path` | `raw_artifacts.storage_uri` |
| `checksum` | `raw_artifacts.content_hash` |
| `imported_at` | `raw_artifacts.created_at` |
| `allowed_jobs_policy` | `source_feeds.allowed_job_types_json` |
| `active` (boolean) | `source_feeds.status = 'active'` (plus `draft` / `paused` / `archived`) |

## 11. Tests

Package [`internal/ingestion_connectors`](../apps/api/internal/ingestion_connectors): governance validation, registry resolution, policy merge, Telegram allowlist / `external_ref`, `getUpdates` offset, metadata merge (`telegram_sync_test.go`), optional `WebhookHandler` lookup.

Integration: `source_feeds_access_test` (sync forbidden for viewer); digest / permission tests use Telegram configs with `allowed_chat_ids`.

## 12. Next implementation tasks

1. Slack OAuth secret ref pattern (beyond bot token in feed config).  
2. Email connector (IMAP / webhook) with feed-level folder / label scope.  
3. Jira / Trello: `external_ref` = board/project key; incremental sync by watermark.  
4. Notion: page/database scope in `connector_config_json`.  
5. Move long-running Drive/Telegram HTTP sync fully behind queue (remove inline fallback in prod config).  
6. Blob `storage_uri` population for large raw payloads.  
7. `ingestion:process_artifact` — `cmd/connectorworker` calls `Service.ProcessQueuedRawArtifact` (idempotent). **`telegram_update`** replays normalization from `metadata_json.telegram_update`; other artifact types no-op (avoid retry loops) until dedicated normalizers are registered. After each task, the worker appends **`ingestion.artifact_processed`** to `audit_events` (`actor_type=system`, `target_type=raw_artifact`, `metadata_json.outcome` = `success` or `error` plus optional `error` text).  
8. Scheduled sync from `sync_mode` + feed status.  
9. Connector-level `ValidateConnectorConfig` using `auth_config_ref` + secret provider.  
10. Expand `modules/ingestion_connectors` transport extraction from `routes_register.go`.

## Related

- [backend-architecture.md](./backend-architecture.md) — worker layout  
- [permission-system.md](./permission-system.md) — source feed enforcement table  
- [migration-plan.md](./migration-plan.md) — phased DB story  
