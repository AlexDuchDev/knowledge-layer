-- Chunk metadata for retrieval / provenance
ALTER TABLE chunks
    ADD COLUMN IF NOT EXISTS token_count INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS metadata_json JSONB NOT NULL DEFAULT '{}';

-- Embedding model lineage (model column remains primary key with chunk_id in UNIQUE)
ALTER TABLE embeddings
    ADD COLUMN IF NOT EXISTS model_version TEXT NOT NULL DEFAULT '';

-- Answer trace: retrieval audit fields
ALTER TABLE answer_traces
    ADD COLUMN IF NOT EXISTS retrieval_mode TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS supporting_chunks_json JSONB NOT NULL DEFAULT '[]',
    ADD COLUMN IF NOT EXISTS metrics_json JSONB NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS prompt_version TEXT NOT NULL DEFAULT '';

-- Approximate nearest neighbor for cosine distance (tune lists for production data volume)
CREATE INDEX IF NOT EXISTS idx_embeddings_ivfflat_cosine
    ON embeddings USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);
