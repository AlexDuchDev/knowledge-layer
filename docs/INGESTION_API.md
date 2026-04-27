# Ingestion & connectors — HTTP (v1)

Operational companion to `INGESTION_AND_CONNECTORS.md` (RTF narrative).

## Connectors

| Method | Path | Gate |
|--------|------|------|
| GET | `/connectors` | open (pilot) |
| GET | `/connectors/:id` | open |
| POST | `/connectors` | identity admin (`publish` on a granted domain) |
| PATCH | `/connectors/:id` | identity admin |

Create body: `type` (unique), `display_name`, optional `auth_mode`, `capabilities_json`.

## Source feeds

| Method | Path | Notes |
|--------|------|--------|
| GET | `/source-feeds` | domains from grants; omits `connector_config_json` in list |
| GET | `/source-feeds/:id` | includes config; `manage_source_feed` |
| POST | `/source-feeds` | `manage_source_feed` on target domain |
| PATCH | `/source-feeds/:id` | same |
| DELETE | `/source-feeds/:id` | archives feed (`status=archived`); excluded from default list |
| POST | `.../preview`, `activate`, `pause`, `resume`, `sync` | `manage_source_feed` |

### Config shapes (pilot)

- **telegram**: `connector_config_json.bot_token`
- **google_drive**: `folder_id`, `service_account` (JSON object), optional `max_files_per_sync`

### Connector template (Epic 10)

1. Migration seed row in `connectors` with stable UUID.
2. Implement `PreviewSourceFeed` / sync in `ingestion_connectors`.
3. Optional admin form in `apps/web/src/app/(dash)/source-feeds`.
