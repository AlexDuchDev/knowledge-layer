-- Second Brain overlay: delivery links, pre-meeting queue stub, product events for OKR-style metrics.

CREATE TABLE user_chat_links (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    telegram_chat_id TEXT,
    mattermost_user_id TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE pre_meeting_brief_queue (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    domain_id UUID NOT NULL REFERENCES domains(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    dedupe_key TEXT NOT NULL,
    scheduled_for TIMESTAMPTZ NOT NULL,
    payload_json JSONB NOT NULL DEFAULT '{}',
    sent_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pre_meeting_brief_queue_dedupe UNIQUE (dedupe_key)
);

CREATE INDEX pre_meeting_brief_queue_due_idx ON pre_meeting_brief_queue (scheduled_for) WHERE sent_at IS NULL;

CREATE TABLE second_brain_product_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type TEXT NOT NULL,
    actor_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    domain_id UUID REFERENCES domains(id) ON DELETE SET NULL,
    extracted_task_id UUID REFERENCES extracted_meeting_tasks(id) ON DELETE SET NULL,
    payload_json JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX second_brain_product_events_type_created_idx ON second_brain_product_events (event_type, created_at DESC);
CREATE INDEX second_brain_product_events_domain_created_idx ON second_brain_product_events (domain_id, created_at DESC) WHERE domain_id IS NOT NULL;
