-- Generic OpenAPI v3 connector (v0.6.0). One row in `connectors` for the
-- new type so existing source-feed creation flows can pick it from the
-- connector dropdown. Per-feed configuration (openapi_url, list_path,
-- bearer token, item_mapping JSONPaths) lives in
-- `source_feeds.connector_config_json` — see internal/ingestion_connectors/
-- adapters/openapi_v3/config.go for the schema.
--
-- ID 0x14 is the next free slot after the v0.2.x mattermost insert at 0x13
-- (verified: 0x01..0x13 taken by telegram, google_drive, slack, notion,
-- jira, trello, fireflies, google_calendar, microsoft_365, gmail,
-- intercom, confluence, hubspot, zendesk, asana, linear, http_url,
-- filesystem, mattermost).
INSERT INTO connectors (id, type, display_name, auth_mode, status) VALUES
    ('20000000-0000-0000-0000-000000000014', 'openapi_v3', 'OpenAPI v3 (generic)', 'api_token', 'draft')
ON CONFLICT (type) DO NOTHING;
