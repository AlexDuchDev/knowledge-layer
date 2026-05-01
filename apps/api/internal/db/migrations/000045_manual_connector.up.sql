-- Manual ingestion connector. One row, type='manual', auth_mode='none'.
-- Each user-created collection becomes a source_feed against this connector;
-- the collection label + description live in source_feeds.connector_config_json
-- under the "collection" key (see internal/ingestion_connectors/adapters/manual).
--
-- ID 0x15 is the next free slot after openapi_v3 at 0x14 (verified against
-- migrations 000010, 000012, 000040, 000044).
INSERT INTO connectors (id, type, display_name, auth_mode, status) VALUES
    ('20000000-0000-0000-0000-000000000015', 'manual', 'Manual upload', 'none', 'active')
ON CONFLICT (type) DO NOTHING;
