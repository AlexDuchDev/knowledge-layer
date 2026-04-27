CREATE TABLE IF NOT EXISTS entity_compiled_truth (
  entity_id UUID PRIMARY KEY REFERENCES entities(id) ON DELETE CASCADE,
  compiled_summary TEXT NULL,
  compiled_body TEXT NULL,
  based_on_version_number INT NULL,
  compiled_by_type TEXT NULL,
  compiled_by_id UUID NULL,
  compiled_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_entity_compiled_truth_compiled_at ON entity_compiled_truth (compiled_at DESC);

