ALTER TABLE policy_overrides
    ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'active',
    ADD COLUMN IF NOT EXISTS reviewed_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS reviewed_by UUID REFERENCES users(id),
    ADD COLUMN IF NOT EXISTS revoked_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS revoked_by UUID REFERENCES users(id);

UPDATE policy_overrides SET status = 'active' WHERE status IS NULL OR status = '';

CREATE TABLE answer_feedback (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    principal_id UUID NOT NULL REFERENCES users(id),
    trace_id TEXT NOT NULL,
    feedback_kind TEXT NOT NULL,
    comment TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_answer_feedback_trace ON answer_feedback (trace_id);
CREATE INDEX idx_answer_feedback_created ON answer_feedback (created_at DESC);
