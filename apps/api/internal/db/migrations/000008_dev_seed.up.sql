INSERT INTO access_policies (id, name, description, status) VALUES
    ('31000000-0000-0000-0000-000000000001', 'default', 'Bootstrap policy', 'active');

INSERT INTO users (id, email, name, status) VALUES
    ('30000000-0000-0000-0000-000000000001', 'admin@local.test', 'Admin', 'active'),
    ('30000000-0000-0000-0000-000000000002', 'viewer@local.test', 'Viewer', 'active');

INSERT INTO domains (id, name, owner_id, default_access_policy_id, status) VALUES
    ('32000000-0000-0000-0000-000000000001', 'Default',
     '30000000-0000-0000-0000-000000000001',
     '31000000-0000-0000-0000-000000000001',
     'active');

UPDATE access_policies SET domain_id = '32000000-0000-0000-0000-000000000001'
WHERE id = '31000000-0000-0000-0000-000000000001';

INSERT INTO domain_grants (user_id, domain_id, access_level, sensitivity_cap) VALUES
    ('30000000-0000-0000-0000-000000000001', '32000000-0000-0000-0000-000000000001', 'admin', 3),
    ('30000000-0000-0000-0000-000000000002', '32000000-0000-0000-0000-000000000001', 'read', 1);

INSERT INTO user_role_bindings (user_id, role_id, scope_type, scope_id) VALUES
    ('30000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000001', 'global', NULL),
    ('30000000-0000-0000-0000-000000000002', '10000000-0000-0000-0000-000000000003', 'domain', '32000000-0000-0000-0000-000000000001');
