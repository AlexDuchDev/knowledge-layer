-- Decouple chunks from entities: a chunk can now be sourced from a normalized_record
-- in addition to an entity. This unblocks retrieval over chat / meeting / doc / email
-- bodies that today never become entities (only Google Drive docs and a handful of
-- knowledge-job summaries are mapped to entities; the other 14+ record types are
-- normalized but never reach the chunk surface).
--
-- After this migration chunks carry a polymorphic source: exactly one of (entity_id,
-- normalized_record_id) is non-null, and source_type discriminates between them.

-- 1. Allow entity_id to be NULL for normalized_record-rooted chunks.
ALTER TABLE chunks ALTER COLUMN entity_id DROP NOT NULL;

-- 2. New optional FK to normalized_records with cascade delete (drop chunks when
--    the underlying normalized_record is deleted, mirroring entity-cascade).
ALTER TABLE chunks ADD COLUMN normalized_record_id UUID REFERENCES normalized_records(id) ON DELETE CASCADE;

-- 3. Exactly-one-source invariant. PG CHECK is per-row; this guarantees every
--    chunk row is rooted in exactly one parent. source_type still discriminates
--    for filterable queries (entity_body, normalized_record, future kinds).
ALTER TABLE chunks ADD CONSTRAINT chunks_exactly_one_source_chk
    CHECK ((entity_id IS NOT NULL)::int + (normalized_record_id IS NOT NULL)::int = 1);

-- 4. Index for the new join path (normalized_record → chunks).
CREATE INDEX idx_chunks_normalized_record ON chunks (normalized_record_id) WHERE normalized_record_id IS NOT NULL;

-- 5. Backfill source_type for any pre-existing rows. Defensive — the production
--    default already populates 'entity_body', but a forced migration should
--    leave nothing ambiguous.
UPDATE chunks SET source_type = 'entity_body' WHERE source_type = '' OR source_type IS NULL;

-- 6. Tracking column on normalized_records: when chunks have been built for
--    this row, chunks_rebuilt_at is set. NULL = pending. The backfill loop in
--    connectorworker (chunks.RebuildPendingNormalizedRecords) drains rows
--    where this is NULL, in source-feed order. Decouples chunk-rebuild from
--    the 20+ inline INSERT call sites scattered across connector adapters —
--    new connectors land without needing to remember the hook.
ALTER TABLE normalized_records ADD COLUMN chunks_rebuilt_at TIMESTAMPTZ;
CREATE INDEX idx_normalized_records_chunks_pending ON normalized_records (created_at)
    WHERE chunks_rebuilt_at IS NULL;
