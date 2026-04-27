CREATE TABLE knowledge_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    job_type TEXT NOT NULL,
    purpose TEXT,
    description TEXT,
    owner_id UUID NOT NULL REFERENCES users(id),
    operator_scope_json JSONB NOT NULL DEFAULT '{}',
    source_scope_json JSONB NOT NULL DEFAULT '{}',
    trigger_type TEXT NOT NULL DEFAULT 'manual',
    output_type TEXT NOT NULL DEFAULT 'artifact',
    output_domain_id UUID REFERENCES domains(id),
    output_sensitivity_level INT NOT NULL DEFAULT 0,
    publication_mode TEXT NOT NULL DEFAULT 'draft',
    review_required BOOLEAN NOT NULL DEFAULT true,
    approval_required BOOLEAN NOT NULL DEFAULT false,
    sanitization_rules_json JSONB NOT NULL DEFAULT '{}',
    config_json JSONB NOT NULL DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'draft',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE job_triggers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    knowledge_job_id UUID NOT NULL REFERENCES knowledge_jobs(id) ON DELETE CASCADE,
    trigger_type TEXT NOT NULL,
    schedule_expr TEXT,
    event_filter_json JSONB NOT NULL DEFAULT '{}',
    window_config_json JSONB NOT NULL DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE job_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    knowledge_job_id UUID NOT NULL REFERENCES knowledge_jobs(id) ON DELETE CASCADE,
    initiated_by_type TEXT NOT NULL,
    initiated_by_id UUID,
    trigger_type TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    input_scope_snapshot_json JSONB NOT NULL DEFAULT '{}',
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ,
    warning_count INT NOT NULL DEFAULT 0,
    error_count INT NOT NULL DEFAULT 0,
    trace_ref TEXT,
    execution_metrics_json JSONB NOT NULL DEFAULT '{}'
);

CREATE TABLE job_outputs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_run_id UUID NOT NULL REFERENCES job_runs(id) ON DELETE CASCADE,
    output_type TEXT NOT NULL,
    structured_payload_json JSONB NOT NULL DEFAULT '{}',
    target_entity_id UUID REFERENCES entities(id),
    target_entity_type TEXT,
    review_task_id UUID,
    publication_status TEXT NOT NULL DEFAULT 'pending_review',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE provenance_records
    ADD CONSTRAINT provenance_job_run_fk FOREIGN KEY (job_run_id) REFERENCES job_runs(id);
