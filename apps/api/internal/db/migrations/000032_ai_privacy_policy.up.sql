-- AI privacy policy rules: scoped overrides for sensitive entity handling before LLM calls.
CREATE TABLE ai_privacy_policy_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scope_kind TEXT NOT NULL CHECK (scope_kind IN (
        'global', 'domain', 'source_feed', 'scenario', 'job_type', 'output_type'
    )),
    scope_id TEXT NULL,
    entity_type TEXT NOT NULL,
    action TEXT NOT NULL CHECK (action IN ('keep', 'tokenize', 'remove', 'disallow_ai')),
    rehydration_mode TEXT NOT NULL DEFAULT 'none' CHECK (rehydration_mode IN ('none', 'partial', 'full')),
    priority INT NOT NULL DEFAULT 0,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_ai_privacy_policy_scope ON ai_privacy_policy_rules (scope_kind, scope_id) WHERE enabled = true;
CREATE INDEX idx_ai_privacy_policy_entity ON ai_privacy_policy_rules (entity_type) WHERE enabled = true;

-- Default global rules: tokenize common sensitive types; disallow_ai for obvious secrets.
INSERT INTO ai_privacy_policy_rules (scope_kind, scope_id, entity_type, action, rehydration_mode, priority) VALUES
('global', NULL, 'security_secret', 'disallow_ai', 'none', 100),
('global', NULL, 'email', 'tokenize', 'partial', 50),
('global', NULL, 'phone', 'tokenize', 'partial', 50),
('global', NULL, 'person_name', 'tokenize', 'partial', 40),
('global', NULL, 'company_name', 'tokenize', 'partial', 40),
('global', NULL, 'customer_id', 'tokenize', 'partial', 40),
('global', NULL, 'account_id', 'tokenize', 'partial', 40),
('global', NULL, 'contract_ref', 'tokenize', 'partial', 40),
('global', NULL, 'invoice_ref', 'tokenize', 'partial', 40),
('global', NULL, 'address', 'tokenize', 'partial', 40),
('global', NULL, 'government_id', 'tokenize', 'none', 60),
('global', NULL, 'financial_account', 'tokenize', 'none', 60),
('global', NULL, 'legal_ref', 'tokenize', 'partial', 40),
('global', NULL, 'internal_codename', 'tokenize', 'partial', 30),
('global', NULL, 'custom_pattern', 'tokenize', 'partial', 20);
