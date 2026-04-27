CREATE TABLE answer_traces (
    id UUID PRIMARY KEY,
    principal_id UUID NOT NULL REFERENCES users(id),
    entity_id UUID NOT NULL REFERENCES entities(id),
    question TEXT NOT NULL,
    answer TEXT NOT NULL,
    citations_json JSONB NOT NULL DEFAULT '[]',
    supporting_entities_json JSONB NOT NULL DEFAULT '[]',
    scope_json JSONB NOT NULL DEFAULT '{}',
    model TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_answer_traces_principal ON answer_traces (principal_id);
CREATE INDEX idx_answer_traces_entity ON answer_traces (entity_id);
CREATE INDEX idx_answer_traces_created ON answer_traces (created_at DESC);
