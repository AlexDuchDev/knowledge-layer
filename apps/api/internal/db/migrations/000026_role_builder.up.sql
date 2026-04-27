-- Role Builder: extend roles, binding tables, assignment dedupe, preset catalog.

ALTER TABLE roles
    ADD COLUMN IF NOT EXISTS code TEXT,
    ADD COLUMN IF NOT EXISTS category TEXT NOT NULL DEFAULT 'legacy',
    ADD COLUMN IF NOT EXISTS active BOOLEAN NOT NULL DEFAULT true,
    ADD COLUMN IF NOT EXISTS scope_model TEXT NOT NULL DEFAULT 'global',
    ADD COLUMN IF NOT EXISTS is_preset BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS preset_key TEXT,
    ADD COLUMN IF NOT EXISTS cloned_from_role_id UUID REFERENCES roles(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS is_system BOOLEAN NOT NULL DEFAULT false;

UPDATE roles SET code = name WHERE code IS NULL;
ALTER TABLE roles ALTER COLUMN code SET NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_roles_code ON roles (code);
CREATE UNIQUE INDEX IF NOT EXISTS idx_roles_preset_key ON roles (preset_key) WHERE preset_key IS NOT NULL;

UPDATE roles SET is_system = true, category = 'legacy'
WHERE id IN (
    '10000000-0000-0000-0000-000000000001',
    '10000000-0000-0000-0000-000000000002',
    '10000000-0000-0000-0000-000000000003'
);

CREATE TABLE role_domain_bindings (
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    domain_id UUID NOT NULL REFERENCES domains(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, domain_id)
);

CREATE TABLE role_entity_type_bindings (
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    entity_type TEXT NOT NULL,
    PRIMARY KEY (role_id, entity_type)
);

CREATE TABLE role_source_scope_bindings (
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    scope_kind TEXT NOT NULL,
    scope_ref TEXT NOT NULL,
    PRIMARY KEY (role_id, scope_kind, scope_ref)
);

CREATE TABLE role_scenario_bindings (
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    scenario_key TEXT NOT NULL,
    PRIMARY KEY (role_id, scenario_key)
);

CREATE TABLE role_dashboard_bindings (
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    dashboard_key TEXT NOT NULL,
    PRIMARY KEY (role_id, dashboard_key)
);

CREATE TABLE role_job_permissions (
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    knowledge_job_id UUID NOT NULL REFERENCES knowledge_jobs(id) ON DELETE CASCADE,
    can_run BOOLEAN NOT NULL DEFAULT false,
    can_configure BOOLEAN NOT NULL DEFAULT false,
    can_review_job_output BOOLEAN NOT NULL DEFAULT false,
    PRIMARY KEY (role_id, knowledge_job_id)
);

CREATE TABLE role_governance_permissions (
    role_id UUID PRIMARY KEY REFERENCES roles(id) ON DELETE CASCADE,
    can_review_outputs BOOLEAN NOT NULL DEFAULT false,
    can_approve_outputs BOOLEAN NOT NULL DEFAULT false,
    can_publish_outputs BOOLEAN NOT NULL DEFAULT false,
    can_override_policies BOOLEAN NOT NULL DEFAULT false,
    can_manage_assignments BOOLEAN NOT NULL DEFAULT false
);

CREATE UNIQUE INDEX user_role_bindings_user_role_scope_unique
    ON user_role_bindings (user_id, role_id, scope_type, COALESCE(scope_id, '00000000-0000-0000-0000-000000000000'::uuid));

-- Preset roles (clone templates; empty domain/entity bindings = no extra restriction)
INSERT INTO roles (id, code, name, description, category, active, scope_model, is_preset, preset_key, is_system, created_at, updated_at)
VALUES
    ('c1000001-0000-4000-8000-000000000001', 'preset_platform_admin', 'Platform Admin (preset)', 'Full platform operations and governance.', 'platform', true, 'global', true, 'platform_admin', true, now(), now()),
    ('c1000001-0000-4000-8000-000000000002', 'preset_domain_owner', 'Domain Owner (preset)', 'Owns a domain: content, jobs, and member access patterns.', 'domain', true, 'domain', true, 'domain_owner', true, now(), now()),
    ('c1000001-0000-4000-8000-000000000003', 'preset_team_lead', 'Team Lead (preset)', 'Leads a team: review workflow and light access management.', 'operational', true, 'team', true, 'team_lead', true, now(), now()),
    ('c1000001-0000-4000-8000-000000000004', 'preset_reviewer', 'Reviewer (preset)', 'Reviews and comments on governed outputs.', 'domain', true, 'domain', true, 'reviewer', true, now(), now()),
    ('c1000001-0000-4000-8000-000000000005', 'preset_executive_viewer', 'Executive Viewer (preset)', 'Read-only visibility across granted scope.', 'operational', true, 'global', true, 'executive_viewer', true, now(), now()),
    ('c1000001-0000-4000-8000-000000000006', 'preset_support_lead', 'Support Lead (preset)', 'Operational support: sources, jobs, and triage.', 'operational', true, 'domain', true, 'support_lead', true, now(), now()),
    ('c1000001-0000-4000-8000-000000000007', 'preset_finance_restricted', 'Finance Restricted (preset)', 'Narrow finance domain; review-sensitive; assign domain bindings when cloning.', 'restricted_specialist', true, 'domain', true, 'finance_restricted', true, now(), now()),
    ('c1000001-0000-4000-8000-000000000008', 'preset_legal_restricted', 'Legal Restricted (preset)', 'Legal review posture; assign domain bindings when cloning.', 'restricted_specialist', true, 'domain', true, 'legal_restricted', true, now(), now())
ON CONFLICT (id) DO NOTHING;

-- Helper: attach all action codes to a role
INSERT INTO role_action_permissions (id, role_id, action_permission_id)
SELECT gen_random_uuid(), r.id, ap.id
FROM roles r
CROSS JOIN action_permissions ap
WHERE r.preset_key = 'platform_admin'
ON CONFLICT (role_id, action_permission_id) DO NOTHING;

INSERT INTO role_action_permissions (id, role_id, action_permission_id)
SELECT gen_random_uuid(), r.id, ap.id
FROM roles r
CROSS JOIN action_permissions ap
WHERE r.preset_key = 'domain_owner'
  AND ap.code IN (
    'view', 'search', 'view_raw', 'create', 'edit', 'archive', 'export',
    'run_job', 'manage_jobs', 'manage_sources', 'manage_source_feed',
    'review', 'approve', 'publish'
  )
ON CONFLICT (role_id, action_permission_id) DO NOTHING;

INSERT INTO role_action_permissions (id, role_id, action_permission_id)
SELECT gen_random_uuid(), r.id, ap.id
FROM roles r
CROSS JOIN action_permissions ap
WHERE r.preset_key = 'team_lead'
  AND ap.code IN (
    'view', 'search', 'view_raw', 'create', 'edit', 'review', 'approve',
    'run_job', 'manage_jobs', 'export'
  )
ON CONFLICT (role_id, action_permission_id) DO NOTHING;

INSERT INTO role_action_permissions (id, role_id, action_permission_id)
SELECT gen_random_uuid(), r.id, ap.id
FROM roles r
CROSS JOIN action_permissions ap
WHERE r.preset_key = 'reviewer'
  AND ap.code IN ('view', 'search', 'view_raw', 'review')
ON CONFLICT (role_id, action_permission_id) DO NOTHING;

INSERT INTO role_action_permissions (id, role_id, action_permission_id)
SELECT gen_random_uuid(), r.id, ap.id
FROM roles r
CROSS JOIN action_permissions ap
WHERE r.preset_key = 'executive_viewer'
  AND ap.code IN ('view', 'search', 'view_raw')
ON CONFLICT (role_id, action_permission_id) DO NOTHING;

INSERT INTO role_action_permissions (id, role_id, action_permission_id)
SELECT gen_random_uuid(), r.id, ap.id
FROM roles r
CROSS JOIN action_permissions ap
WHERE r.preset_key = 'support_lead'
  AND ap.code IN (
    'view', 'search', 'view_raw', 'create', 'edit', 'run_job', 'manage_jobs',
    'manage_sources', 'manage_source_feed', 'review'
  )
ON CONFLICT (role_id, action_permission_id) DO NOTHING;

INSERT INTO role_action_permissions (id, role_id, action_permission_id)
SELECT gen_random_uuid(), r.id, ap.id
FROM roles r
CROSS JOIN action_permissions ap
WHERE r.preset_key = 'finance_restricted'
  AND ap.code IN ('view', 'search', 'view_raw', 'export', 'review')
ON CONFLICT (role_id, action_permission_id) DO NOTHING;

INSERT INTO role_action_permissions (id, role_id, action_permission_id)
SELECT gen_random_uuid(), r.id, ap.id
FROM roles r
CROSS JOIN action_permissions ap
WHERE r.preset_key = 'legal_restricted'
  AND ap.code IN ('view', 'search', 'view_raw', 'review', 'approve')
ON CONFLICT (role_id, action_permission_id) DO NOTHING;

INSERT INTO role_governance_permissions (
    role_id, can_review_outputs, can_approve_outputs, can_publish_outputs, can_override_policies, can_manage_assignments
)
SELECT id, true, true, true, true, true FROM roles WHERE preset_key = 'platform_admin'
ON CONFLICT (role_id) DO UPDATE SET
    can_review_outputs = EXCLUDED.can_review_outputs,
    can_approve_outputs = EXCLUDED.can_approve_outputs,
    can_publish_outputs = EXCLUDED.can_publish_outputs,
    can_override_policies = EXCLUDED.can_override_policies,
    can_manage_assignments = EXCLUDED.can_manage_assignments;

INSERT INTO role_governance_permissions (
    role_id, can_review_outputs, can_approve_outputs, can_publish_outputs, can_override_policies, can_manage_assignments
)
SELECT id, true, true, true, false, true FROM roles WHERE preset_key = 'domain_owner'
ON CONFLICT (role_id) DO UPDATE SET
    can_review_outputs = EXCLUDED.can_review_outputs,
    can_approve_outputs = EXCLUDED.can_approve_outputs,
    can_publish_outputs = EXCLUDED.can_publish_outputs,
    can_override_policies = EXCLUDED.can_override_policies,
    can_manage_assignments = EXCLUDED.can_manage_assignments;

INSERT INTO role_governance_permissions (
    role_id, can_review_outputs, can_approve_outputs, can_publish_outputs, can_override_policies, can_manage_assignments
)
SELECT id, true, false, false, false, true FROM roles WHERE preset_key = 'team_lead'
ON CONFLICT (role_id) DO UPDATE SET
    can_review_outputs = EXCLUDED.can_review_outputs,
    can_approve_outputs = EXCLUDED.can_approve_outputs,
    can_publish_outputs = EXCLUDED.can_publish_outputs,
    can_override_policies = EXCLUDED.can_override_policies,
    can_manage_assignments = EXCLUDED.can_manage_assignments;

INSERT INTO role_governance_permissions (
    role_id, can_review_outputs, can_approve_outputs, can_publish_outputs, can_override_policies, can_manage_assignments
)
SELECT id, true, false, false, false, false FROM roles WHERE preset_key = 'reviewer'
ON CONFLICT (role_id) DO UPDATE SET
    can_review_outputs = EXCLUDED.can_review_outputs,
    can_approve_outputs = EXCLUDED.can_approve_outputs,
    can_publish_outputs = EXCLUDED.can_publish_outputs,
    can_override_policies = EXCLUDED.can_override_policies,
    can_manage_assignments = EXCLUDED.can_manage_assignments;

INSERT INTO role_governance_permissions (
    role_id, can_review_outputs, can_approve_outputs, can_publish_outputs, can_override_policies, can_manage_assignments
)
SELECT id, false, false, false, false, false FROM roles WHERE preset_key = 'executive_viewer'
ON CONFLICT (role_id) DO UPDATE SET
    can_review_outputs = EXCLUDED.can_review_outputs,
    can_approve_outputs = EXCLUDED.can_approve_outputs,
    can_publish_outputs = EXCLUDED.can_publish_outputs,
    can_override_policies = EXCLUDED.can_override_policies,
    can_manage_assignments = EXCLUDED.can_manage_assignments;

INSERT INTO role_governance_permissions (
    role_id, can_review_outputs, can_approve_outputs, can_publish_outputs, can_override_policies, can_manage_assignments
)
SELECT id, true, false, false, false, false FROM roles WHERE preset_key = 'support_lead'
ON CONFLICT (role_id) DO UPDATE SET
    can_review_outputs = EXCLUDED.can_review_outputs,
    can_approve_outputs = EXCLUDED.can_approve_outputs,
    can_publish_outputs = EXCLUDED.can_publish_outputs,
    can_override_policies = EXCLUDED.can_override_policies,
    can_manage_assignments = EXCLUDED.can_manage_assignments;

INSERT INTO role_governance_permissions (
    role_id, can_review_outputs, can_approve_outputs, can_publish_outputs, can_override_policies, can_manage_assignments
)
SELECT id, true, false, false, false, false FROM roles WHERE preset_key = 'finance_restricted'
ON CONFLICT (role_id) DO UPDATE SET
    can_review_outputs = EXCLUDED.can_review_outputs,
    can_approve_outputs = EXCLUDED.can_approve_outputs,
    can_publish_outputs = EXCLUDED.can_publish_outputs,
    can_override_policies = EXCLUDED.can_override_policies,
    can_manage_assignments = EXCLUDED.can_manage_assignments;

INSERT INTO role_governance_permissions (
    role_id, can_review_outputs, can_approve_outputs, can_publish_outputs, can_override_policies, can_manage_assignments
)
SELECT id, true, true, false, false, false FROM roles WHERE preset_key = 'legal_restricted'
ON CONFLICT (role_id) DO UPDATE SET
    can_review_outputs = EXCLUDED.can_review_outputs,
    can_approve_outputs = EXCLUDED.can_approve_outputs,
    can_publish_outputs = EXCLUDED.can_publish_outputs,
    can_override_policies = EXCLUDED.can_override_policies,
    can_manage_assignments = EXCLUDED.can_manage_assignments;
