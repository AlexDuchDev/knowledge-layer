-- Legacy publication_mode value "draft" on knowledge_jobs meant outputs went through review (pending_review).
-- Canonical modes: draft_only | reviewed_publish | auto_publish
UPDATE knowledge_jobs
SET publication_mode = 'reviewed_publish'
WHERE publication_mode = 'draft';

ALTER TABLE knowledge_jobs
    ALTER COLUMN publication_mode SET DEFAULT 'draft_only';
