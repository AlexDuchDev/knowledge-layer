CREATE TABLE IF NOT EXISTS tool_calls (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  principal_id UUID NOT NULL REFERENCES users(id),
  tool_name TEXT NOT NULL,
  args_redacted_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  target_type TEXT NULL,
  target_id UUID NULL,
  trace_ref TEXT NULL,
  started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  finished_at TIMESTAMPTZ NULL,
  ok BOOLEAN NULL,
  error_code TEXT NULL,
  error_message TEXT NULL,
  metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS idx_tool_calls_principal_started_at ON tool_calls (principal_id, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_tool_calls_tool_started_at ON tool_calls (tool_name, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_tool_calls_trace_ref ON tool_calls (trace_ref);

