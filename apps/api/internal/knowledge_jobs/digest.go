package knowledge_jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/knowledgelayer/api/internal/knowledge_core"
	"github.com/knowledgelayer/api/internal/review"
)

type DigestRunner struct {
	pool     *pgxpool.Pool
	entities *knowledge_core.EntityRepo
	review   *review.Service
}

func NewDigestRunner(pool *pgxpool.Pool, entities *knowledge_core.EntityRepo, rev *review.Service) *DigestRunner {
	return &DigestRunner{pool: pool, entities: entities, review: rev}
}

// RunWeeklyDigest builds a derived digest entity from normalized records in scope and opens a review task.
// There is no LLM call here; if digest text is ever sent to a model, route through internal/ai/privacy.PrivacyGateway.
func (d *DigestRunner) RunWeeklyDigest(ctx context.Context, runID uuid.UUID, job *KnowledgeJob, operator uuid.UUID) error {
	var scope struct {
		SourceFeedID uuid.UUID `json:"source_feed_id"`
		DomainID     uuid.UUID `json:"domain_id"`
	}
	if err := json.Unmarshal(job.SourceScopeJSON, &scope); err != nil {
		return fmt.Errorf("digest: scope: %w", err)
	}
	if scope.SourceFeedID == uuid.Nil || scope.DomainID == uuid.Nil {
		return fmt.Errorf("digest: source_feed_id and domain_id required in source_scope_json")
	}

	ok, err := JobAllowsSourceFeed(ctx, d.pool, job.ID, scope.SourceFeedID)
	if err != nil {
		return fmt.Errorf("digest: source allow check: %w", err)
	}
	if !ok {
		return fmt.Errorf("digest: source feed %s is not declared for this job", scope.SourceFeedID)
	}

	since := time.Now().UTC().Add(-7 * 24 * time.Hour)
	rows, err := d.pool.Query(ctx, `
		SELECT structured_payload_json FROM normalized_records
		WHERE source_feed_id = $1 AND created_at >= $2 ORDER BY created_at ASC LIMIT 500`,
		scope.SourceFeedID, since)
	if err != nil {
		return err
	}
	defer rows.Close()

	var lines []string
	for rows.Next() {
		var raw json.RawMessage
		if err := rows.Scan(&raw); err != nil {
			return err
		}
		lines = append(lines, string(raw))
	}
	if err := rows.Err(); err != nil {
		return err
	}

	plan := PlanWeeklyDigestOutput(job)

	body := fmt.Sprintf("Weekly digest (%d normalized records)\n---\n%s", len(lines), joinLimit(lines, 20))
	ent, err := d.entities.Create(ctx, knowledge_core.CreateEntityInput{
		Type:             "Insight",
		Title:            fmt.Sprintf("Weekly digest for feed %s", scope.SourceFeedID.String()[:8]),
		Body:             &body,
		OwnerID:          &operator,
		DomainID:         scope.DomainID,
		SensitivityLevel: job.OutputSensitivity,
		TruthMode:        "derived",
		LifecycleState:   plan.EntityLifecycle,
		PayloadJSON:      json.RawMessage(`{"kind":"weekly_digest","record_count":` + fmtInt(len(lines)) + `}`),
	})
	if err != nil {
		return err
	}

	ref := "digest-" + runID.String()
	_, err = d.entities.AttachProvenance(ctx, knowledge_core.ProvenanceRecord{
		TargetType: "entity",
		TargetID:   ent.ID,
		OriginType: "knowledge_job_run",
		OriginRef:  &ref,
		JobRunID:   &runID,
	}, nil, nil)
	if err != nil {
		return err
	}

	var reviewTaskID *uuid.UUID
	if plan.CreateReviewTask {
		rt, err := d.review.Create(ctx, "entity", ent.ID, operator, &operator, nil)
		if err != nil {
			return err
		}
		reviewTaskID = &rt.ID
	}

	payload := map[string]any{"digest_entity_id": ent.ID.String()}
	if reviewTaskID != nil {
		payload["review_task_id"] = reviewTaskID.String()
	}
	payloadBytes, _ := json.Marshal(payload)
	_, err = d.pool.Exec(ctx, `
		INSERT INTO job_outputs (job_run_id, output_type, structured_payload_json, target_entity_id, target_entity_type, review_task_id, publication_status)
		VALUES ($1,'weekly_digest',$2,$3,'entity',$4,$5)`,
		runID, payloadBytes, ent.ID, reviewTaskID, plan.JobOutputPublicationStatus)
	return err
}

func joinLimit(lines []string, max int) string {
	if len(lines) > max {
		lines = lines[:max]
	}
	out := ""
	for _, l := range lines {
		out += l + "\n"
	}
	return out
}

func fmtInt(i int) string {
	return fmt.Sprintf("%d", i)
}
