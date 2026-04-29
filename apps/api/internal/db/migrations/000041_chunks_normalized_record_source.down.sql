-- Rollback: restore entity-only chunks. Drops any normalized_record-rooted
-- chunks (and their embeddings via cascade) because the column they depend on
-- ceases to exist. Operators MUST regenerate embeddings after rollback if any
-- chat / meeting / docs content was already chunked under v0.3.0.
DELETE FROM chunks WHERE normalized_record_id IS NOT NULL;
ALTER TABLE chunks DROP CONSTRAINT IF EXISTS chunks_exactly_one_source_chk;
DROP INDEX IF EXISTS idx_chunks_normalized_record;
ALTER TABLE chunks DROP COLUMN IF EXISTS normalized_record_id;
ALTER TABLE chunks ALTER COLUMN entity_id SET NOT NULL;
DROP INDEX IF EXISTS idx_normalized_records_chunks_pending;
ALTER TABLE normalized_records DROP COLUMN IF EXISTS chunks_rebuilt_at;
