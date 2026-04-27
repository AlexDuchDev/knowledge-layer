DELETE FROM role_action_permissions
WHERE role_id = '10000000-0000-0000-0000-000000000002'
  AND action_permission_id IN (SELECT id FROM action_permissions WHERE code IN ('review', 'approve'));
