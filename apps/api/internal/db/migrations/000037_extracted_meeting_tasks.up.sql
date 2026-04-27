-- Extracted meeting tasks: LLM proposals with human confirm/reject (Second Brain–aligned workflow).
CREATE TABLE extracted_meeting_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    domain_id UUID NOT NULL REFERENCES domains(id) ON DELETE CASCADE,
    source_feed_id UUID REFERENCES source_feeds(id) ON DELETE SET NULL,
    source_normalized_record_id UUID REFERENCES normalized_records(id) ON DELETE SET NULL,
    linked_meeting_entity_id UUID REFERENCES entities(id) ON DELETE SET NULL,
    linked_decision_entity_ids UUID[] NOT NULL DEFAULT '{}',
    participant_refs TEXT[] NOT NULL DEFAULT '{}',
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    assignee_email TEXT,
    assignee_display TEXT,
    deadline_date DATE,
    priority TEXT NOT NULL DEFAULT 'medium',
    review_status TEXT NOT NULL DEFAULT 'draft',
    llm_extraction_version INT NOT NULL DEFAULT 1,
    extraction_metadata_json JSONB NOT NULL DEFAULT '{}',
    confirmed_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    confirmed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT extracted_meeting_tasks_priority_chk CHECK (priority IN ('high', 'medium', 'low')),
    CONSTRAINT extracted_meeting_tasks_review_chk CHECK (review_status IN ('draft', 'confirmed', 'edited', 'rejected'))
);

CREATE INDEX extracted_meeting_tasks_domain_status_idx ON extracted_meeting_tasks (domain_id, review_status);
CREATE INDEX extracted_meeting_tasks_domain_created_idx ON extracted_meeting_tasks (domain_id, created_at DESC);
CREATE INDEX extracted_meeting_tasks_norm_record_idx ON extracted_meeting_tasks (source_normalized_record_id) WHERE source_normalized_record_id IS NOT NULL;

CREATE TABLE extracted_task_review_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    extracted_task_id UUID NOT NULL REFERENCES extracted_meeting_tasks(id) ON DELETE CASCADE,
    actor_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,
    detail_json JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT extracted_task_review_events_type_chk CHECK (event_type IN (
        'created', 'confirm_no_edit', 'confirm_after_edit', 'reject', 'edit_save'
    ))
);

CREATE INDEX extracted_task_review_events_task_idx ON extracted_task_review_events (extracted_task_id, created_at DESC);

-- Optional Mattermost connector (v1: personal access token + channel id per feed).
-- Connector PK takes the next free slot after the wave-2 set (000025: 0x0c..0x10)
-- and the http_url/filesystem entries (000040: 0x11, 0x12). An earlier draft of
-- this migration reused 0x0c (confluence) and broke first-time installs with a
-- primary-key collision before catching the type-conflict clause.
INSERT INTO connectors (id, type, display_name, auth_mode, status) VALUES
    ('20000000-0000-0000-0000-000000000013', 'mattermost', 'Mattermost', 'api_token', 'draft')
ON CONFLICT (type) DO NOTHING;
