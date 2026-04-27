-- Onboarding / setup wizard: templates, resumable sessions, launch logs.

CREATE TABLE onboarding_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    metadata_json JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE onboarding_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'ready', 'launched', 'abandoned')),
    template_code TEXT REFERENCES onboarding_templates(code) ON DELETE SET NULL,
    created_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    org_profile_json JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_onboarding_sessions_status ON onboarding_sessions (status);
CREATE INDEX idx_onboarding_sessions_created_by ON onboarding_sessions (created_by_user_id);

CREATE TABLE onboarding_session_steps (
    session_id UUID NOT NULL REFERENCES onboarding_sessions(id) ON DELETE CASCADE,
    step_key TEXT NOT NULL,
    payload_json JSONB NOT NULL DEFAULT '{}',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (session_id, step_key)
);

CREATE TABLE onboarding_selected_presets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES onboarding_sessions(id) ON DELETE CASCADE,
    preset_catalog_entry_id UUID NOT NULL REFERENCES preset_catalog_entries(id) ON DELETE CASCADE,
    slot TEXT NOT NULL DEFAULT 'default',
    customizations_json JSONB NOT NULL DEFAULT '{}',
    UNIQUE (session_id, preset_catalog_entry_id),
    UNIQUE (session_id, slot)
);

CREATE TABLE onboarding_connector_selections (
    session_id UUID NOT NULL REFERENCES onboarding_sessions(id) ON DELETE CASCADE,
    connector_family_code TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT false,
    PRIMARY KEY (session_id, connector_family_code)
);

CREATE TABLE onboarding_source_feed_drafts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES onboarding_sessions(id) ON DELETE CASCADE,
    draft_json JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE onboarding_assignment_drafts (
    session_id UUID PRIMARY KEY REFERENCES onboarding_sessions(id) ON DELETE CASCADE,
    initial_admin_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    domain_owner_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    assignments_json JSONB NOT NULL DEFAULT '{}',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE onboarding_launch_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES onboarding_sessions(id) ON DELETE CASCADE,
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at TIMESTAMPTZ,
    status TEXT NOT NULL DEFAULT 'running' CHECK (status IN ('running', 'succeeded', 'failed', 'partial')),
    result_json JSONB NOT NULL DEFAULT '{}',
    error_text TEXT
);

CREATE INDEX idx_onboarding_launch_logs_session ON onboarding_launch_logs (session_id);

INSERT INTO onboarding_templates (code, title, description, metadata_json) VALUES
    ('startup_product', 'Startup / Product Team', 'Slack/Telegram, Jira, Notion/Docs, planning and digest jobs.',
     '{"role_codes":["team_lead","reviewer"],"scenario_codes":["ask_allowed_knowledge","weekly_team_digest","planning_summary"],"job_codes":["weekly_digest","planning_summary","decision_extraction"],"connector_families":["slack","telegram","jira","notion"]}'),
    ('agency', 'Agency / Service Business', 'Client projects, docs, memory pages, handoff-oriented workflows.',
     '{"role_codes":["domain_owner","team_lead"],"scenario_codes":["project_memory_page","planning_summary","ask_allowed_knowledge"],"job_codes":["planning_summary","weekly_digest"],"connector_families":["slack","notion"]}'),
    ('support_heavy', 'Support-heavy Company', 'Support trends, governance review, ticket-oriented sources.',
     '{"role_codes":["support_lead","reviewer"],"scenario_codes":["support_trends_digest","governance_review_queue","ask_allowed_knowledge"],"job_codes":["support_trends_extraction","weekly_digest"],"connector_families":["zendesk","slack","intercom"]}'),
    ('leadership_first', 'Leadership-first Setup', 'Executive briefs and consolidation with governed sources.',
     '{"role_codes":["executive_viewer","domain_owner"],"scenario_codes":["executive_weekly_brief","decision_explorer"],"job_codes":["executive_consolidation","decision_extraction"],"connector_families":["notion","google_drive"]}'),
    ('minimal', 'Minimal Setup', 'One admin, core ask scenario, one digest job, minimal connectors.',
     '{"role_codes":["platform_admin"],"scenario_codes":["ask_allowed_knowledge","weekly_team_digest"],"job_codes":["weekly_digest"],"connector_families":["telegram","slack"]}')
ON CONFLICT (code) DO NOTHING;
