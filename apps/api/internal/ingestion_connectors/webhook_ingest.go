package ingestion_connectors

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"
)

// IngestWebhookResult persists artifacts emitted by a WebhookHandler under a
// single ingestion_runs row (trigger_type captured as 'manual' to reuse the
// existing pipeline; the metadata_json adds {transport:"webhook"} so dashboards
// can split webhook vs polled deliveries).
//
// Returns (run_id, ingested_count, deduped_count, error). The route uses
// ingested_count to decide whether to also call ProcessQueuedRawArtifact
// inline (matches existing polled-sync behaviour: persist raw, then normalise
// in the same request unless deferred to the connectorworker).
//
// Behaviour:
//   - Empty/nil result records an empty run; useful for url_verification or
//     unknown event types where the signature was valid but nothing maps to
//     an artifact. Operators see "we accepted a delivery, persisted nothing".
//   - Each RawArtifactInput is hashed (SHA-256 of payload), inserted via the
//     same insertRawArtifactRow helper used by polling syncs. Dedup on
//     (source_feed_id, content_hash) is enforced — a Slack retry of the same
//     event is silently absorbed without producing a duplicate row.
func (s *Service) IngestWebhookResult(ctx context.Context, feed SourceFeed, conn Connector, result *WebhookResult) (uuid.UUID, int, int, error) {
	runID, err := s.startIngestionRun(ctx, feed.ID)
	if err != nil {
		return uuid.Nil, 0, 0, fmt.Errorf("ingest webhook: start run: %w", err)
	}
	if result == nil || len(result.RawArtifacts) == 0 {
		s.finalizeIngestionRun(ctx, runID, "completed", 0, 0, 0, 0)
		s.completeSourceFeedSync(ctx, feed.ID, 0, false)
		return runID, 0, 0, nil
	}

	ingested := 0
	deduped := 0
	errCount := 0
	insertedIDs := make([]uuid.UUID, 0, len(result.RawArtifacts))
	for _, a := range result.RawArtifacts {
		rawID, inserted, err := s.persistOneWebhookArtifact(ctx, feed, conn, runID, a)
		if err != nil {
			errCount++
			continue
		}
		if inserted {
			ingested++
			insertedIDs = append(insertedIDs, rawID)
		} else {
			deduped++
		}
	}

	status := syncRunStatusFromCounts(ingested, errCount)
	s.finalizeIngestionRun(ctx, runID, status, ingested, deduped, 0, errCount)
	s.completeSourceFeedSync(ctx, feed.ID, errCount, false)

	// Inline-normalise newly inserted artifacts so webhook deliveries flow
	// into chunks/embeddings without waiting for the connectorworker. Errors
	// here are logged-and-continue inside ProcessQueuedRawArtifact when a
	// normalizer is missing; we deliberately do not flip the run status, the
	// raw artifact is durable.
	for _, id := range insertedIDs {
		_ = s.ProcessQueuedRawArtifact(ctx, id)
	}

	return runID, ingested, deduped, nil
}

func (s *Service) persistOneWebhookArtifact(ctx context.Context, feed SourceFeed, conn Connector, runID uuid.UUID, a RawArtifactInput) (uuid.UUID, bool, error) {
	if a.ArtifactType == "" {
		return uuid.Nil, false, fmt.Errorf("webhook artifact: missing artifact_type")
	}
	hash := sha256.Sum256(a.Payload)
	contentHash := hex.EncodeToString(hash[:])

	metaJSON, err := s.buildRawArtifactMetadataJSON(ctx, conn, feed, a.ArtifactType, a.Payload, mergeTransport(a))
	if err != nil {
		return uuid.Nil, false, fmt.Errorf("webhook artifact: metadata: %w", err)
	}

	var authorRef *string
	if a.SourceAuthorRef != "" {
		ar := a.SourceAuthorRef
		authorRef = &ar
	}

	return insertRawArtifactRow(ctx, s.pool, feed.ID, runID, a.ArtifactType, a.ExternalRef, contentHash, "", metaJSON, a.SourceCreatedAt, authorRef)
}

// mergeTransport carries the adapter's ExtraMetadata plus a constant marker so
// downstream tooling can distinguish webhook-delivered artifacts from polled
// ones (useful for source-feed health dashboards).
func mergeTransport(a RawArtifactInput) map[string]any {
	out := map[string]any{
		"transport": "webhook",
	}
	for k, v := range a.ExtraMetadata {
		out[k] = v
	}
	return out
}
