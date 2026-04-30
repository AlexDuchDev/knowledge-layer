-- Add a synthesized-summary column to entity_search_projection so the
-- entity_summarize knowledge job (v0.4.0) can store an LLM-derived 1–3
-- sentence summary alongside the entity title. OpenSearch indexed text
-- widens to include this field — search recall improves on entities whose
-- title alone wouldn't surface the right results (e.g. "Q3 finance review"
-- title with body referencing Stripe billing).
--
-- The column is nullable: entities without a summary fall through to the
-- existing title+body indexing path, so v0.4.0 deploys without disrupting
-- search behavior on day zero. The job backfills incrementally.
ALTER TABLE entity_search_projection ADD COLUMN synthesized_summary TEXT;
ALTER TABLE entity_search_projection ADD COLUMN synthesized_at TIMESTAMPTZ;

-- Lookup: which entities still need summarizing? The job paginates over
-- this predicate. Partial index because most entities will have a summary
-- once the backfill completes.
CREATE INDEX idx_entity_search_projection_pending_summary
    ON entity_search_projection (entity_id)
    WHERE synthesized_summary IS NULL;
