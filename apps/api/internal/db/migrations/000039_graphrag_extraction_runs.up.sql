-- GraphRAG extraction bookkeeping (citations-first answers already exist via answer_traces.*)
--
-- Goal: track idempotent, re-runnable graph extraction over an entity's chunk set
-- and capture operational metadata (tokens, errors, versioning).

CREATE TABLE IF NOT EXISTS graphrag_extraction_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_id UUID NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    extractor_version TEXT NOT NULL,
    input_fingerprint TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at TIMESTAMPTZ,
    tokens_in INT NOT NULL DEFAULT 0,
    tokens_out INT NOT NULL DEFAULT 0,
    error TEXT,
    metadata_json JSONB NOT NULL DEFAULT '{}',
    UNIQUE (entity_id, extractor_version, input_fingerprint)
);

CREATE INDEX IF NOT EXISTS idx_graphrag_extraction_runs_entity
    ON graphrag_extraction_runs (entity_id, started_at DESC);

