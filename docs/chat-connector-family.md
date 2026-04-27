# Chat connector family

The chat family covers **channel-style and group chat** sources (Telegram, Slack, Mattermost, Microsoft Teams, etc.). It defines shared **feed kinds**, **artifact types**, **normalized record shapes**, and **validation** for `connector_config_json` and `external_ref`.

## Feed kinds (`connector_config_json.feed_kind`)


| Value               | Meaning                                             |
| ------------------- | --------------------------------------------------- |
| `channel`           | Public or workspace channel                         |
| `private_channel`   | Private channel                                     |
| `group_chat`        | Multi-party group (default for Telegram when unset) |
| `direct_chat`       | 1:1 DM                                              |
| `thread_collection` | Feed scoped to threads (vendor-specific)            |


**Slack v1** requires `feed_kind` to be set and valid.

**Telegram v1** treats `feed_kind` as optional; if omitted, behavior is `**group_chat`** for documentation and future digest filters.

## Raw artifact types

Stable `raw_artifacts.artifact_type` values for chat-shaped payloads:

- `chat_message`, `chat_message_batch`, `chat_thread`, `chat_reply_set`, `chat_file_reference`

Telegram continues to store the Bot API envelope as `**telegram_update**` for provenance; Mattermost stores `**mattermost_post**` in raw metadata; Slack stores `**slack_message**`. Normalized rows for all three use the shared `**chat_message**` record type with `connector_type` set to `telegram`, `mattermost`, or `slack`.

## Normalized records


| `record_type`  | Payload                        |
| -------------- | ------------------------------ |
| `chat_message` | `NormalizedChatMessage` (JSON) |
| `chat_thread`  | `NormalizedChatThread` (JSON)  |


Key fields on `NormalizedChatMessage`:

- `source_feed_id`, `connector_family` (`chat`), `connector_type` (`telegram`, `slack`, `mattermost`, …)
- `channel_or_chat_ref`, `external_thread_id`, `external_message_id`
- `posted_at`, `author_ref`, `author_display`, `text_body`
- `attachments[]`, optional `raw_provider_payload` for traceability

## Package

Go: `internal/ingestion_connectors/families/chat` — kinds, DTOs, `ValidateFeedKind`, `RequireFeedKindForSlack`, `DefaultFeedKindForTelegram`, `FromTelegramUpdate`, `FromMattermostPost`.

## Governance

Chat family validation does **not** replace framework checks (`MergeFeedPolicyMetadata`, domain, sensitivity, `CanActivate`). Adapters remain in `adapters/<vendor>/` and call family validators where appropriate.