ALTER TABLE provenance_records DROP CONSTRAINT IF EXISTS provenance_source_feed_fk;
DROP TABLE IF EXISTS normalized_records;
DROP TABLE IF EXISTS raw_artifacts;
DROP TABLE IF EXISTS ingestion_runs;
DROP TABLE IF EXISTS source_feeds;
DROP TABLE IF EXISTS connectors;
