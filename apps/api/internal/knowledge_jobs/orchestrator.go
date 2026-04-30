package knowledge_jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	platformmetrics "github.com/knowledgelayer/api/internal/platform/metrics"
)

// runOrchestrator executes a single knowledge job run: validates context, resolves sources,
// dispatches to a type-specific processor (e.g. weekly_digest), and merges run metrics.
// It is internal to the knowledge_jobs package; the HTTP surface enqueues work via Asynq.
type runOrchestrator struct {
	pool       *pgxpool.Pool
	digest     *DigestRunner
	summarizer *EntitySummarizer
}

func newRunOrchestrator(pool *pgxpool.Pool, digest *DigestRunner, summarizer *EntitySummarizer) *runOrchestrator {
	return &runOrchestrator{pool: pool, digest: digest, summarizer: summarizer}
}

func (o *runOrchestrator) execute(ctx context.Context, runID uuid.UUID, job *KnowledgeJob, operator uuid.UUID) error {
	if job == nil {
		return errors.New("orchestrator: nil job")
	}
	if err := o.validateRunContext(ctx, job); err != nil {
		return err
	}
	if err := o.resolveSources(ctx, job); err != nil {
		return err
	}
	feedIDs, warnings := o.buildInputRefs(job)
	metricsPatch := map[string]any{
		"processor":            job.JobType,
		"source_feed_ids_read": feedIDs,
		"warnings":             warnings,
	}
	start := time.Now()
	procErr := o.executeProcessor(ctx, runID, job, operator)
	status := "completed"
	if procErr != nil {
		status = "failed"
	}
	platformmetrics.ObserveJobRun(job.JobType, status, time.Since(start))
	if procErr != nil {
		return procErr
	}
	return o.mergeRunMetrics(ctx, runID, metricsPatch)
}

func (o *runOrchestrator) validateRunContext(ctx context.Context, job *KnowledgeJob) error {
	_ = ctx
	if job.ID == uuid.Nil {
		return errors.New("orchestrator: job id required")
	}
	return nil
}

func (o *runOrchestrator) resolveSources(ctx context.Context, job *KnowledgeJob) error {
	switch job.JobType {
	case "weekly_digest", "decision_extraction", "planning_summary", "support_trends_extraction":
		// these processors may read from a declared source_feed_id in scope
	default:
		return nil
	}
	var scope struct {
		SourceFeedID uuid.UUID `json:"source_feed_id"`
	}
	if err := json.Unmarshal(job.SourceScopeJSON, &scope); err != nil {
		return fmt.Errorf("orchestrator: scope: %w", err)
	}
	if scope.SourceFeedID == uuid.Nil {
		return nil
	}
	ok, err := JobAllowsSourceFeed(ctx, o.pool, job.ID, scope.SourceFeedID)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("job cannot read undeclared source feed")
	}
	return nil
}

func (o *runOrchestrator) buildInputRefs(job *KnowledgeJob) (feedIDs []string, warnings []string) {
	switch job.JobType {
	case "weekly_digest", "decision_extraction", "planning_summary", "support_trends_extraction":
		// these processors may read from a declared source_feed_id in scope
	default:
		return nil, nil
	}
	var scope struct {
		SourceFeedID uuid.UUID `json:"source_feed_id"`
	}
	if err := json.Unmarshal(job.SourceScopeJSON, &scope); err != nil {
		warnings = append(warnings, "scope_parse_error")
		return nil, warnings
	}
	if scope.SourceFeedID != uuid.Nil {
		feedIDs = append(feedIDs, scope.SourceFeedID.String())
	}
	return feedIDs, warnings
}

func (o *runOrchestrator) executeProcessor(ctx context.Context, runID uuid.UUID, job *KnowledgeJob, operator uuid.UUID) error {
	if !IsKnowledgeJobProcessorImplemented(job.JobType) {
		return fmt.Errorf("orchestrator: unsupported job_type %q", job.JobType)
	}
	switch job.JobType {
	case "weekly_digest":
		if o.digest == nil {
			return errors.New("orchestrator: weekly_digest processor not configured")
		}
		return o.digest.RunWeeklyDigest(ctx, runID, job, operator)
	case "decision_extraction":
		if o.digest == nil {
			return errors.New("orchestrator: decision_extraction processor not configured")
		}
		return o.digest.RunDecisionExtraction(ctx, runID, job, operator)
	case "planning_summary":
		if o.digest == nil {
			return errors.New("orchestrator: planning_summary processor not configured")
		}
		return o.digest.RunPlanningSummary(ctx, runID, job, operator)
	case "stale_scan":
		if o.digest == nil {
			return errors.New("orchestrator: stale_scan processor not configured")
		}
		return o.digest.RunStaleScan(ctx, runID, job, operator)
	case "support_trends_extraction":
		if o.digest == nil {
			return errors.New("orchestrator: support_trends_extraction processor not configured")
		}
		return o.digest.RunSupportTrendsExtraction(ctx, runID, job, operator)
	case "entity_summarize":
		if o.summarizer == nil {
			return errors.New("orchestrator: entity_summarize processor not configured (LLM provider missing?)")
		}
		return o.summarizer.RunEntitySummarize(ctx, runID, job, operator)
	default:
		return fmt.Errorf("orchestrator: unsupported job_type %q", job.JobType)
	}
}

func (o *runOrchestrator) mergeRunMetrics(ctx context.Context, runID uuid.UUID, patch map[string]any) error {
	var raw []byte
	err := o.pool.QueryRow(ctx, `SELECT execution_metrics_json FROM job_runs WHERE id=$1`, runID).Scan(&raw)
	if err != nil {
		return err
	}
	var base map[string]any
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &base)
	}
	if base == nil {
		base = map[string]any{}
	}
	for k, v := range patch {
		base[k] = v
	}
	out, err := json.Marshal(base)
	if err != nil {
		return err
	}
	_, err = o.pool.Exec(ctx, `UPDATE job_runs SET execution_metrics_json=$2::jsonb WHERE id=$1`, runID, out)
	return err
}
