-- Surfacing preferences only: following does NOT grant access.
CREATE TABLE user_scope_follows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    scope_type TEXT NOT NULL CHECK (scope_type IN ('domain', 'content_hub', 'knowledge_topic', 'digest_stream')),
    ref_id UUID NOT NULL,
    entity_type TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, scope_type, ref_id, entity_type)
);

CREATE INDEX user_scope_follows_user_idx ON user_scope_follows (user_id);
