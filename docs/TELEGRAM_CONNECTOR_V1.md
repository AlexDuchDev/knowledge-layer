# Telegram connector v1

Governed ingestion only: this connector is **not** a general-purpose Telegram bot product. It pulls Bot API `getUpdates` into `raw_artifacts` / `normalized_records` for feeds that are explicitly configured and active.

**Implementation:** [`apps/api/internal/ingestion_connectors`](../apps/api/internal/ingestion_connectors) (`SyncTelegram`, `telegram_v1.go`, [`adapters/telegram`](../apps/api/internal/ingestion_connectors/adapters/telegram/telegram.go)). **Framework:** [connector-framework.md](./connector-framework.md), [Connector Framework Specification.md](./Connector%20Framework%20Specification.md).

## Supported

- Connector type `telegram` (seed row in migration `000003`).
- One **primary chat** per source feed via `source_feeds.external_ref` (numeric string, e.g. `-100123`).
- `connector_config_json`: `bot_token`, `allowed_chat_ids` (must include the primary chat id).
- Optional `sync_state.last_update_id` inside `connector_config_json` for incremental `getUpdates` (`offset = last_id + 1`).
- Raw persistence with dedup on `(source_feed_id, content_hash)`; metadata includes adapter `MapArtifactMetadata` output merged with `MergeFeedPolicyMetadata`.
- Normalized rows `record_type = chat_message` with the shared chat-family payload (`NormalizedChatMessage`; staging only; not canonical entities).
- Async: `POST /source-feeds/:id/sync` enqueues `connector:source_sync` when Redis is configured; [`cmd/connectorworker`](../apps/api/cmd/connectorworker/main.go) runs `ingestion_connectors/app.SyncOrchestrator.RunSync`.

## Not supported (v1)

- Listing or discovering all chats (`ListAvailableFeeds` → not supported).
- Ingesting chats that are not the feed’s `external_ref` (updates from other chats in the allowlist are ignored for this feed).
- Full chat history export (only the Bot API update stream; historical messages before bot membership are not backfilled here).
- Webhook push (`WebhookHandler` not implemented on the Telegram adapter yet).

## Governance enforcement

| Checkpoint | Rule |
|------------|------|
| Activation | `ValidateTelegramV1ForActivation(feed, cfg)` — `bot_token`, non-empty `allowed_chat_ids`, `external_ref` parses as int64 and is listed in `allowed_chat_ids`. |
| Adapter | [`adapters/telegram`](../apps/api/internal/ingestion_connectors/adapters/telegram/telegram.go) `ValidateSourceFeedConfig` runs the same for `active`; draft may omit allowlist until configuration is complete. |
| Sync | `SyncTelegram` requires `status = active` and the same validation before calling Telegram. |
| Ingestion | `filterTelegramUpdatesForFeed` keeps only updates whose `message.chat.id` equals the primary chat and is allowlisted. |
| Artifacts | `buildRawArtifactMetadataJSON` → feed governance on every raw row. |

## Sync flow (short)

1. Load feed and connector; ensure `telegram` and `active`.
2. Parse config; validate token, allowlist, and `external_ref`.
3. `startIngestionRun` + `sync_status = syncing`.
4. `getUpdates` with `offset` from `sync_state` (incremental).
5. Compute `max(update_id)` over the **full** API batch to advance the cursor (acknowledges filtered-out updates).
6. For each update for the primary chat: insert `raw_artifact`, then `normalized_record` if new.
7. Finalize run; update feed health / `sync_status`; persist new `last_update_id` when the batch had any updates.

## Next steps

- Webhook-based delivery implementing `WebhookHandler` + deduplicated `webhook_events` (optional table).
- Secret storage via `auth_config_ref` instead of inline `bot_token` in feed config (product decision).
- Richer normalization (media, forwards) and optional blob `storage_uri` for large payloads.
