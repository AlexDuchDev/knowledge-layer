-- Scenario Builder: definitions, output policies, bindings, presets catalog.

CREATE TABLE scenario_presets (
    preset_key TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    scenario_type TEXT NOT NULL,
    template_json JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE scenario_definitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT,
    scenario_type TEXT NOT NULL,
    active BOOLEAN NOT NULL DEFAULT true,
    target_role_scope_json JSONB NOT NULL DEFAULT '{}',
    input_scope_json JSONB NOT NULL DEFAULT '{}',
    trigger_type TEXT NOT NULL,
    trigger_config_json JSONB NOT NULL DEFAULT '{}',
    processing_mode TEXT NOT NULL,
    output_mode TEXT NOT NULL,
    ui_surface TEXT NOT NULL DEFAULT 'admin_only',
    config_json JSONB NOT NULL DEFAULT '{}',
    preview_config JSONB NOT NULL DEFAULT '{}',
    notes TEXT,
    owner_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    owner_team_id UUID REFERENCES teams(id) ON DELETE SET NULL,
    is_preset BOOLEAN NOT NULL DEFAULT false,
    preset_key TEXT UNIQUE,
    cloned_from_scenario_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT scenario_definitions_cloned_from_fk
        FOREIGN KEY (cloned_from_scenario_id) REFERENCES scenario_definitions(id) ON DELETE SET NULL
);

CREATE INDEX idx_scenario_definitions_active_type ON scenario_definitions (active, scenario_type);
CREATE INDEX idx_scenario_definitions_is_preset ON scenario_definitions (is_preset) WHERE is_preset = true;

CREATE TABLE scenario_output_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scenario_id UUID NOT NULL UNIQUE REFERENCES scenario_definitions(id) ON DELETE CASCADE,
    output_domain_id UUID REFERENCES domains(id) ON DELETE SET NULL,
    output_sensitivity_level INT NOT NULL DEFAULT 0,
    review_required BOOLEAN NOT NULL DEFAULT false,
    publication_mode TEXT NOT NULL DEFAULT 'draft',
    citations_required BOOLEAN NOT NULL DEFAULT false,
    provenance_required BOOLEAN NOT NULL DEFAULT false,
    extra_json JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE scenario_definitions
    ADD COLUMN output_policy_id UUID REFERENCES scenario_output_policies(id) ON DELETE SET NULL;

CREATE TABLE scenario_role_bindings (
    scenario_id UUID NOT NULL REFERENCES scenario_definitions(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    can_see BOOLEAN NOT NULL DEFAULT false,
    can_run BOOLEAN NOT NULL DEFAULT false,
    can_manage BOOLEAN NOT NULL DEFAULT false,
    can_review_publish BOOLEAN NOT NULL DEFAULT false,
    PRIMARY KEY (scenario_id, role_id)
);

CREATE TABLE scenario_source_bindings (
    scenario_id UUID NOT NULL REFERENCES scenario_definitions(id) ON DELETE CASCADE,
    source_feed_id UUID NOT NULL REFERENCES source_feeds(id) ON DELETE CASCADE,
    binding_role TEXT NOT NULL DEFAULT 'primary',
    PRIMARY KEY (scenario_id, source_feed_id)
);

CREATE TABLE scenario_job_bindings (
    scenario_id UUID NOT NULL REFERENCES scenario_definitions(id) ON DELETE CASCADE,
    knowledge_job_id UUID NOT NULL REFERENCES knowledge_jobs(id) ON DELETE CASCADE,
    relationship TEXT NOT NULL DEFAULT 'supports',
    PRIMARY KEY (scenario_id, knowledge_job_id),
    CONSTRAINT scenario_job_bindings_relationship_chk CHECK (relationship IN ('primary_support', 'supports', 'optional'))
);

CREATE TABLE scenario_ui_bindings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scenario_id UUID NOT NULL REFERENCES scenario_definitions(id) ON DELETE CASCADE,
    surface_key TEXT NOT NULL,
    nav_group TEXT,
    sort_order INT NOT NULL DEFAULT 0,
    config_json JSONB NOT NULL DEFAULT '{}',
    UNIQUE (scenario_id, surface_key)
);

CREATE INDEX idx_scenario_ui_bindings_scenario ON scenario_ui_bindings (scenario_id);

-- Preset catalog rows (templates for clone-from-preset)
INSERT INTO scenario_presets (preset_key, name, description, scenario_type, template_json) VALUES
('ask_allowed_knowledge', 'Ask over my allowed knowledge', 'Interactive Q&A scoped to the caller''s permitted knowledge; citations and trace required.', 'ask',
 '{"trigger_type":"interactive","trigger_config_json":{},"processing_mode":"ask","output_mode":"ui_response","ui_surface":"ask","input_scope_json":{"inherit_user_retrieval_scope":true},"target_role_scope_json":{"audience":"all_authenticated"},"output_policy":{"citations_required":true,"provenance_required":true,"review_required":false,"publication_mode":"draft","output_sensitivity_level":0}}'),
('weekly_team_digest', 'Weekly team digest', 'Recurring summary of team activity and blockers.', 'digest',
 '{"trigger_type":"scheduled","trigger_config_json":{"schedule_expr":"0 9 * * MON","timezone":"UTC"},"processing_mode":"summarize","output_mode":"digest_entity","ui_surface":"dashboard","input_scope_json":{"time_window":"last_7d","requires_explicit_scope":true},"target_role_scope_json":{"audience":"team"},"output_policy":{"citations_required":true,"provenance_required":true,"review_required":true,"publication_mode":"draft","output_sensitivity_level":1}}'),
('planning_summary', 'Planning summary', 'After planning, consolidate commitments and risks.', 'process',
 '{"trigger_type":"event_driven","trigger_config_json":{"event_types":["planning_session_completed"]},"processing_mode":"consolidate","output_mode":"summary_entity","ui_surface":"workflows","input_scope_json":{"entity_types":["decision","task"],"requires_explicit_scope":true},"target_role_scope_json":{"audience":"team"},"output_policy":{"citations_required":true,"provenance_required":true,"review_required":true,"publication_mode":"draft","output_sensitivity_level":1}}'),
('retro_summary', 'Retro summary', 'Extract lessons learned after retrospectives.', 'process',
 '{"trigger_type":"manual","trigger_config_json":{},"processing_mode":"extract","output_mode":"summary_entity","ui_surface":"workflows","input_scope_json":{"entity_types":["meeting","decision"],"time_window":"last_14d","requires_explicit_scope":true},"target_role_scope_json":{"audience":"team"},"output_policy":{"citations_required":true,"provenance_required":true,"review_required":true,"publication_mode":"draft","output_sensitivity_level":1}}'),
('project_memory_page', 'Project memory page', 'Structured explorer for project-linked knowledge.', 'explorer',
 '{"trigger_type":"interactive","trigger_config_json":{},"processing_mode":"explore","output_mode":"explorer_view","ui_surface":"knowledge","input_scope_json":{"entity_types":["project","document"],"requires_explicit_scope":true},"target_role_scope_json":{"audience":"project_members"},"output_policy":{"citations_required":false,"provenance_required":true,"review_required":false,"publication_mode":"draft","output_sensitivity_level":0}}'),
('decision_explorer', 'Decision explorer', 'Navigate decisions and linked evidence.', 'explorer',
 '{"trigger_type":"interactive","trigger_config_json":{},"processing_mode":"explore","output_mode":"explorer_view","ui_surface":"knowledge","input_scope_json":{"entity_types":["decision"],"requires_explicit_scope":true},"target_role_scope_json":{"audience":"domain_members"},"output_policy":{"citations_required":false,"provenance_required":true,"review_required":false,"publication_mode":"draft","output_sensitivity_level":0}}'),
('executive_weekly_brief', 'Executive weekly brief', 'Leadership-oriented weekly brief with elevated sensitivity.', 'digest',
 '{"trigger_type":"scheduled","trigger_config_json":{"schedule_expr":"0 7 * * MON","timezone":"UTC"},"processing_mode":"summarize","output_mode":"summary_entity","ui_surface":"dashboard","input_scope_json":{"time_window":"last_7d","requires_explicit_scope":true,"leadership_scope":true},"target_role_scope_json":{"audience":"leadership"},"output_policy":{"citations_required":true,"provenance_required":true,"review_required":true,"publication_mode":"draft","output_sensitivity_level":3,"extra_json":{"sensitivity_label":"leadership_restricted"}}}'),
('support_trends_digest', 'Support trends digest', 'Trends and themes from support channels.', 'digest',
 '{"trigger_type":"scheduled","trigger_config_json":{"schedule_expr":"0 8 * * *","timezone":"UTC"},"processing_mode":"detect","output_mode":"dashboard_block","ui_surface":"dashboard","input_scope_json":{"source_categories":["support","crm"],"requires_explicit_scope":true},"target_role_scope_json":{"audience":"support_ops"},"output_policy":{"citations_required":true,"provenance_required":true,"review_required":true,"publication_mode":"draft","output_sensitivity_level":1}}'),
('governance_review_queue', 'Governance review queue', 'Operational queue for stale content, approvals, and failed jobs.', 'governance',
 '{"trigger_type":"interactive","trigger_config_json":{},"processing_mode":"explore","output_mode":"review_task","ui_surface":"governance","input_scope_json":{"governance_views":["stale_content","pending_approvals","failed_jobs"],"requires_explicit_scope":true},"target_role_scope_json":{"audience":"governance_operators"},"output_policy":{"citations_required":false,"provenance_required":true,"review_required":false,"publication_mode":"draft","output_sensitivity_level":0}}');

-- Seed system preset scenarios (mirror catalog; ids fixed for stable references)
INSERT INTO scenario_definitions (id, code, name, description, scenario_type, active, target_role_scope_json, input_scope_json, trigger_type, trigger_config_json, processing_mode, output_mode, ui_surface, config_json, preview_config, is_preset, preset_key) VALUES
('f1000001-0000-4000-8000-000000000001', 'ask_allowed_knowledge', 'Ask over my allowed knowledge', 'Interactive Q&A scoped to permitted knowledge.', 'ask', true,
 '{"audience":"all_authenticated"}'::jsonb, '{"inherit_user_retrieval_scope":true}'::jsonb, 'interactive', '{}'::jsonb, 'ask', 'ui_response', 'ask', '{}'::jsonb, '{}'::jsonb, true, 'ask_allowed_knowledge'),
('f1000001-0000-4000-8000-000000000002', 'weekly_team_digest', 'Weekly team digest', 'Recurring team summary.', 'digest', true,
 '{"audience":"team"}'::jsonb, '{"time_window":"last_7d","requires_explicit_scope":true}'::jsonb, 'scheduled', '{"schedule_expr":"0 9 * * MON","timezone":"UTC"}'::jsonb, 'summarize', 'digest_entity', 'dashboard', '{}'::jsonb, '{}'::jsonb, true, 'weekly_team_digest'),
('f1000001-0000-4000-8000-000000000003', 'planning_summary', 'Planning summary', 'Post-planning consolidation.', 'process', true,
 '{"audience":"team"}'::jsonb, '{"entity_types":["decision","task"],"requires_explicit_scope":true}'::jsonb, 'event_driven', '{"event_types":["planning_session_completed"]}'::jsonb, 'consolidate', 'summary_entity', 'workflows', '{}'::jsonb, '{}'::jsonb, true, 'planning_summary'),
('f1000001-0000-4000-8000-000000000004', 'retro_summary', 'Retro summary', 'Retro lessons learned.', 'process', true,
 '{"audience":"team"}'::jsonb, '{"entity_types":["meeting","decision"],"time_window":"last_14d","requires_explicit_scope":true}'::jsonb, 'manual', '{}'::jsonb, 'extract', 'summary_entity', 'workflows', '{}'::jsonb, '{}'::jsonb, true, 'retro_summary'),
('f1000001-0000-4000-8000-000000000005', 'project_memory_page', 'Project memory page', 'Project knowledge explorer.', 'explorer', true,
 '{"audience":"project_members"}'::jsonb, '{"entity_types":["project","document"],"requires_explicit_scope":true}'::jsonb, 'interactive', '{}'::jsonb, 'explore', 'explorer_view', 'knowledge', '{}'::jsonb, '{}'::jsonb, true, 'project_memory_page'),
('f1000001-0000-4000-8000-000000000006', 'decision_explorer', 'Decision explorer', 'Decision navigation.', 'explorer', true,
 '{"audience":"domain_members"}'::jsonb, '{"entity_types":["decision"],"requires_explicit_scope":true}'::jsonb, 'interactive', '{}'::jsonb, 'explore', 'explorer_view', 'knowledge', '{}'::jsonb, '{}'::jsonb, true, 'decision_explorer'),
('f1000001-0000-4000-8000-000000000007', 'executive_weekly_brief', 'Executive weekly brief', 'Leadership weekly brief.', 'digest', true,
 '{"audience":"leadership"}'::jsonb, '{"time_window":"last_7d","requires_explicit_scope":true,"leadership_scope":true}'::jsonb, 'scheduled', '{"schedule_expr":"0 7 * * MON","timezone":"UTC"}'::jsonb, 'summarize', 'summary_entity', 'dashboard', '{}'::jsonb, '{}'::jsonb, true, 'executive_weekly_brief'),
('f1000001-0000-4000-8000-000000000008', 'support_trends_digest', 'Support trends digest', 'Support trends and themes.', 'digest', true,
 '{"audience":"support_ops"}'::jsonb, '{"source_categories":["support","crm"],"requires_explicit_scope":true}'::jsonb, 'scheduled', '{"schedule_expr":"0 8 * * *","timezone":"UTC"}'::jsonb, 'detect', 'dashboard_block', 'dashboard', '{}'::jsonb, '{}'::jsonb, true, 'support_trends_digest'),
('f1000001-0000-4000-8000-000000000009', 'governance_review_queue', 'Governance review queue', 'Governance operational queue.', 'governance', true,
 '{"audience":"governance_operators"}'::jsonb, '{"governance_views":["stale_content","pending_approvals","failed_jobs"],"requires_explicit_scope":true}'::jsonb, 'interactive', '{}'::jsonb, 'explore', 'review_task', 'governance', '{}'::jsonb, '{}'::jsonb, true, 'governance_review_queue');

INSERT INTO scenario_output_policies (id, scenario_id, output_domain_id, output_sensitivity_level, review_required, publication_mode, citations_required, provenance_required, extra_json) VALUES
('f2000001-0000-4000-8000-000000000001', 'f1000001-0000-4000-8000-000000000001', NULL, 0, false, 'draft', true, true, '{}'),
('f2000001-0000-4000-8000-000000000002', 'f1000001-0000-4000-8000-000000000002', NULL, 1, true, 'draft', true, true, '{}'),
('f2000001-0000-4000-8000-000000000003', 'f1000001-0000-4000-8000-000000000003', NULL, 1, true, 'draft', true, true, '{}'),
('f2000001-0000-4000-8000-000000000004', 'f1000001-0000-4000-8000-000000000004', NULL, 1, true, 'draft', true, true, '{}'),
('f2000001-0000-4000-8000-000000000005', 'f1000001-0000-4000-8000-000000000005', NULL, 0, false, 'draft', false, true, '{}'),
('f2000001-0000-4000-8000-000000000006', 'f1000001-0000-4000-8000-000000000006', NULL, 0, false, 'draft', false, true, '{}'),
('f2000001-0000-4000-8000-000000000007', 'f1000001-0000-4000-8000-000000000007', NULL, 3, true, 'draft', true, true, '{"sensitivity_label":"leadership_restricted"}'),
('f2000001-0000-4000-8000-000000000008', 'f1000001-0000-4000-8000-000000000008', NULL, 1, true, 'draft', true, true, '{}'),
('f2000001-0000-4000-8000-000000000009', 'f1000001-0000-4000-8000-000000000009', NULL, 0, false, 'draft', false, true, '{}');

UPDATE scenario_definitions d SET output_policy_id = p.id
FROM scenario_output_policies p WHERE p.scenario_id = d.id;

INSERT INTO scenario_ui_bindings (id, scenario_id, surface_key, nav_group, sort_order, config_json) VALUES
(gen_random_uuid(), 'f1000001-0000-4000-8000-000000000001', 'ask', 'Knowledge', 10, '{}'),
(gen_random_uuid(), 'f1000001-0000-4000-8000-000000000002', 'dashboard_digest', 'Workflows', 20, '{}'),
(gen_random_uuid(), 'f1000001-0000-4000-8000-000000000003', 'workflow_planning', 'Workflows', 30, '{}'),
(gen_random_uuid(), 'f1000001-0000-4000-8000-000000000004', 'workflow_retro', 'Workflows', 40, '{}'),
(gen_random_uuid(), 'f1000001-0000-4000-8000-000000000005', 'explorer_project', 'Knowledge', 50, '{}'),
(gen_random_uuid(), 'f1000001-0000-4000-8000-000000000006', 'explorer_decisions', 'Knowledge', 60, '{}'),
(gen_random_uuid(), 'f1000001-0000-4000-8000-000000000007', 'dashboard_executive', 'Workflows', 70, '{}'),
(gen_random_uuid(), 'f1000001-0000-4000-8000-000000000008', 'dashboard_support', 'Workflows', 80, '{}'),
(gen_random_uuid(), 'f1000001-0000-4000-8000-000000000009', 'governance_queue', 'Governance', 90, '{}');
