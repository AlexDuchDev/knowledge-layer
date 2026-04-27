# CRM / support connector family

Support and conversation systems map to family DTOs with distinct `record_type`s so vendors are not conflated.

| `record_type` | When |
|---------------|------|
| `support_conversation` | Intercom (`NormalizedSupportConversation` usage in sync) |
| `crm_record` | HubSpot CRM objects |
| `support_ticket` | Zendesk tickets |

## Intercom v1

- `connector_config_json`: `intercom_access_token`.
- Lists recent conversations (`/conversations`).

## HubSpot v1

- `connector_config_json`: `hubspot_private_app_token` or `hubspot_access_token`, `hubspot_feed_kind`: `contacts` | `companies` | `deals`.
- Lists up to 25 objects per sync via CRM v3.

## Zendesk v1

- `connector_config_json`: `zendesk_subdomain`, `zendesk_email`, `zendesk_api_token`, `zendesk_feed_kind`: `all` | `view`.
- `external_ref`: required when `zendesk_feed_kind` is `view` (numeric view id).
- Ingests tickets plus per-comment raw rows (`zendesk_comment`).

## Package

Go: `internal/ingestion_connectors/families/crm_support`.
