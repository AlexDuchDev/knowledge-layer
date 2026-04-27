DELETE FROM connectors WHERE type IN (
    'notion', 'jira', 'trello', 'fireflies', 'google_calendar',
    'microsoft_365', 'gmail', 'intercom'
);
