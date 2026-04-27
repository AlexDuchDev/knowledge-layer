ALTER TABLE provenance_records DROP CONSTRAINT IF EXISTS provenance_job_run_fk;
DROP TABLE IF EXISTS job_outputs;
DROP TABLE IF EXISTS job_runs;
DROP TABLE IF EXISTS job_triggers;
DROP TABLE IF EXISTS knowledge_jobs;
