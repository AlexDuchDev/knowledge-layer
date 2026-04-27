-- Governance and framework fields for connectors / source feeds (see docs/connector-framework.md).

ALTER TABLE connectors
    ADD COLUMN IF NOT EXISTS auth_config_ref TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS config_json JSONB NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE source_feeds
    ADD COLUMN IF NOT EXISTS external_ref TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS knowledge_scope TEXT NOT NULL DEFAULT 'domain_linked',
    ADD COLUMN IF NOT EXISTS owner_team_id UUID REFERENCES teams(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS notes TEXT,
    ADD COLUMN IF NOT EXISTS sync_status TEXT NOT NULL DEFAULT 'idle';

-- Backfill external_ref from legacy source_uri when present.
UPDATE source_feeds
SET external_ref = source_uri
WHERE external_ref = '' AND source_uri IS NOT NULL AND source_uri <> '';

COMMENT ON COLUMN source_feeds.external_ref IS 'Stable external identifier for the governed source (chat id, channel id, board key, etc.).';
COMMENT ON COLUMN source_feeds.knowledge_scope IS 'How ingested knowledge is scoped (e.g. domain_linked, workspace).';
COMMENT ON COLUMN source_feeds.sync_status IS 'Last-known sync lifecycle state for ops (idle, syncing, error).';
COMMENT ON COLUMN connectors.auth_config_ref IS 'Opaque reference to stored credentials (not the secret itself).';
