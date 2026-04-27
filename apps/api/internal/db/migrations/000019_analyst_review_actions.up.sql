-- Analysts often operate review queues; governance routes require review/approve on the target entity.
INSERT INTO role_action_permissions (role_id, action_permission_id)
SELECT '10000000-0000-0000-0000-000000000002', id FROM action_permissions
WHERE code IN ('review', 'approve')
ON CONFLICT (role_id, action_permission_id) DO NOTHING;
