package ingestion_connectors

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// RawArtifact is a stored ingestion payload row (joined with feed domain for access checks).
type RawArtifact struct {
	ID                 uuid.UUID       `json:"id"`
	SourceFeedID       uuid.UUID       `json:"source_feed_id"`
	DomainID           uuid.UUID       `json:"domain_id"`
	FeedSensitivity    int             `json:"feed_sensitivity_level"`
	IngestionRunID     uuid.UUID       `json:"ingestion_run_id"`
	ArtifactType       string          `json:"artifact_type"`
	ExternalArtifactID *string         `json:"external_artifact_id,omitempty"`
	StorageURI         string          `json:"storage_uri"`
	ContentHash        string          `json:"content_hash"`
	SourceCreatedAt    *time.Time      `json:"source_created_at,omitempty"`
	SourceAuthorRef    *string         `json:"source_author_ref,omitempty"`
	MetadataJSON       json.RawMessage `json:"metadata_json"`
	CreatedAt          time.Time       `json:"created_at"`
}

// NormalizedRecord is a normalized view of a raw artifact.
type NormalizedRecord struct {
	ID                    uuid.UUID       `json:"id"`
	RawArtifactID         uuid.UUID       `json:"raw_artifact_id"`
	SourceFeedID          uuid.UUID       `json:"source_feed_id"`
	DomainID              uuid.UUID       `json:"domain_id"`
	FeedSensitivity       int             `json:"feed_sensitivity_level"`
	RecordType            string          `json:"record_type"`
	StructuredPayloadJSON json.RawMessage `json:"structured_payload_json"`
	RecordHash            string          `json:"record_hash"`
	SourceTimestamp       *time.Time      `json:"source_timestamp,omitempty"`
	DetectedAuthorRef     *string         `json:"detected_author_ref,omitempty"`
	NormalizationVersion  int             `json:"normalization_version"`
	CreatedAt             time.Time       `json:"created_at"`
}

func (s *Service) ListRawArtifactsForFeed(ctx context.Context, feedID uuid.UUID, limit int) ([]RawArtifact, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx, `
		SELECT r.id, r.source_feed_id, f.domain_id, f.sensitivity_level, r.ingestion_run_id, r.artifact_type, r.external_artifact_id,
		       r.storage_uri, r.content_hash, r.source_created_at, r.source_author_ref, r.metadata_json, r.created_at
		FROM raw_artifacts r
		JOIN source_feeds f ON f.id = r.source_feed_id
		WHERE r.source_feed_id = $1
		ORDER BY r.created_at DESC
		LIMIT $2`, feedID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRawArtifacts(rows)
}

func (s *Service) GetRawArtifact(ctx context.Context, id uuid.UUID) (*RawArtifact, error) {
	var a RawArtifact
	err := s.pool.QueryRow(ctx, `
		SELECT r.id, r.source_feed_id, f.domain_id, f.sensitivity_level, r.ingestion_run_id, r.artifact_type, r.external_artifact_id,
		       r.storage_uri, r.content_hash, r.source_created_at, r.source_author_ref, r.metadata_json, r.created_at
		FROM raw_artifacts r
		JOIN source_feeds f ON f.id = r.source_feed_id
		WHERE r.id = $1`, id,
	).Scan(&a.ID, &a.SourceFeedID, &a.DomainID, &a.FeedSensitivity, &a.IngestionRunID, &a.ArtifactType, &a.ExternalArtifactID,
		&a.StorageURI, &a.ContentHash, &a.SourceCreatedAt, &a.SourceAuthorRef, &a.MetadataJSON, &a.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func scanRawArtifacts(rows pgx.Rows) ([]RawArtifact, error) {
	var list []RawArtifact
	for rows.Next() {
		var a RawArtifact
		if err := rows.Scan(&a.ID, &a.SourceFeedID, &a.DomainID, &a.FeedSensitivity, &a.IngestionRunID, &a.ArtifactType, &a.ExternalArtifactID,
			&a.StorageURI, &a.ContentHash, &a.SourceCreatedAt, &a.SourceAuthorRef, &a.MetadataJSON, &a.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, rows.Err()
}

func (s *Service) ListNormalizedRecordsForFeed(ctx context.Context, feedID uuid.UUID, limit int) ([]NormalizedRecord, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx, `
		SELECT n.id, n.raw_artifact_id, n.source_feed_id, f.domain_id, f.sensitivity_level, n.record_type, n.structured_payload_json,
		       n.record_hash, n.source_timestamp, n.detected_author_ref, n.normalization_version, n.created_at
		FROM normalized_records n
		JOIN source_feeds f ON f.id = n.source_feed_id
		WHERE n.source_feed_id = $1
		ORDER BY n.created_at DESC
		LIMIT $2`, feedID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []NormalizedRecord
	for rows.Next() {
		var nrec NormalizedRecord
		if err := rows.Scan(&nrec.ID, &nrec.RawArtifactID, &nrec.SourceFeedID, &nrec.DomainID, &nrec.FeedSensitivity, &nrec.RecordType,
			&nrec.StructuredPayloadJSON, &nrec.RecordHash, &nrec.SourceTimestamp, &nrec.DetectedAuthorRef,
			&nrec.NormalizationVersion, &nrec.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, nrec)
	}
	return list, rows.Err()
}

func (s *Service) GetNormalizedRecord(ctx context.Context, id uuid.UUID) (*NormalizedRecord, error) {
	var nrec NormalizedRecord
	err := s.pool.QueryRow(ctx, `
		SELECT n.id, n.raw_artifact_id, n.source_feed_id, f.domain_id, f.sensitivity_level, n.record_type, n.structured_payload_json,
		       n.record_hash, n.source_timestamp, n.detected_author_ref, n.normalization_version, n.created_at
		FROM normalized_records n
		JOIN source_feeds f ON f.id = n.source_feed_id
		WHERE n.id = $1`, id,
	).Scan(&nrec.ID, &nrec.RawArtifactID, &nrec.SourceFeedID, &nrec.DomainID, &nrec.FeedSensitivity, &nrec.RecordType,
		&nrec.StructuredPayloadJSON, &nrec.RecordHash, &nrec.SourceTimestamp, &nrec.DetectedAuthorRef,
		&nrec.NormalizationVersion, &nrec.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &nrec, nil
}
