DROP INDEX IF EXISTS idx_entity_search_projection_pending_summary;
ALTER TABLE entity_search_projection DROP COLUMN IF EXISTS synthesized_at;
ALTER TABLE entity_search_projection DROP COLUMN IF EXISTS synthesized_summary;
