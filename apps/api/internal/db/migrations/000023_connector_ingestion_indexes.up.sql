-- Query paths for governed ingestion: feeds by connector, artifacts and runs by feed + time.

CREATE INDEX IF NOT EXISTS idx_source_feeds_connector_id ON source_feeds (connector_id);
CREATE INDEX IF NOT EXISTS idx_raw_artifacts_source_feed_created ON raw_artifacts (source_feed_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ingestion_runs_source_feed_started ON ingestion_runs (source_feed_id, started_at DESC);
