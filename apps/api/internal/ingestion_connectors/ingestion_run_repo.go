package ingestion_connectors

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// FailedIngestionRun is a compact row for ops dashboards.
type FailedIngestionRun struct {
	ID           uuid.UUID `json:"id"`
	SourceFeedID uuid.UUID `json:"source_feed_id"`
	Status       string    `json:"status"`
	StartedAt    time.Time `json:"started_at"`
}

// ListRecentFailedIngestionRuns returns latest failed runs across all feeds (ops only).
func (s *Service) ListRecentFailedIngestionRuns(ctx context.Context, limit int) ([]FailedIngestionRun, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, source_feed_id, status, started_at FROM ingestion_runs
		WHERE status = 'failed' ORDER BY started_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []FailedIngestionRun
	for rows.Next() {
		var r FailedIngestionRun
		if err := rows.Scan(&r.ID, &r.SourceFeedID, &r.Status, &r.StartedAt); err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

type IngestionRun struct {
	ID                  uuid.UUID `json:"id"`
	SourceFeedID        uuid.UUID `json:"source_feed_id"`
	Status              string    `json:"status"`
	RecordsIngested     int       `json:"records_ingested_count"`
	RecordsDeduplicated int       `json:"records_deduplicated_count"`
	WarningCount        int       `json:"warning_count"`
	ErrorCount          int       `json:"error_count"`
}

func (s *Service) GetIngestionRun(ctx context.Context, id uuid.UUID) (*IngestionRun, error) {
	var r IngestionRun
	err := s.pool.QueryRow(ctx, `
		SELECT id, source_feed_id, status, records_ingested_count, records_deduplicated_count, warning_count, error_count
		FROM ingestion_runs WHERE id=$1`, id,
	).Scan(&r.ID, &r.SourceFeedID, &r.Status, &r.RecordsIngested, &r.RecordsDeduplicated, &r.WarningCount, &r.ErrorCount)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *Service) ListIngestionRuns(ctx context.Context, feedID uuid.UUID) ([]IngestionRun, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, source_feed_id, status, records_ingested_count, records_deduplicated_count, warning_count, error_count
		FROM ingestion_runs WHERE source_feed_id=$1 ORDER BY started_at DESC LIMIT 100`, feedID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []IngestionRun
	for rows.Next() {
		var r IngestionRun
		if err := rows.Scan(&r.ID, &r.SourceFeedID, &r.Status, &r.RecordsIngested, &r.RecordsDeduplicated, &r.WarningCount, &r.ErrorCount); err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, rows.Err()
}
