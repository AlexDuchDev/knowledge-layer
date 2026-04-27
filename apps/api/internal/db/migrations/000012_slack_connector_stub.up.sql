-- Template connector row for Epic 10 backlog: implement sync/preview in code before production use.
INSERT INTO connectors (id, type, display_name, auth_mode, status)
VALUES ('20000000-0000-0000-0000-000000000003', 'slack', 'Slack (stub)', 'oauth', 'draft')
ON CONFLICT (type) DO NOTHING;
