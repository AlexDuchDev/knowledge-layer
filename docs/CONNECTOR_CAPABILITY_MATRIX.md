# Connector capability matrix (single source)

**Purpose:** One table for operators and contributors: **manual sync**, **raw artifact types** produced by sync, **async normalization** in the connector worker, and how that relates to **downstream records / entities**. Canonical worker behavior: [`ProcessQueuedRawArtifact`](../apps/api/internal/ingestion_connectors/artifact_worker.go). Manual sync entry points: [`SyncSourceFeed`](../apps/api/internal/ingestion_connectors/google_drive.go). Stubs and degraded modes: [LIMITATIONS.md](./LIMITATIONS.md).

**Glossary:** “Normalized record” is the preferred term for persisted normalized rows (see [GLOSSARY.md](./GLOSSARY.md)).

| Connector `type` | Manual sync (`SyncSourceFeed`) | Example / primary `artifact_type` from sync | Async normalizer in worker | Notes |
|--------------------|----------------------------------|---------------------------------------------|------------------------------|--------|
| `telegram` | Yes | `telegram_update` | **Yes** | Strong chat path; metadata embeds `telegram_update`. |
| `slack` | Yes | `slack_message` | **Yes** | |
| `mattermost` | Yes | `mattermost_post` | **Yes** | |
| `google_drive` | Yes | `google_drive_file` | **Yes** | |
| `notion` | Yes | `notion_page` | **Yes** | |
| `confluence` | Yes | `confluence_page` (+ `confluence_page_body`, `confluence_page_metadata`, `confluence_child_pages`) | **Partial** | Worker normalizes **`confluence_page`** only; other Confluence artifact kinds dequeue as **no-op** until a normalizer exists. |
| `google_calendar` | Yes | `google_calendar_event` | **Yes** | |
| `fireflies` | Yes | `fireflies_transcript` | **Yes** | |
| `jira` | Yes | `jira_issue` | No | Raw retained; queued task **no-op** in worker. |
| `trello` | Yes | `trello_card` | No | Same. |
| `gmail` | Yes | `gmail_message` | No | Same. |
| `intercom` | Yes | `intercom_conversation` | No | Same. |
| `asana` | Yes | `asana_task`, `asana_story` | No | Same. |
| `linear` | Yes | `linear_issue`, `linear_comment` | No | Same. |
| `hubspot` | Yes | `hubspot_object` (kind `hubspot_note` exists for future payloads) | No | Same. |
| `zendesk` | Yes | `zendesk_ticket`, `zendesk_comment` | No | Same. |
| `microsoft_365` | Yes | e.g. `m365_mail_message`, `m365_teams_message`, `m365_calendar_event`, file metadata kinds | No | Same. |

**Entity / search path:** After normalization, families differ in how aggressively they populate `normalized_records`, entity projections, and search indexes. Treat **“sync works”** and **“normalizer runs”** as independent gates; absence of a worker normalizer means **raw preservation without structured normalized rows** for that artifact type (see [LIMITATIONS.md](./LIMITATIONS.md) connector row).

**Maintenance:** When adding a connector type, a new `artifact_type`, or a worker `case`, update this file and [LIMITATIONS.md](./LIMITATIONS.md) in the same change.
