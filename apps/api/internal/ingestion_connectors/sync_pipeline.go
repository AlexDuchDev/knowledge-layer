package ingestion_connectors

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// startIngestionRun inserts an ingestion_runs row and marks the feed as syncing.
func (s *Service) startIngestionRun(ctx context.Context, feedID uuid.UUID) (uuid.UUID, error) {
	runID := uuid.New()
	_, err := s.pool.Exec(ctx, `
		INSERT INTO ingestion_runs (id, source_feed_id, trigger_type, status, started_at)
		VALUES ($1,$2,'manual','running',now())`, runID, feedID)
	if err != nil {
		return uuid.Nil, err
	}
	_, _ = s.pool.Exec(ctx, `UPDATE source_feeds SET sync_status='syncing', updated_at=now() WHERE id=$1`, feedID)
	return runID, nil
}

// finalizeIngestionRun sets run terminal status and counts.
func (s *Service) finalizeIngestionRun(ctx context.Context, runID uuid.UUID, status string, ingested, deduped, warnings, errs int) {
	_, _ = s.pool.Exec(ctx, `
		UPDATE ingestion_runs SET status=$2, completed_at=now(),
			records_ingested_count=$3, records_deduplicated_count=$4,
			warning_count=$5, error_count=$6
		WHERE id=$1`,
		runID, status, ingested, deduped, warnings, errs)
}

// completeSourceFeedSync updates feed health and sync_status after a connector sync.
func (s *Service) completeSourceFeedSync(ctx context.Context, feedID uuid.UUID, errs int, withTelegramSyncStatus bool) {
	syncSt := "idle"
	if withTelegramSyncStatus && errs > 0 {
		syncSt = "error"
	}
	health := "ok"
	if errs > 0 {
		health = "degraded"
	}
	if withTelegramSyncStatus {
		_, _ = s.pool.Exec(ctx, `
			UPDATE source_feeds SET last_sync_at=now(), health_status=$2, sync_status=$3, updated_at=now() WHERE id=$1`,
			feedID, health, syncSt)
		return
	}
	_, _ = s.pool.Exec(ctx, `UPDATE source_feeds SET last_sync_at=now(), health_status=$2, updated_at=now() WHERE id=$1`,
		feedID, health)
}

// syncRunStatusFromCounts maps per-item errors to ingestion_runs.status.
func syncRunStatusFromCounts(ingested, errs int) string {
	switch {
	case errs > 0 && ingested == 0:
		return "failed"
	case errs > 0:
		return "completed_with_errors"
	default:
		return "completed"
	}
}

// buildRawArtifactMetadataJSON applies adapter MapArtifactMetadata (if registered), merges extra fields, then feed governance.
func (s *Service) buildRawArtifactMetadataJSON(ctx context.Context, conn Connector, feed SourceFeed, artifactType string, rawPayload []byte, extra map[string]any) (json.RawMessage, error) {
	base := make(map[string]any)
	if s.Registry != nil {
		if a, err := s.Registry.AdapterForConnectorType(conn.Type); err == nil {
			m, err := a.MapArtifactMetadata(ctx, artifactType, rawPayload)
			if err != nil {
				return nil, fmt.Errorf("adapter MapArtifactMetadata: %w", err)
			}
			for k, v := range m {
				base[k] = v
			}
		}
	}
	for k, v := range extra {
		base[k] = v
	}
	merged := MergeFeedPolicyMetadata(feed, base)
	return json.Marshal(merged)
}

// appendFeedGovernanceToRawJSON unmarshals connector-built JSON, merges feed governance (for connectors without a registry adapter).
func appendFeedGovernanceToRawJSON(feed SourceFeed, rawJSON json.RawMessage) (json.RawMessage, error) {
	var base map[string]any
	if err := json.Unmarshal(rawJSON, &base); err != nil {
		return nil, err
	}
	return json.Marshal(MergeFeedPolicyMetadata(feed, base))
}

// insertRawArtifactRow inserts one raw_artifact with dedup on (source_feed_id, content_hash). Returns (rawID, inserted bool, err).
func insertRawArtifactRow(ctx context.Context, pool *pgxpool.Pool, feedID, runID uuid.UUID, artifactType, externalID, contentHash, storageURI string, metaJSON json.RawMessage, sourceCreatedAt *time.Time, sourceAuthorRef *string) (uuid.UUID, bool, error) {
	var rawID uuid.UUID
	err := pool.QueryRow(ctx, `
		INSERT INTO raw_artifacts (source_feed_id, ingestion_run_id, artifact_type, external_artifact_id, storage_uri, content_hash, metadata_json, source_created_at, source_author_ref)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT (source_feed_id, content_hash) DO NOTHING
		RETURNING id`,
		feedID, runID, artifactType, nullString(externalID), storageURI, contentHash, metaJSON, sourceCreatedAt, sourceAuthorRef).Scan(&rawID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, false, nil
	}
	if err != nil {
		return uuid.Nil, false, err
	}
	return rawID, true, nil
}

func nullString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
