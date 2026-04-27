DROP TABLE IF EXISTS preset_instantiation_logs;
DROP TABLE IF EXISTS preset_relationships;
DROP TABLE IF EXISTS preset_catalog_category_assignments;
DROP TABLE IF EXISTS preset_catalog_entries;
DROP TABLE IF EXISTS preset_categories;

ALTER TABLE roles DROP COLUMN IF EXISTS source_preset_code;
ALTER TABLE knowledge_jobs DROP COLUMN IF EXISTS source_preset_code;
ALTER TABLE scenario_definitions DROP COLUMN IF EXISTS source_preset_code;
