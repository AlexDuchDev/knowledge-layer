CREATE TABLE entities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type TEXT NOT NULL,
    title TEXT NOT NULL,
    summary TEXT,
    body TEXT,
    owner_id UUID REFERENCES users(id),
    domain_id UUID NOT NULL REFERENCES domains(id),
    sensitivity_level INT NOT NULL DEFAULT 0,
    truth_mode TEXT NOT NULL DEFAULT 'derived',
    canonical_status TEXT NOT NULL DEFAULT 'draft',
    approval_status TEXT NOT NULL DEFAULT 'none',
    freshness_status TEXT NOT NULL DEFAULT 'unknown',
    lifecycle_state TEXT NOT NULL DEFAULT 'draft',
    access_policy_id UUID REFERENCES access_policies(id),
    policy_source_type TEXT,
    policy_source_id UUID,
    external_ref TEXT,
    review_due_at TIMESTAMPTZ,
    approved_at TIMESTAMPTZ,
    approved_by_id UUID REFERENCES users(id),
    superseded_by_id UUID REFERENCES entities(id),
    created_by_type TEXT NOT NULL DEFAULT 'user',
    created_by_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    archived_at TIMESTAMPTZ
);

CREATE TABLE entity_payloads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_id UUID NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    entity_type TEXT NOT NULL,
    payload_json JSONB NOT NULL DEFAULT '{}',
    schema_version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (entity_id)
);

CREATE TABLE entity_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_id UUID NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    entity_type TEXT NOT NULL,
    version_number INT NOT NULL,
    snapshot_json JSONB NOT NULL,
    change_summary TEXT,
    changed_by_type TEXT,
    changed_by_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (entity_id, version_number)
);

CREATE TABLE entity_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    from_entity_id UUID NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    from_entity_type TEXT NOT NULL,
    relation_type TEXT NOT NULL,
    to_entity_id UUID NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    to_entity_type TEXT NOT NULL,
    created_by_type TEXT,
    created_by_id UUID,
    confidence_score DOUBLE PRECISION,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE provenance_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    target_type TEXT NOT NULL,
    target_id UUID NOT NULL,
    origin_type TEXT NOT NULL,
    origin_ref TEXT,
    source_feed_id UUID,
    job_run_id UUID,
    created_by_type TEXT,
    created_by_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE provenance_raw_artifacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provenance_record_id UUID NOT NULL REFERENCES provenance_records(id) ON DELETE CASCADE,
    raw_artifact_id UUID NOT NULL,
    UNIQUE (provenance_record_id, raw_artifact_id)
);

CREATE TABLE provenance_normalized_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provenance_record_id UUID NOT NULL REFERENCES provenance_records(id) ON DELETE CASCADE,
    normalized_record_id UUID NOT NULL,
    UNIQUE (provenance_record_id, normalized_record_id)
);

CREATE TABLE entity_search_projection (
    entity_id UUID PRIMARY KEY REFERENCES entities(id) ON DELETE CASCADE,
    domain_id UUID NOT NULL REFERENCES domains(id),
    owner_id UUID,
    entity_type TEXT NOT NULL,
    truth_mode TEXT NOT NULL,
    lifecycle_state TEXT NOT NULL,
    freshness_status TEXT NOT NULL,
    title TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_entities_domain ON entities (domain_id);
CREATE INDEX idx_entities_type ON entities (type);
CREATE INDEX idx_entity_search_domain ON entity_search_projection (domain_id);
