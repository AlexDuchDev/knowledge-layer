INSERT INTO connectors (id, type, display_name, auth_mode, status) VALUES
    ('20000000-0000-0000-0000-000000000002', 'google_drive', 'Google Drive / Docs', 'service_account', 'active')
ON CONFLICT (type) DO NOTHING;
