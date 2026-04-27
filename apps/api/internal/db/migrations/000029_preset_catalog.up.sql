-- Preset catalog: unified entries, categories, relationships, instantiation audit.
-- Provenance columns on live objects (no FK to catalog; audit + optional denormalized code).

ALTER TABLE scenario_definitions
    ADD COLUMN IF NOT EXISTS source_preset_code TEXT;

ALTER TABLE knowledge_jobs
    ADD COLUMN IF NOT EXISTS source_preset_code TEXT;

ALTER TABLE roles
    ADD COLUMN IF NOT EXISTS source_preset_code TEXT;

CREATE INDEX IF NOT EXISTS idx_scenario_definitions_source_preset
    ON scenario_definitions (source_preset_code) WHERE source_preset_code IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_knowledge_jobs_source_preset
    ON knowledge_jobs (source_preset_code) WHERE source_preset_code IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_roles_source_preset
    ON roles (source_preset_code) WHERE source_preset_code IS NOT NULL;

CREATE TABLE preset_categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    axis TEXT NOT NULL CHECK (axis IN ('function', 'usage_mode', 'maturity', 'object_type')),
    code TEXT NOT NULL,
    label TEXT NOT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (axis, code)
);

CREATE TABLE preset_catalog_entries (
    id UUID PRIMARY KEY,
    preset_type TEXT NOT NULL CHECK (preset_type IN ('role', 'scenario', 'job')),
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    active BOOLEAN NOT NULL DEFAULT true,
    metadata_json JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (preset_type, code)
);

CREATE TABLE preset_catalog_category_assignments (
    preset_catalog_entry_id UUID NOT NULL REFERENCES preset_catalog_entries(id) ON DELETE CASCADE,
    category_id UUID NOT NULL REFERENCES preset_categories(id) ON DELETE CASCADE,
    PRIMARY KEY (preset_catalog_entry_id, category_id)
);

CREATE TABLE preset_relationships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    from_preset_id UUID NOT NULL REFERENCES preset_catalog_entries(id) ON DELETE CASCADE,
    to_preset_id UUID NOT NULL REFERENCES preset_catalog_entries(id) ON DELETE CASCADE,
    relationship_type TEXT NOT NULL,
    metadata_json JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT preset_relationships_no_self CHECK (from_preset_id <> to_preset_id),
    UNIQUE (from_preset_id, to_preset_id, relationship_type)
);

CREATE INDEX idx_preset_relationships_from ON preset_relationships (from_preset_id);
CREATE INDEX idx_preset_relationships_to ON preset_relationships (to_preset_id);

CREATE TABLE preset_instantiation_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    preset_catalog_entry_id UUID REFERENCES preset_catalog_entries(id) ON DELETE SET NULL,
    principal_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    target_kind TEXT NOT NULL CHECK (target_kind IN ('role', 'scenario', 'job')),
    target_id UUID NOT NULL,
    payload_json JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_preset_instantiation_logs_entry ON preset_instantiation_logs (preset_catalog_entry_id);
CREATE INDEX idx_preset_instantiation_logs_created ON preset_instantiation_logs (created_at DESC);

-- Categories (axes)
INSERT INTO preset_categories (id, axis, code, label, sort_order) VALUES
    ('d1000001-0000-4000-8000-000000000001', 'function', 'leadership', 'Leadership', 10),
    ('d1000001-0000-4000-8000-000000000002', 'function', 'product', 'Product', 20),
    ('d1000001-0000-4000-8000-000000000003', 'function', 'engineering', 'Engineering', 30),
    ('d1000001-0000-4000-8000-000000000004', 'function', 'operations', 'Operations', 40),
    ('d1000001-0000-4000-8000-000000000005', 'function', 'support', 'Support', 50),
    ('d1000001-0000-4000-8000-000000000006', 'function', 'finance', 'Finance', 60),
    ('d1000001-0000-4000-8000-000000000007', 'function', 'legal', 'Legal', 70),
    ('d1000001-0000-4000-8000-000000000008', 'function', 'cross_functional', 'Cross-functional', 80),
    ('d1000001-0000-4000-8000-000000000010', 'usage_mode', 'ask', 'Ask', 10),
    ('d1000001-0000-4000-8000-000000000011', 'usage_mode', 'digest', 'Digest', 20),
    ('d1000001-0000-4000-8000-000000000012', 'usage_mode', 'process', 'Process', 30),
    ('d1000001-0000-4000-8000-000000000013', 'usage_mode', 'explorer', 'Explorer', 40),
    ('d1000001-0000-4000-8000-000000000014', 'usage_mode', 'governance', 'Governance', 50),
    ('d1000001-0000-4000-8000-000000000020', 'maturity', 'starter', 'Starter', 10),
    ('d1000001-0000-4000-8000-000000000021', 'maturity', 'standard', 'Standard', 20),
    ('d1000001-0000-4000-8000-000000000022', 'maturity', 'advanced', 'Advanced', 30),
    ('d1000001-0000-4000-8000-000000000030', 'object_type', 'role', 'Role', 10),
    ('d1000001-0000-4000-8000-000000000031', 'object_type', 'scenario', 'Scenario', 20),
    ('d1000001-0000-4000-8000-000000000032', 'object_type', 'job', 'Job', 30)
ON CONFLICT (axis, code) DO NOTHING;

-- Role catalog entries (codes match roles.preset_key)
INSERT INTO preset_catalog_entries (id, preset_type, code, name, description, active, metadata_json) VALUES
    ('b1000001-0000-4000-8000-000000000001', 'role', 'platform_admin', 'Platform Admin', 'Full platform operations and governance.', true, '{}'),
    ('b1000001-0000-4000-8000-000000000002', 'role', 'domain_owner', 'Domain Owner', 'Owns a domain: content, jobs, and member access patterns.', true, '{}'),
    ('b1000001-0000-4000-8000-000000000003', 'role', 'team_lead', 'Team Lead', 'Leads a team: review workflow and light access management.', true, '{}'),
    ('b1000001-0000-4000-8000-000000000004', 'role', 'reviewer', 'Reviewer', 'Reviews and comments on governed outputs.', true, '{}'),
    ('b1000001-0000-4000-8000-000000000005', 'role', 'executive_viewer', 'Executive Viewer', 'Read-only visibility across granted scope.', true, '{}'),
    ('b1000001-0000-4000-8000-000000000006', 'role', 'support_lead', 'Support Lead', 'Operational support: sources, jobs, and triage.', true, '{}'),
    ('b1000001-0000-4000-8000-000000000007', 'role', 'finance_restricted', 'Finance Restricted', 'Narrow finance domain; review-sensitive.', true, '{}'),
    ('b1000001-0000-4000-8000-000000000008', 'role', 'legal_restricted', 'Legal Restricted', 'Legal review posture; assign domain bindings when cloning.', true, '{}')
ON CONFLICT (preset_type, code) DO NOTHING;

-- Scenario catalog (codes match scenario_presets.preset_key)
INSERT INTO preset_catalog_entries (id, preset_type, code, name, description, active, metadata_json) VALUES
    ('b1000002-0000-4000-8000-000000000001', 'scenario', 'ask_allowed_knowledge', 'Ask over my allowed knowledge', 'Interactive Q&A scoped to permitted knowledge.', true, '{}'),
    ('b1000002-0000-4000-8000-000000000002', 'scenario', 'weekly_team_digest', 'Weekly team digest', 'Recurring summary of team activity and blockers.', true, '{}'),
    ('b1000002-0000-4000-8000-000000000003', 'scenario', 'planning_summary', 'Planning summary', 'After planning, consolidate commitments and risks.', true, '{}'),
    ('b1000002-0000-4000-8000-000000000004', 'scenario', 'retro_summary', 'Retro summary', 'Extract lessons learned after retrospectives.', true, '{}'),
    ('b1000002-0000-4000-8000-000000000005', 'scenario', 'project_memory_page', 'Project memory page', 'Structured explorer for project-linked knowledge.', true, '{}'),
    ('b1000002-0000-4000-8000-000000000006', 'scenario', 'decision_explorer', 'Decision explorer', 'Navigate decisions and linked evidence.', true, '{}'),
    ('b1000002-0000-4000-8000-000000000007', 'scenario', 'executive_weekly_brief', 'Executive weekly brief', 'Leadership-oriented weekly brief.', true, '{}'),
    ('b1000002-0000-4000-8000-000000000008', 'scenario', 'support_trends_digest', 'Support trends digest', 'Trends and themes from support channels.', true, '{}'),
    ('b1000002-0000-4000-8000-000000000009', 'scenario', 'governance_review_queue', 'Governance review queue', 'Operational queue for stale content and approvals.', true, '{}')
ON CONFLICT (preset_type, code) DO NOTHING;

-- Job catalog (codes match job_builder_presets.preset_key)
INSERT INTO preset_catalog_entries (id, preset_type, code, name, description, active, metadata_json) VALUES
    ('b1000003-0000-4000-8000-000000000001', 'job', 'weekly_digest', 'Weekly Digest', 'Roll up feed activity into a governed digest with review.', true, '{}'),
    ('b1000003-0000-4000-8000-000000000002', 'job', 'planning_summary', 'Planning Summary', 'Synthesize planning horizon, commitments, and risks.', true, '{}'),
    ('b1000003-0000-4000-8000-000000000003', 'job', 'decision_extraction', 'Decision Extraction', 'Extract explicit decisions, owners, and rationale.', true, '{}'),
    ('b1000003-0000-4000-8000-000000000004', 'job', 'blocker_detection', 'Blocker Detection', 'Detect blockers and dependencies from operational signals.', true, '{}'),
    ('b1000003-0000-4000-8000-000000000005', 'job', 'incident_summary', 'Incident Summary', 'Post-incident structured summary and follow-ups.', true, '{}'),
    ('b1000003-0000-4000-8000-000000000006', 'job', 'executive_consolidation', 'Executive Consolidation', 'Multi-source executive brief with citations.', true, '{}'),
    ('b1000003-0000-4000-8000-000000000007', 'job', 'support_trends_extraction', 'Support Trends Extraction', 'Trends and themes from support interactions.', true, '{}'),
    ('b1000003-0000-4000-8000-000000000008', 'job', 'retro_summary', 'Retro Summary', 'Retrospective themes, actions, and ownership.', true, '{}')
ON CONFLICT (preset_type, code) DO NOTHING;

-- Category assignments: object_type for all; function/usage/maturity samples
INSERT INTO preset_catalog_category_assignments (preset_catalog_entry_id, category_id)
SELECT e.id, c.id FROM preset_catalog_entries e
CROSS JOIN preset_categories c
WHERE e.preset_type = 'role' AND c.axis = 'object_type' AND c.code = 'role'
ON CONFLICT DO NOTHING;

INSERT INTO preset_catalog_category_assignments (preset_catalog_entry_id, category_id)
SELECT e.id, c.id FROM preset_catalog_entries e
CROSS JOIN preset_categories c
WHERE e.preset_type = 'scenario' AND c.axis = 'object_type' AND c.code = 'scenario'
ON CONFLICT DO NOTHING;

INSERT INTO preset_catalog_category_assignments (preset_catalog_entry_id, category_id)
SELECT e.id, c.id FROM preset_catalog_entries e
CROSS JOIN preset_categories c
WHERE e.preset_type = 'job' AND c.axis = 'object_type' AND c.code = 'job'
ON CONFLICT DO NOTHING;

-- Team lead: leadership + standard
INSERT INTO preset_catalog_category_assignments (preset_catalog_entry_id, category_id)
SELECT 'b1000001-0000-4000-8000-000000000003', id FROM preset_categories WHERE axis = 'function' AND code = 'leadership'
ON CONFLICT DO NOTHING;
INSERT INTO preset_catalog_category_assignments (preset_catalog_entry_id, category_id)
SELECT 'b1000001-0000-4000-8000-000000000003', id FROM preset_categories WHERE axis = 'maturity' AND code = 'standard'
ON CONFLICT DO NOTHING;

-- Executive viewer: leadership + standard
INSERT INTO preset_catalog_category_assignments (preset_catalog_entry_id, category_id)
SELECT 'b1000001-0000-4000-8000-000000000005', id FROM preset_categories WHERE axis = 'function' AND code = 'leadership'
ON CONFLICT DO NOTHING;
INSERT INTO preset_catalog_category_assignments (preset_catalog_entry_id, category_id)
SELECT 'b1000001-0000-4000-8000-000000000005', id FROM preset_categories WHERE axis = 'maturity' AND code = 'standard'
ON CONFLICT DO NOTHING;

-- Reviewer: governance function + standard
INSERT INTO preset_catalog_category_assignments (preset_catalog_entry_id, category_id)
SELECT 'b1000001-0000-4000-8000-000000000004', id FROM preset_categories WHERE axis = 'function' AND code = 'cross_functional'
ON CONFLICT DO NOTHING;
INSERT INTO preset_catalog_category_assignments (preset_catalog_entry_id, category_id)
SELECT 'b1000001-0000-4000-8000-000000000004', id FROM preset_categories WHERE axis = 'maturity' AND code = 'standard'
ON CONFLICT DO NOTHING;

-- Support lead: support + operations
INSERT INTO preset_catalog_category_assignments (preset_catalog_entry_id, category_id)
SELECT 'b1000001-0000-4000-8000-000000000006', id FROM preset_categories WHERE axis = 'function' AND code = 'support'
ON CONFLICT DO NOTHING;

-- Finance / legal restricted
INSERT INTO preset_catalog_category_assignments (preset_catalog_entry_id, category_id)
SELECT 'b1000001-0000-4000-8000-000000000007', id FROM preset_categories WHERE axis = 'function' AND code = 'finance'
ON CONFLICT DO NOTHING;
INSERT INTO preset_catalog_category_assignments (preset_catalog_entry_id, category_id)
SELECT 'b1000001-0000-4000-8000-000000000008', id FROM preset_categories WHERE axis = 'function' AND code = 'legal'
ON CONFLICT DO NOTHING;

-- Platform admin / domain owner: cross_functional + advanced
INSERT INTO preset_catalog_category_assignments (preset_catalog_entry_id, category_id)
SELECT 'b1000001-0000-4000-8000-000000000001', id FROM preset_categories WHERE axis = 'maturity' AND code = 'advanced'
ON CONFLICT DO NOTHING;
INSERT INTO preset_catalog_category_assignments (preset_catalog_entry_id, category_id)
SELECT 'b1000001-0000-4000-8000-000000000002', id FROM preset_categories WHERE axis = 'maturity' AND code = 'advanced'
ON CONFLICT DO NOTHING;

-- Scenario usage_mode assignments
INSERT INTO preset_catalog_category_assignments (preset_catalog_entry_id, category_id)
SELECT 'b1000002-0000-4000-8000-000000000001', id FROM preset_categories WHERE axis = 'usage_mode' AND code = 'ask'
ON CONFLICT DO NOTHING;
INSERT INTO preset_catalog_category_assignments (preset_catalog_entry_id, category_id)
SELECT 'b1000002-0000-4000-8000-000000000001', id FROM preset_categories WHERE axis = 'maturity' AND code = 'starter'
ON CONFLICT DO NOTHING;

INSERT INTO preset_catalog_category_assignments (preset_catalog_entry_id, category_id)
SELECT e.id, c.id FROM preset_catalog_entries e, preset_categories c
WHERE e.id IN (
    'b1000002-0000-4000-8000-000000000002',
    'b1000002-0000-4000-8000-000000000007',
    'b1000002-0000-4000-8000-000000000008'
) AND c.axis = 'usage_mode' AND c.code = 'digest'
ON CONFLICT DO NOTHING;

INSERT INTO preset_catalog_category_assignments (preset_catalog_entry_id, category_id)
SELECT e.id, c.id FROM preset_catalog_entries e, preset_categories c
WHERE e.id IN (
    'b1000002-0000-4000-8000-000000000003',
    'b1000002-0000-4000-8000-000000000004'
) AND c.axis = 'usage_mode' AND c.code = 'process'
ON CONFLICT DO NOTHING;

INSERT INTO preset_catalog_category_assignments (preset_catalog_entry_id, category_id)
SELECT e.id, c.id FROM preset_catalog_entries e, preset_categories c
WHERE e.id IN (
    'b1000002-0000-4000-8000-000000000005',
    'b1000002-0000-4000-8000-000000000006'
) AND c.axis = 'usage_mode' AND c.code = 'explorer'
ON CONFLICT DO NOTHING;

INSERT INTO preset_catalog_category_assignments (preset_catalog_entry_id, category_id)
SELECT 'b1000002-0000-4000-8000-000000000009', id FROM preset_categories WHERE axis = 'usage_mode' AND code = 'governance'
ON CONFLICT DO NOTHING;

-- Jobs: standard maturity for most
INSERT INTO preset_catalog_category_assignments (preset_catalog_entry_id, category_id)
SELECT e.id, c.id FROM preset_catalog_entries e, preset_categories c
WHERE e.preset_type = 'job' AND c.axis = 'maturity' AND c.code = 'standard'
ON CONFLICT DO NOTHING;

-- Relationships: team_lead -> scenarios and jobs
INSERT INTO preset_relationships (from_preset_id, to_preset_id, relationship_type) VALUES
    ('b1000001-0000-4000-8000-000000000003', 'b1000002-0000-4000-8000-000000000002', 'role_recommends_scenario'),
    ('b1000001-0000-4000-8000-000000000003', 'b1000002-0000-4000-8000-000000000003', 'role_recommends_scenario'),
    ('b1000001-0000-4000-8000-000000000003', 'b1000003-0000-4000-8000-000000000001', 'role_recommends_job'),
    ('b1000001-0000-4000-8000-000000000003', 'b1000003-0000-4000-8000-000000000002', 'role_recommends_job'),
    ('b1000001-0000-4000-8000-000000000003', 'b1000003-0000-4000-8000-000000000004', 'role_recommends_job'),
    ('b1000001-0000-4000-8000-000000000005', 'b1000002-0000-4000-8000-000000000007', 'role_recommends_scenario'),
    ('b1000001-0000-4000-8000-000000000005', 'b1000003-0000-4000-8000-000000000006', 'role_recommends_job'),
    ('b1000001-0000-4000-8000-000000000004', 'b1000002-0000-4000-8000-000000000009', 'role_recommends_scenario')
ON CONFLICT DO NOTHING;

-- Scenario -> recommended jobs (digest scenarios -> digest jobs)
INSERT INTO preset_relationships (from_preset_id, to_preset_id, relationship_type) VALUES
    ('b1000002-0000-4000-8000-000000000002', 'b1000003-0000-4000-8000-000000000001', 'scenario_recommends_job'),
    ('b1000002-0000-4000-8000-000000000003', 'b1000003-0000-4000-8000-000000000002', 'scenario_recommends_job'),
    ('b1000002-0000-4000-8000-000000000004', 'b1000003-0000-4000-8000-000000000008', 'scenario_recommends_job'),
    ('b1000002-0000-4000-8000-000000000007', 'b1000003-0000-4000-8000-000000000006', 'scenario_recommends_job'),
    ('b1000002-0000-4000-8000-000000000008', 'b1000003-0000-4000-8000-000000000007', 'scenario_recommends_job')
ON CONFLICT DO NOTHING;

-- Job pairs with scenario (reverse hint)
INSERT INTO preset_relationships (from_preset_id, to_preset_id, relationship_type) VALUES
    ('b1000003-0000-4000-8000-000000000003', 'b1000002-0000-4000-8000-000000000006', 'job_pairs_with_scenario')
ON CONFLICT DO NOTHING;
