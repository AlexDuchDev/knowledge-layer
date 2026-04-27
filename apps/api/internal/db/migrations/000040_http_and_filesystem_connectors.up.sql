INSERT INTO connectors (id, type, display_name, auth_mode, status) VALUES
    ('20000000-0000-0000-0000-000000000011', 'http_url', 'HTTP URL', 'none', 'draft'),
    ('20000000-0000-0000-0000-000000000012', 'filesystem', 'Filesystem', 'none', 'draft')
ON CONFLICT (type) DO NOTHING;

