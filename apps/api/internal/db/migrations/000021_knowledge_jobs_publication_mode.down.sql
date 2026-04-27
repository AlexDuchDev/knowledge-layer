ALTER TABLE knowledge_jobs
    ALTER COLUMN publication_mode SET DEFAULT 'draft';

UPDATE knowledge_jobs
SET publication_mode = 'draft'
WHERE publication_mode = 'reviewed_publish';
