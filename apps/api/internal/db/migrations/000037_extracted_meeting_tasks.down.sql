DROP TABLE IF EXISTS extracted_task_review_events;
DROP TABLE IF EXISTS extracted_meeting_tasks;
DELETE FROM connectors WHERE type = 'mattermost';
