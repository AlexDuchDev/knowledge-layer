ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash TEXT;

CREATE TABLE user_invitations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL,
    domain_id UUID NOT NULL REFERENCES domains(id) ON DELETE CASCADE,
    role_id UUID REFERENCES roles(id),
    access_level TEXT NOT NULL DEFAULT 'read',
    sensitivity_cap INT NOT NULL DEFAULT 1,
    token_hash TEXT NOT NULL UNIQUE,
    invited_by UUID REFERENCES users(id),
    expires_at TIMESTAMPTZ NOT NULL,
    accepted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX user_invitations_email_pending_idx ON user_invitations (email) WHERE accepted_at IS NULL;
