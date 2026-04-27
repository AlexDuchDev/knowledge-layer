ALTER TABLE source_feeds
    DROP COLUMN IF EXISTS sync_status,
    DROP COLUMN IF EXISTS notes,
    DROP COLUMN IF EXISTS owner_team_id,
    DROP COLUMN IF EXISTS knowledge_scope,
    DROP COLUMN IF EXISTS external_ref;

ALTER TABLE connectors
    DROP COLUMN IF EXISTS config_json,
    DROP COLUMN IF EXISTS auth_config_ref;
