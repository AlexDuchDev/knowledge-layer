-- Connector rows for v1 family adapters (sync implemented in application code).
INSERT INTO connectors (id, type, display_name, auth_mode, status) VALUES
    ('20000000-0000-0000-0000-000000000004', 'notion', 'Notion', 'api_token', 'draft'),
    ('20000000-0000-0000-0000-000000000005', 'jira', 'Jira Cloud', 'oauth', 'draft'),
    ('20000000-0000-0000-0000-000000000006', 'trello', 'Trello', 'api_key', 'draft'),
    ('20000000-0000-0000-0000-000000000007', 'fireflies', 'Fireflies.ai', 'api_key', 'draft'),
    ('20000000-0000-0000-0000-000000000008', 'google_calendar', 'Google Calendar', 'oauth', 'draft'),
    ('20000000-0000-0000-0000-000000000009', 'microsoft_365', 'Microsoft 365', 'oauth', 'draft'),
    ('20000000-0000-0000-0000-00000000000a', 'gmail', 'Gmail', 'oauth', 'draft'),
    ('20000000-0000-0000-0000-00000000000b', 'intercom', 'Intercom', 'api_token', 'draft')
ON CONFLICT (type) DO NOTHING;
