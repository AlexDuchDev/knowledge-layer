DROP TABLE IF EXISTS job_builder_presets;

ALTER TABLE knowledge_jobs DROP CONSTRAINT IF EXISTS knowledge_jobs_processing_mode_chk;

ALTER TABLE knowledge_jobs
    DROP COLUMN IF EXISTS allow_domain_run_job,
    DROP COLUMN IF EXISTS cloned_from_job_id,
    DROP COLUMN IF EXISTS scenario_only_exposure,
    DROP COLUMN IF EXISTS provenance_required,
    DROP COLUMN IF EXISTS citations_required,
    DROP COLUMN IF EXISTS processing_mode,
    DROP COLUMN IF EXISTS template_key;
