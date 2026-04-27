-- Job Builder: definition lineage, processing/output policy fields, builder presets catalog.

ALTER TABLE knowledge_jobs
    ADD COLUMN IF NOT EXISTS template_key TEXT,
    ADD COLUMN IF NOT EXISTS processing_mode TEXT NOT NULL DEFAULT 'summarize',
    ADD COLUMN IF NOT EXISTS citations_required BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS provenance_required BOOLEAN NOT NULL DEFAULT true,
    ADD COLUMN IF NOT EXISTS scenario_only_exposure BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS cloned_from_job_id UUID REFERENCES knowledge_jobs(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS allow_domain_run_job BOOLEAN NOT NULL DEFAULT true;

ALTER TABLE knowledge_jobs
    ADD CONSTRAINT knowledge_jobs_processing_mode_chk CHECK (
        processing_mode IN ('summarize', 'extract', 'consolidate', 'detect', 'transform', 'publish')
    );

CREATE TABLE job_builder_presets (
    preset_key TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    template_key TEXT NOT NULL,
    defaults_json JSONB NOT NULL DEFAULT '{}',
    is_system BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_job_builder_presets_template ON job_builder_presets (template_key);

INSERT INTO job_builder_presets (preset_key, name, description, template_key, defaults_json, is_system) VALUES
    ('weekly_digest', 'Weekly Digest', 'Roll up feed activity into a governed digest with review.', 'weekly_digest',
     '{"processing_mode":"summarize","config_json":{"window_days":7,"channels":["summary","highlights"]},"source_scope_hint":{"requires":["domain_id","source_feed_id"]}}', true),
    ('planning_summary', 'Planning Summary', 'Synthesize planning horizon, commitments, and risks from in-scope knowledge.', 'planning_summary',
     '{"processing_mode":"summarize","config_json":{"focus":"planning","horizon_weeks":4}}', true),
    ('decision_extraction', 'Decision Extraction', 'Extract explicit decisions, owners, and rationale for review.', 'decision_extraction',
     '{"processing_mode":"extract","config_json":{"mode":"decision_mentions","min_confidence":0.6}}', true),
    ('blocker_detection', 'Blocker Detection', 'Detect blockers and dependencies from operational signals.', 'blocker_detection',
     '{"processing_mode":"detect","config_json":{"signal_types":["status","risk","dependency"],"lookback_days":14}}', true),
    ('incident_summary', 'Incident Summary', 'Post-incident structured summary and follow-ups.', 'incident_summary',
     '{"processing_mode":"summarize","config_json":{"template":"incident","include_timeline":true}}', true),
    ('executive_consolidation', 'Executive Consolidation', 'Multi-source executive brief with citations (governed).', 'executive_consolidation',
     '{"processing_mode":"consolidate","config_json":{"format":"executive_brief","max_sections":7},"citations_required":true}', true),
    ('support_trends_extraction', 'Support Trends Extraction', 'Trends and themes from support interactions in scope.', 'support_trends_extraction',
     '{"processing_mode":"extract","config_json":{"domain":"support","aggregation":"weekly"}}', true),
    ('retro_summary', 'Retro Summary', 'Retrospective themes, actions, and ownership from in-scope materials.', 'retro_summary',
     '{"processing_mode":"summarize","config_json":{"retro_format":"standard","include_action_items":true}}', true)
ON CONFLICT (preset_key) DO NOTHING;
