CREATE TABLE connectors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    auth_mode TEXT NOT NULL DEFAULT 'unspecified',
    capabilities_json JSONB NOT NULL DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE source_feeds (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    connector_id UUID NOT NULL REFERENCES connectors(id),
    source_uri TEXT NOT NULL DEFAULT '',
    display_name TEXT NOT NULL,
    owner_id UUID NOT NULL REFERENCES users(id),
    domain_id UUID NOT NULL REFERENCES domains(id),
    sensitivity_level INT NOT NULL DEFAULT 0,
    allowed_job_types_json JSONB NOT NULL DEFAULT '[]',
    ingestion_mode TEXT NOT NULL DEFAULT 'ingestion_only',
    sync_mode TEXT NOT NULL DEFAULT 'manual',
    access_policy_id UUID REFERENCES access_policies(id),
    connector_config_json JSONB NOT NULL DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'draft',
    health_status TEXT NOT NULL DEFAULT 'unknown',
    last_sync_at TIMESTAMPTZ,
    last_successful_sync_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE ingestion_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_feed_id UUID NOT NULL REFERENCES source_feeds(id) ON DELETE CASCADE,
    trigger_type TEXT NOT NULL DEFAULT 'manual',
    status TEXT NOT NULL DEFAULT 'pending',
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ,
    records_ingested_count INT NOT NULL DEFAULT 0,
    records_deduplicated_count INT NOT NULL DEFAULT 0,
    warning_count INT NOT NULL DEFAULT 0,
    error_count INT NOT NULL DEFAULT 0,
    trace_ref TEXT
);

CREATE TABLE raw_artifacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_feed_id UUID NOT NULL REFERENCES source_feeds(id) ON DELETE CASCADE,
    ingestion_run_id UUID NOT NULL REFERENCES ingestion_runs(id) ON DELETE CASCADE,
    artifact_type TEXT NOT NULL,
    external_artifact_id TEXT,
    storage_uri TEXT NOT NULL DEFAULT '',
    content_hash TEXT NOT NULL,
    source_created_at TIMESTAMPTZ,
    source_author_ref TEXT,
    metadata_json JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (source_feed_id, content_hash)
);

CREATE TABLE normalized_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    raw_artifact_id UUID NOT NULL REFERENCES raw_artifacts(id) ON DELETE CASCADE,
    source_feed_id UUID NOT NULL REFERENCES source_feeds(id) ON DELETE CASCADE,
    record_type TEXT NOT NULL,
    structured_payload_json JSONB NOT NULL DEFAULT '{}',
    record_hash TEXT NOT NULL,
    source_timestamp TIMESTAMPTZ,
    detected_author_ref TEXT,
    normalization_version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (source_feed_id, record_hash)
);

ALTER TABLE provenance_records
    ADD CONSTRAINT provenance_source_feed_fk FOREIGN KEY (source_feed_id) REFERENCES source_feeds(id);

INSERT INTO connectors (id, type, display_name, auth_mode, status) VALUES
    ('20000000-0000-0000-0000-000000000001', 'telegram', 'Telegram', 'bot_token', 'active');
