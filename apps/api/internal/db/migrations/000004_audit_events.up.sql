CREATE TABLE audit_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type TEXT NOT NULL,
    actor_type TEXT NOT NULL,
    actor_id UUID,
    target_type TEXT NOT NULL,
    target_id UUID,
    decision TEXT,
    reason TEXT,
    policy_refs_json JSONB NOT NULL DEFAULT '[]',
    trace_ref TEXT,
    metadata_json JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_events_target ON audit_events (target_type, target_id);
CREATE INDEX idx_audit_events_created ON audit_events (created_at);
