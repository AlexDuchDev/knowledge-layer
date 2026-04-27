DROP TABLE IF EXISTS scenario_ui_bindings;
DROP TABLE IF EXISTS scenario_job_bindings;
DROP TABLE IF EXISTS scenario_source_bindings;
DROP TABLE IF EXISTS scenario_role_bindings;

ALTER TABLE scenario_definitions DROP CONSTRAINT IF EXISTS scenario_definitions_output_policy_id_fkey;
ALTER TABLE scenario_definitions DROP COLUMN IF EXISTS output_policy_id;

DROP TABLE IF EXISTS scenario_output_policies;
DROP TABLE IF EXISTS scenario_definitions;
DROP TABLE IF EXISTS scenario_presets;
