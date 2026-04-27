-- Emergency / compliance blocks: evaluated in permission resolution before domain grants (see docs/permission-system.md).
CREATE TABLE global_access_denials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reason TEXT NOT NULL DEFAULT 'blocked',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id)
);

CREATE INDEX idx_global_access_denials_user ON global_access_denials (user_id);
