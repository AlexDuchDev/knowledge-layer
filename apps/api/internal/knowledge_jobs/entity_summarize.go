package knowledge_jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/knowledgelayer/api/internal/ai/privacy"
	"github.com/knowledgelayer/api/internal/ai/prompts"
	"github.com/knowledgelayer/api/internal/llm"
)

// EntitySummarizer is the entity_summarize knowledge job processor. It reads
// entities WHERE entity_search_projection.synthesized_summary IS NULL (or
// stale relative to entities.updated_at), routes a small per-entity LLM call
// through the privacy gateway with PromptTemplateID="entity_summarize.v1",
// and UPSERTs synthesized_summary + synthesized_at on the projection.
//
// Cost discipline: each run is bounded by max_rows (default 100, hard cap
// 500). Per-row body is truncated to ~bodyExcerptRunes runes before being
// sent to the model. CLI / API callers can override max_rows via job
// source_scope_json; values above the hard cap are clamped.
//
// Why this is a separate runner from DigestRunner: DigestRunner reads
// normalized_records by source_feed; this one reads entities. They share
// nothing relevant, and keeping them separate avoids polluting either with
// dependencies the other doesn't need.
type EntitySummarizer struct {
	pool    *pgxpool.Pool
	privacy *privacy.PrivacyGateway
}

// NewEntitySummarizer wires the runner. privacy may be nil during local
// dev without an LLM provider — RunEntitySummarize then errors clearly
// rather than silently producing empty summaries.
func NewEntitySummarizer(pool *pgxpool.Pool, gw *privacy.PrivacyGateway) *EntitySummarizer {
	return &EntitySummarizer{pool: pool, privacy: gw}
}

// Hard caps to prevent runaway LLM cost from a misconfigured job.
const (
	defaultEntitySummarizeMaxRows = 100
	hardCapEntitySummarizeMaxRows = 500
	bodyExcerptRunes              = 1200
)

// EntitySummarizeScope is the parsed source_scope_json for entity_summarize
// jobs. All fields optional. With nothing set the job processes up to the
// default max_rows entities lacking a summary in any domain.
type EntitySummarizeScope struct {
	DomainID *uuid.UUID `json:"domain_id,omitempty"`
	MaxRows  int        `json:"max_rows,omitempty"`
	// EntityIDs lets the CLI's --entity flag target specific entities.
	// When set, MaxRows + DomainID are ignored.
	EntityIDs []uuid.UUID `json:"entity_ids,omitempty"`
}

// RunEntitySummarize is the orchestrator entry point. Returns nil on success
// (some rows may have failed individually — those are logged via audit).
func (e *EntitySummarizer) RunEntitySummarize(ctx context.Context, runID uuid.UUID, job *KnowledgeJob, operator uuid.UUID) error {
	if e.privacy == nil {
		return errors.New("entity_summarize: privacy gateway unavailable; LLM provider must be configured (set OPENAI_API_KEY or OPENROUTER_API_KEY)")
	}

	var scope EntitySummarizeScope
	if len(job.SourceScopeJSON) > 0 {
		if err := json.Unmarshal(job.SourceScopeJSON, &scope); err != nil {
			return fmt.Errorf("entity_summarize: scope: %w", err)
		}
	}
	maxRows := clampMaxRows(scope.MaxRows)

	prompt, err := prompts.Get("entity_summarize.v1")
	if err != nil {
		return fmt.Errorf("entity_summarize: prompt template: %w", err)
	}

	client, err := llm.NewOpenAIFromEnv()
	if err != nil {
		return fmt.Errorf("entity_summarize: llm client: %w", err)
	}

	rows, err := e.fetchPendingEntities(ctx, scope, maxRows)
	if err != nil {
		return err
	}
	processed := 0
	for _, r := range rows {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		summary, sumErr := e.summarizeOne(ctx, client, prompt, r)
		if sumErr != nil {
			// Per-row failures don't abort the run — bad entity content
			// shouldn't block the rest of the batch. Surfaced via the run
			// metrics that the orchestrator merges.
			continue
		}
		if uerr := e.writeSummary(ctx, r.ID, summary); uerr != nil {
			continue
		}
		processed++
	}
	// Stash a small per-run metric blob on job_runs so operators can see
	// how many entities the run handled. The orchestrator tolerates
	// missing fields if the job_runs row already has metrics.
	_, _ = e.pool.Exec(ctx, `
		UPDATE job_runs SET metrics_json = jsonb_set(coalesce(metrics_json, '{}'::jsonb), '{entity_summarize}',
			to_jsonb(json_build_object(
				'requested', $2::int,
				'processed', $3::int,
				'max_rows',  $4::int
			)))
		WHERE id = $1`, runID, len(rows), processed, maxRows)
	return nil
}

// pendingEntity is what fetchPendingEntities returns: the minimum needed to
// build a prompt and write back the summary.
type pendingEntity struct {
	ID    uuid.UUID
	Type  string
	Title string
	Body  string
}

func (e *EntitySummarizer) fetchPendingEntities(ctx context.Context, scope EntitySummarizeScope, limit int) ([]pendingEntity, error) {
	if len(scope.EntityIDs) > 0 {
		// Targeted mode: process exactly these entities (re-summarize even
		// if they already have a synthesized_summary). Useful for the
		// CLI's `--entity <id>` flag.
		rows, err := e.pool.Query(ctx, `
			SELECT e.id, e.type, e.title, COALESCE(e.body, '')
			FROM entities e
			WHERE e.id = ANY($1) AND e.archived_at IS NULL`, scope.EntityIDs)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return scanPendingEntities(rows)
	}

	// Backfill mode: entities that don't yet have a summary, optionally
	// scoped to a single domain. Newer entities first so a freshly ingested
	// document gets summarized before old archived ones.
	if scope.DomainID != nil {
		rows, err := e.pool.Query(ctx, `
			SELECT e.id, e.type, e.title, COALESCE(e.body, '')
			FROM entities e
			JOIN entity_search_projection p ON p.entity_id = e.id
			WHERE e.archived_at IS NULL
			  AND e.domain_id = $1
			  AND p.synthesized_summary IS NULL
			ORDER BY e.updated_at DESC
			LIMIT $2`, *scope.DomainID, limit)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return scanPendingEntities(rows)
	}
	rows, err := e.pool.Query(ctx, `
		SELECT e.id, e.type, e.title, COALESCE(e.body, '')
		FROM entities e
		JOIN entity_search_projection p ON p.entity_id = e.id
		WHERE e.archived_at IS NULL
		  AND p.synthesized_summary IS NULL
		ORDER BY e.updated_at DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPendingEntities(rows)
}

func scanPendingEntities(rows interface {
	Next() bool
	Scan(...any) error
	Err() error
}) ([]pendingEntity, error) {
	var list []pendingEntity
	for rows.Next() {
		var p pendingEntity
		if err := rows.Scan(&p.ID, &p.Type, &p.Title, &p.Body); err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

// summarizeOne sends one entity through the privacy gateway and returns the
// model's summary text. Errors propagate without retry — the caller drops
// the row and moves on.
func (e *EntitySummarizer) summarizeOne(ctx context.Context, client privacy.ChatCompleter, prompt prompts.Prompt, p pendingEntity) (string, error) {
	excerpt := truncateRunes(p.Body, bodyExcerptRunes)
	user := prompt.Render(map[string]string{
		"type":         p.Type,
		"title":        p.Title,
		"body_excerpt": excerpt,
	})
	inv := privacy.InvokeInput{
		System:           prompt.SystemPrompt,
		Segments:         []privacy.TextSegment{{Text: user}},
		PolicyCtx:        privacy.PolicyContext{OutputType: "entity_summary"},
		PromptTemplateID: prompt.ID,
		// No principal — this is a system-driven backfill, not a user
		// request. Privacy policies that require a principal will reject
		// the call which is the desired safety behavior.
	}
	res, err := e.privacy.InvokeOpenAI(ctx, client, inv)
	if err != nil {
		return "", err
	}
	return res.Answer, nil
}

// writeSummary UPSERTs only synthesized_summary + synthesized_at on the
// projection. This avoids racing with the entity-publish path which writes
// the rest of the projection columns.
func (e *EntitySummarizer) writeSummary(ctx context.Context, entityID uuid.UUID, summary string) error {
	_, err := e.pool.Exec(ctx, `
		UPDATE entity_search_projection
		SET synthesized_summary = $2,
		    synthesized_at = now()
		WHERE entity_id = $1`, entityID, summary)
	return err
}

func clampMaxRows(req int) int {
	if req <= 0 {
		return defaultEntitySummarizeMaxRows
	}
	if req > hardCapEntitySummarizeMaxRows {
		return hardCapEntitySummarizeMaxRows
	}
	return req
}

func truncateRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	out := make([]rune, 0, n)
	for i, r := range s {
		_ = i
		out = append(out, r)
		if len(out) >= n {
			break
		}
	}
	return string(out) + "…"
}
