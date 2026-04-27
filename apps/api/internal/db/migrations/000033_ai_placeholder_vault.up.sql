CREATE TABLE ai_placeholder_mappings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    correlation_id TEXT NOT NULL,
    principal_id UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    job_run_id UUID NULL REFERENCES job_runs(id) ON DELETE SET NULL,
    placeholder TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    nonce BYTEA NOT NULL,
    ciphertext BYTEA NOT NULL,
    policy_snapshot_json JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_ai_ph_corr ON ai_placeholder_mappings (correlation_id);
CREATE INDEX idx_ai_ph_expires ON ai_placeholder_mappings (expires_at);
