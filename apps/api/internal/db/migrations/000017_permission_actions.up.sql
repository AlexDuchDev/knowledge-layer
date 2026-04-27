-- Additional canonical actions for governed organizational memory (see docs/permission-system.md).
INSERT INTO action_permissions (id, code, description) VALUES
    ('b0000001-0000-0000-0000-000000000001', 'create', 'Create entities and other resources in granted domains'),
    ('b0000002-0000-0000-0000-000000000002', 'archive', 'Archive or soft-retire governed resources'),
    ('b0000003-0000-0000-0000-000000000003', 'export', 'Export or bulk copy governed content'),
    ('b0000004-0000-0000-0000-000000000004', 'manage_jobs', 'Create, update, and operate knowledge jobs'),
    ('b0000005-0000-0000-0000-000000000005', 'manage_permissions', 'Change grants, bindings, or sensitive access controls'),
    ('b0000006-0000-0000-0000-000000000006', 'manage_policies', 'Create or change access policies'),
    ('b0000007-0000-0000-0000-000000000007', 'manage_sources', 'Manage connectors and source feeds (alias layer; maps to feed operations)')
ON CONFLICT (code) DO NOTHING;

-- Admin: all new actions
INSERT INTO role_action_permissions (role_id, action_permission_id)
SELECT '10000000-0000-0000-0000-000000000001', id FROM action_permissions
WHERE code IN ('create', 'archive', 'export', 'manage_jobs', 'manage_permissions', 'manage_policies', 'manage_sources')
ON CONFLICT (role_id, action_permission_id) DO NOTHING;

-- Analyst: job and content ops without policy/permission admin
INSERT INTO role_action_permissions (role_id, action_permission_id)
SELECT '10000000-0000-0000-0000-000000000002', id FROM action_permissions
WHERE code IN ('create', 'archive', 'export', 'manage_jobs', 'manage_sources')
ON CONFLICT (role_id, action_permission_id) DO NOTHING;

-- Viewer: read-only (no new codes)
