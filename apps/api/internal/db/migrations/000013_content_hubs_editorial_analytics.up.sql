CREATE TABLE content_hubs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    domain_id UUID NOT NULL REFERENCES domains(id),
    slug TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL DEFAULT 'draft',
    created_by_id UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (domain_id, slug)
);

CREATE INDEX idx_content_hubs_domain ON content_hubs (domain_id);

CREATE TABLE content_hub_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    hub_id UUID NOT NULL REFERENCES content_hubs(id) ON DELETE CASCADE,
    entity_id UUID NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    role TEXT NOT NULL DEFAULT 'curated',
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (hub_id, entity_id)
);

CREATE TABLE search_interaction_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    principal_id UUID NOT NULL REFERENCES users(id),
    q TEXT NOT NULL DEFAULT '',
    filters_json JSONB NOT NULL DEFAULT '{}',
    hit_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_search_interaction_log_created ON search_interaction_log (created_at DESC);

CREATE TABLE content_blocks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    domain_id UUID NOT NULL REFERENCES domains(id),
    owner_id UUID REFERENCES users(id),
    title TEXT NOT NULL,
    body TEXT NOT NULL,
    truth_mode TEXT NOT NULL DEFAULT 'derived',
    lifecycle_state TEXT NOT NULL DEFAULT 'draft',
    approval_status TEXT NOT NULL DEFAULT 'none',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE entity_content_block_refs (
    entity_id UUID NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    block_id UUID NOT NULL REFERENCES content_blocks(id) ON DELETE CASCADE,
    placement TEXT NOT NULL DEFAULT 'inline',
    sort_order INT NOT NULL DEFAULT 0,
    PRIMARY KEY (entity_id, block_id)
);

CREATE TABLE editorial_holdings (
    entity_id UUID PRIMARY KEY REFERENCES entities(id) ON DELETE CASCADE,
    held_by_id UUID REFERENCES users(id),
    reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
