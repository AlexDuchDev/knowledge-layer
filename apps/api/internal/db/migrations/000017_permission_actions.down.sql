DELETE FROM role_action_permissions
WHERE action_permission_id IN (SELECT id FROM action_permissions WHERE code IN (
    'create', 'archive', 'export', 'manage_jobs', 'manage_permissions', 'manage_policies', 'manage_sources'
));

DELETE FROM action_permissions WHERE code IN (
    'create', 'archive', 'export', 'manage_jobs', 'manage_permissions', 'manage_policies', 'manage_sources'
);
