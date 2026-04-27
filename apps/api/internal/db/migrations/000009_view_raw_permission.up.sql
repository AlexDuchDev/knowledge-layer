-- Separate permission for reading raw ingestion artifacts (stricter than entity view).
INSERT INTO action_permissions (id, code, description) VALUES
    ('00000000-0000-0000-0000-000000000009', 'view_raw', 'View raw ingestion artifacts');

INSERT INTO role_action_permissions (role_id, action_permission_id)
SELECT '10000000-0000-0000-0000-000000000001', id FROM action_permissions WHERE code = 'view_raw';

INSERT INTO role_action_permissions (role_id, action_permission_id)
SELECT '10000000-0000-0000-0000-000000000002', id FROM action_permissions WHERE code = 'view_raw';
