INSERT INTO connectors (id, type, display_name, auth_mode, status) VALUES
    ('20000000-0000-0000-0000-00000000000c', 'confluence', 'Confluence', 'api_token', 'draft'),
    ('20000000-0000-0000-0000-00000000000d', 'hubspot', 'HubSpot', 'api_token', 'draft'),
    ('20000000-0000-0000-0000-00000000000e', 'zendesk', 'Zendesk', 'api_token', 'draft'),
    ('20000000-0000-0000-0000-00000000000f', 'asana', 'Asana', 'api_token', 'draft'),
    ('20000000-0000-0000-0000-000000000010', 'linear', 'Linear', 'api_token', 'draft')
ON CONFLICT (type) DO NOTHING;
