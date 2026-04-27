package ingestion_connectors

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/knowledgelayer/api/internal/ingestion_connectors/families/docs_wiki"
)

const artifactFilesystemFile = "filesystem_file"

// SyncFilesystem ingests a single file per source feed from a read-only mounted directory (default /data).
// The file path is provided as source_feeds.external_ref, and must be relative (no absolute paths).
func (s *Service) SyncFilesystem(ctx context.Context, feedID uuid.UUID) (*IngestionRun, error) {
	feed, err := s.GetSourceFeed(ctx, feedID)
	if err != nil {
		return nil, err
	}
	conn, err := s.GetConnector(ctx, feed.ConnectorID)
	if err != nil {
		return nil, err
	}
	if conn.Type != "filesystem" {
		return nil, fmt.Errorf("connector is %s, not filesystem", conn.Type)
	}
	if feed.Status != "active" {
		return nil, errors.New("source feed must be active")
	}
	rel := strings.TrimSpace(feed.ExternalRef)
	if rel == "" {
		return nil, errors.New("source_feeds.external_ref must be a relative file path under /data")
	}
	if filepath.IsAbs(rel) || strings.Contains(rel, "..") {
		return nil, errors.New("filesystem external_ref must be a safe relative path")
	}
	full := filepath.Join("/data", rel)

	runID, err := s.startIngestionRun(ctx, feedID)
	if err != nil {
		return nil, err
	}

	ingested, deduped, warnings, errs := 0, 0, 0, 0

	b, err := os.ReadFile(full)
	if err != nil {
		errs++
		s.finalizeIngestionRun(ctx, runID, "failed", ingested, deduped, warnings, errs)
		s.completeSourceFeedSync(ctx, feedID, errs, false)
		return nil, err
	}

	extra := map[string]any{
		"filesystem_file": map[string]any{
			"path":         rel,
			"content_text": string(b),
		},
	}
	metaJSON, err := s.buildRawArtifactMetadataJSON(ctx, *conn, *feed, artifactFilesystemFile, b, extra)
	if err != nil {
		errs++
		s.finalizeIngestionRun(ctx, runID, "failed", ingested, deduped, warnings, errs)
		s.completeSourceFeedSync(ctx, feedID, errs, false)
		return nil, err
	}

	h := hashBytes(metaJSON)
	rawID, inserted, err := insertRawArtifactRow(ctx, s.pool, feedID, runID, artifactFilesystemFile, rel, h, "", metaJSON, timePtr(time.Now().UTC()), nil)
	if err != nil {
		errs++
		s.finalizeIngestionRun(ctx, runID, "failed", ingested, deduped, warnings, errs)
		s.completeSourceFeedSync(ctx, feedID, errs, false)
		return nil, err
	}
	if !inserted {
		deduped++
		status := syncRunStatusFromCounts(ingested, errs)
		s.finalizeIngestionRun(ctx, runID, status, ingested, deduped, warnings, errs)
		s.completeSourceFeedSync(ctx, feedID, errs, false)
		return s.GetIngestionRun(ctx, runID)
	}

	title := filepath.Base(rel)
	now := timePtr(time.Now().UTC())
	norm := docs_wiki.NormalizedDocPage{
		SourceFeedID:    feedID,
		ConnectorFamily: "docs_wiki",
		ConnectorType:   "filesystem",
		Title:           title,
		ExternalRef:     rel,
		LastModifiedAt:  now,
		BodyText:        string(b),
		WebViewLink:     "",
		DownstreamHint: map[string]any{
			"suggested_entity_type": "ReferenceDocument",
			"default_truth_mode":    "mirrored_authority",
		},
	}
	normPayload, err := json.Marshal(norm)
	if err != nil {
		errs++
		s.finalizeIngestionRun(ctx, runID, "failed", ingested, deduped, warnings, errs)
		s.completeSourceFeedSync(ctx, feedID, errs, false)
		return nil, err
	}
	nh := hashBytes(normPayload)
	tag, err := s.pool.Exec(ctx, `
		INSERT INTO normalized_records (raw_artifact_id, source_feed_id, record_type, structured_payload_json, record_hash, source_timestamp, detected_author_ref, normalization_version)
		VALUES ($1,$2,$3,$4,$5,$6,$7,1)
		ON CONFLICT (source_feed_id, record_hash) DO NOTHING`,
		rawID, feedID, docs_wiki.RecordTypeDocsPage, normPayload, nh, now, nil)
	if err != nil {
		errs++
	} else if tag.RowsAffected() == 0 {
		deduped++
	} else {
		ingested++
	}

	status := syncRunStatusFromCounts(ingested, errs)
	s.finalizeIngestionRun(ctx, runID, status, ingested, deduped, warnings, errs)
	s.completeSourceFeedSync(ctx, feedID, errs, false)
	return s.GetIngestionRun(ctx, runID)
}
