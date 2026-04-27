CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    primary_team_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE teams (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    description TEXT,
    owner_id UUID REFERENCES users(id),
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE users ADD CONSTRAINT users_primary_team_fk FOREIGN KEY (primary_team_id) REFERENCES teams(id);

CREATE TABLE user_team_memberships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    membership_type TEXT NOT NULL DEFAULT 'member',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, team_id)
);

CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE action_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code TEXT NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE role_action_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    action_permission_id UUID NOT NULL REFERENCES action_permissions(id) ON DELETE CASCADE,
    UNIQUE (role_id, action_permission_id)
);

CREATE TABLE user_role_bindings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    scope_type TEXT NOT NULL DEFAULT 'global',
    scope_id UUID,
    granted_by UUID REFERENCES users(id),
    granted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ
);

CREATE TABLE access_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    description TEXT,
    domain_id UUID,
    entity_type_scope TEXT,
    inheritance_mode TEXT NOT NULL DEFAULT 'inherit',
    rules_json JSONB NOT NULL DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE domains (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    description TEXT,
    owner_id UUID REFERENCES users(id),
    default_access_policy_id UUID REFERENCES access_policies(id),
    default_sensitivity_level INT NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE access_policies
    ADD CONSTRAINT access_policies_domain_fk FOREIGN KEY (domain_id) REFERENCES domains(id);

CREATE TABLE domain_grants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    domain_id UUID NOT NULL REFERENCES domains(id) ON DELETE CASCADE,
    access_level TEXT NOT NULL DEFAULT 'read',
    sensitivity_cap INT NOT NULL DEFAULT 3,
    granted_by UUID REFERENCES users(id),
    granted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ,
    UNIQUE (user_id, domain_id)
);

CREATE TABLE policy_overrides (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    target_type TEXT NOT NULL,
    target_id UUID NOT NULL,
    override_type TEXT NOT NULL,
    policy_payload JSONB NOT NULL DEFAULT '{}',
    reason TEXT,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ
);

INSERT INTO action_permissions (id, code, description) VALUES
    ('00000000-0000-0000-0000-000000000001', 'view', 'View resource'),
    ('00000000-0000-0000-0000-000000000002', 'search', 'Search within scope'),
    ('00000000-0000-0000-0000-000000000003', 'edit', 'Edit resource'),
    ('00000000-0000-0000-0000-000000000004', 'review', 'Review artifacts'),
    ('00000000-0000-0000-0000-000000000005', 'approve', 'Approve publication'),
    ('00000000-0000-0000-0000-000000000006', 'publish', 'Publish canonical changes'),
    ('00000000-0000-0000-0000-000000000007', 'run_job', 'Execute knowledge jobs'),
    ('00000000-0000-0000-0000-000000000008', 'manage_source_feed', 'Manage source feeds');

INSERT INTO roles (id, name, description) VALUES
    ('10000000-0000-0000-0000-000000000001', 'admin', 'Full operator'),
    ('10000000-0000-0000-0000-000000000002', 'analyst', 'Read + search + jobs'),
    ('10000000-0000-0000-0000-000000000003', 'viewer', 'Read only');

INSERT INTO role_action_permissions (role_id, action_permission_id)
SELECT '10000000-0000-0000-0000-000000000001', id FROM action_permissions;

INSERT INTO role_action_permissions (role_id, action_permission_id)
SELECT '10000000-0000-0000-0000-000000000002', id FROM action_permissions WHERE code IN ('view','search','run_job');

INSERT INTO role_action_permissions (role_id, action_permission_id)
SELECT '10000000-0000-0000-0000-000000000003', id FROM action_permissions WHERE code IN ('view','search');
