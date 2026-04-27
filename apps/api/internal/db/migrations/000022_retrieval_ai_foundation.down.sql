DROP INDEX IF EXISTS idx_embeddings_ivfflat_cosine;

ALTER TABLE answer_traces
    DROP COLUMN IF EXISTS prompt_version,
    DROP COLUMN IF EXISTS metrics_json,
    DROP COLUMN IF EXISTS supporting_chunks_json,
    DROP COLUMN IF EXISTS retrieval_mode;

ALTER TABLE embeddings
    DROP COLUMN IF EXISTS model_version;

ALTER TABLE chunks
    DROP COLUMN IF EXISTS metadata_json,
    DROP COLUMN IF EXISTS token_count;
