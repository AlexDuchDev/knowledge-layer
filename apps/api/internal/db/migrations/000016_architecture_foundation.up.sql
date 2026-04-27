CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE entity_types (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    default_sensitivity_level INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE entity_acl (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_id UUID NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    principal_type TEXT NOT NULL,
    principal_id UUID NOT NULL,
    effect TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (entity_id, principal_type, principal_id)
);

CREATE INDEX idx_entity_acl_entity ON entity_acl (entity_id);

CREATE TABLE chunks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_id UUID NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    source_type TEXT NOT NULL DEFAULT 'entity_body',
    text_content TEXT NOT NULL,
    ordinal INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_chunks_entity ON chunks (entity_id);

CREATE TABLE embeddings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chunk_id UUID NOT NULL REFERENCES chunks(id) ON DELETE CASCADE,
    model TEXT NOT NULL,
    embedding vector(1536),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (chunk_id, model)
);

CREATE INDEX idx_embeddings_chunk ON embeddings (chunk_id);

CREATE TABLE knowledge_job_sources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    knowledge_job_id UUID NOT NULL REFERENCES knowledge_jobs(id) ON DELETE CASCADE,
    source_type TEXT NOT NULL,
    source_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (knowledge_job_id, source_type, source_id)
);

CREATE INDEX idx_knowledge_job_sources_job ON knowledge_job_sources (knowledge_job_id);

CREATE TABLE knowledge_job_operators (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    knowledge_job_id UUID NOT NULL REFERENCES knowledge_jobs(id) ON DELETE CASCADE,
    principal_type TEXT NOT NULL,
    principal_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (knowledge_job_id, principal_type, principal_id)
);

CREATE INDEX idx_knowledge_job_operators_job ON knowledge_job_operators (knowledge_job_id);

CREATE TABLE approval_flows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    description TEXT,
    config_json JSONB NOT NULL DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO entity_types (code, display_name, default_sensitivity_level) VALUES
    ('policy', 'Policy', 2),
    ('process_sop', 'Process / SOP', 1),
    ('Insight', 'Insight', 0)
ON CONFLICT (code) DO NOTHING;
