package knowledge_jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/knowledgelayer/api/internal/knowledge_core"
)

// RunDecisionExtraction builds a derived Insight listing candidate decisions found in normalized records.
// v1 implementation is heuristic (no LLM). If sent to an LLM later, route via PrivacyGateway.
func (d *DigestRunner) RunDecisionExtraction(ctx context.Context, runID uuid.UUID, job *KnowledgeJob, operator uuid.UUID) error {
	var scope struct {
		SourceFeedID uuid.UUID `json:"source_feed_id"`
		DomainID     uuid.UUID `json:"domain_id"`
	}
	if err := json.Unmarshal(job.SourceScopeJSON, &scope); err != nil {
		return fmt.Errorf("decision_extraction: scope: %w", err)
	}
	if scope.SourceFeedID == uuid.Nil || scope.DomainID == uuid.Nil {
		return fmt.Errorf("decision_extraction: source_feed_id and domain_id required in source_scope_json")
	}

	ok, err := JobAllowsSourceFeed(ctx, d.pool, job.ID, scope.SourceFeedID)
	if err != nil {
		return fmt.Errorf("decision_extraction: source allow check: %w", err)
	}
	if !ok {
		return fmt.Errorf("decision_extraction: source feed %s is not declared for this job", scope.SourceFeedID)
	}

	windowDays := 7
	if len(job.ConfigJSON) > 0 {
		var cfg struct {
			WindowDays *int `json:"window_days"`
		}
		_ = json.Unmarshal(job.ConfigJSON, &cfg)
		if cfg.WindowDays != nil && *cfg.WindowDays > 0 && *cfg.WindowDays <= 90 {
			windowDays = *cfg.WindowDays
		}
	}

	since := time.Now().UTC().Add(-time.Duration(windowDays) * 24 * time.Hour)
	rows, err := d.pool.Query(ctx, `
		SELECT structured_payload_json FROM normalized_records
		WHERE source_feed_id = $1 AND created_at >= $2
		ORDER BY created_at ASC
		LIMIT 1000`,
		scope.SourceFeedID, since)
	if err != nil {
		return err
	}
	defer rows.Close()

	var (
		recordCount int
		matches     []string
	)
	for rows.Next() {
		recordCount++
		var raw json.RawMessage
		if err := rows.Scan(&raw); err != nil {
			return err
		}
		s := strings.ToLower(string(raw))
		if strings.Contains(s, "decision") || strings.Contains(s, "decided") || strings.Contains(s, "we will") || strings.Contains(s, "agreed") {
			matches = append(matches, string(raw))
		}
		if len(matches) >= 50 {
			break
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	plan := PlanWeeklyDigestOutput(job)

	body := fmt.Sprintf(
		"Decision extraction (%d records scanned, %d matches, window_days=%d)\n---\n%s",
		recordCount,
		len(matches),
		windowDays,
		joinLimit(matches, 30),
	)

	ent, err := d.entities.Create(ctx, knowledge_core.CreateEntityInput{
		Type:             "Insight",
		Title:            fmt.Sprintf("Decision candidates for feed %s", scope.SourceFeedID.String()[:8]),
		Body:             &body,
		OwnerID:          &operator,
		DomainID:         scope.DomainID,
		SensitivityLevel: job.OutputSensitivity,
		TruthMode:        "derived",
		LifecycleState:   plan.EntityLifecycle,
		PayloadJSON:      json.RawMessage(`{"kind":"decision_extraction","record_count":` + fmtInt(recordCount) + `,"match_count":` + fmtInt(len(matches)) + `}`),
	})
	if err != nil {
		return err
	}

	ref := "decision-extraction-" + runID.String()
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

	payload := map[string]any{"entity_id": ent.ID.String(), "kind": "decision_extraction"}
	payloadBytes, _ := json.Marshal(payload)
	_, err = d.pool.Exec(ctx, `
		INSERT INTO job_outputs (job_run_id, output_type, structured_payload_json, target_entity_id, target_entity_type, review_task_id, publication_status)
		VALUES ($1,'decision_extraction',$2,$3,'entity',$4,$5)`,
		runID, payloadBytes, ent.ID, reviewTaskID, plan.JobOutputPublicationStatus)
	return err
}

// RunStaleScan builds a derived Insight listing stale or review-due entities in a domain.
func (d *DigestRunner) RunStaleScan(ctx context.Context, runID uuid.UUID, job *KnowledgeJob, operator uuid.UUID) error {
	var scope struct {
		DomainID   uuid.UUID `json:"domain_id"`
		EntityType *string   `json:"entity_type"`
	}
	if err := json.Unmarshal(job.SourceScopeJSON, &scope); err != nil {
		return fmt.Errorf("stale_scan: scope: %w", err)
	}
	if scope.DomainID == uuid.Nil {
		return fmt.Errorf("stale_scan: domain_id required in source_scope_json")
	}

	maxAgeDays := 365
	if len(job.ConfigJSON) > 0 {
		var cfg struct {
			MaxAgeDays *int `json:"max_age_days"`
		}
		_ = json.Unmarshal(job.ConfigJSON, &cfg)
		if cfg.MaxAgeDays != nil && *cfg.MaxAgeDays > 0 && *cfg.MaxAgeDays <= 3650 {
			maxAgeDays = *cfg.MaxAgeDays
		}
	}

	cutoff := time.Now().UTC().Add(-time.Duration(maxAgeDays) * 24 * time.Hour)
	etype := ""
	if scope.EntityType != nil {
		etype = strings.TrimSpace(*scope.EntityType)
	}

	q := `
		SELECT id, type, title, updated_at, freshness_status, approval_status
		FROM entities
		WHERE domain_id = $1
		  AND archived_at IS NULL
		  AND (freshness_status IN ('stale','review_due') OR updated_at < $2)`
	args := []any{scope.DomainID, cutoff}
	if etype != "" {
		q += ` AND type = $3`
		args = append(args, etype)
	}
	q += ` ORDER BY updated_at ASC LIMIT 300`

	rows, err := d.pool.Query(ctx, q, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	type item struct {
		ID             uuid.UUID
		Type           string
		Title          string
		UpdatedAt      time.Time
		Freshness      string
		ApprovalStatus string
	}
	var items []item
	for rows.Next() {
		var it item
		if err := rows.Scan(&it.ID, &it.Type, &it.Title, &it.UpdatedAt, &it.Freshness, &it.ApprovalStatus); err != nil {
			return err
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	lines := make([]string, 0, len(items))
	for _, it := range items {
		lines = append(lines, fmt.Sprintf("- %s [%s] (%s) updated=%s approval=%s",
			it.Title, it.Type, it.ID.String()[:8], it.UpdatedAt.Format("2006-01-02"), it.ApprovalStatus))
	}

	plan := PlanWeeklyDigestOutput(job)
	body := fmt.Sprintf("Stale scan (%d items, max_age_days=%d)\n---\n%s", len(items), maxAgeDays, joinLimit(lines, 200))

	ent, err := d.entities.Create(ctx, knowledge_core.CreateEntityInput{
		Type:             "Insight",
		Title:            "Stale knowledge scan",
		Body:             &body,
		OwnerID:          &operator,
		DomainID:         scope.DomainID,
		SensitivityLevel: job.OutputSensitivity,
		TruthMode:        "derived",
		LifecycleState:   plan.EntityLifecycle,
		PayloadJSON:      json.RawMessage(`{"kind":"stale_scan","item_count":` + fmtInt(len(items)) + `,"max_age_days":` + fmtInt(maxAgeDays) + `}`),
	})
	if err != nil {
		return err
	}

	ref := "stale-scan-" + runID.String()
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

	payload := map[string]any{"entity_id": ent.ID.String(), "kind": "stale_scan"}
	payloadBytes, _ := json.Marshal(payload)
	_, err = d.pool.Exec(ctx, `
		INSERT INTO job_outputs (job_run_id, output_type, structured_payload_json, target_entity_id, target_entity_type, review_task_id, publication_status)
		VALUES ($1,'stale_scan',$2,$3,'entity',$4,$5)`,
		runID, payloadBytes, ent.ID, reviewTaskID, plan.JobOutputPublicationStatus)
	return err
}

// RunPlanningSummary builds a derived Insight highlighting planning-oriented signals from one source feed.
func (d *DigestRunner) RunPlanningSummary(ctx context.Context, runID uuid.UUID, job *KnowledgeJob, operator uuid.UUID) error {
	var scope struct {
		SourceFeedID uuid.UUID `json:"source_feed_id"`
		DomainID     uuid.UUID `json:"domain_id"`
	}
	if err := json.Unmarshal(job.SourceScopeJSON, &scope); err != nil {
		return fmt.Errorf("planning_summary: scope: %w", err)
	}
	if scope.SourceFeedID == uuid.Nil || scope.DomainID == uuid.Nil {
		return fmt.Errorf("planning_summary: source_feed_id and domain_id required in source_scope_json")
	}
	ok, err := JobAllowsSourceFeed(ctx, d.pool, job.ID, scope.SourceFeedID)
	if err != nil {
		return fmt.Errorf("planning_summary: source allow check: %w", err)
	}
	if !ok {
		return fmt.Errorf("planning_summary: source feed %s is not declared for this job", scope.SourceFeedID)
	}

	windowDays := 14
	if len(job.ConfigJSON) > 0 {
		var cfg struct {
			WindowDays *int `json:"window_days"`
		}
		_ = json.Unmarshal(job.ConfigJSON, &cfg)
		if cfg.WindowDays != nil && *cfg.WindowDays > 0 && *cfg.WindowDays <= 90 {
			windowDays = *cfg.WindowDays
		}
	}
	since := time.Now().UTC().Add(-time.Duration(windowDays) * 24 * time.Hour)
	rows, err := d.pool.Query(ctx, `
		SELECT structured_payload_json FROM normalized_records
		WHERE source_feed_id = $1 AND created_at >= $2
		ORDER BY created_at ASC
		LIMIT 1000`,
		scope.SourceFeedID, since)
	if err != nil {
		return err
	}
	defer rows.Close()

	keywords := []string{"plan", "planning", "next step", "roadmap", "milestone", "todo", "action item", "owner", "deliver"}
	var (
		recordCount int
		matches     []string
	)
	for rows.Next() {
		recordCount++
		var raw json.RawMessage
		if err := rows.Scan(&raw); err != nil {
			return err
		}
		s := strings.ToLower(string(raw))
		for _, kw := range keywords {
			if strings.Contains(s, kw) {
				matches = append(matches, string(raw))
				break
			}
		}
		if len(matches) >= 50 {
			break
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	body := fmt.Sprintf(
		"Planning summary (%d records scanned, %d planning signals, window_days=%d)\n---\n%s",
		recordCount,
		len(matches),
		windowDays,
		joinLimit(matches, 30),
	)
	return d.persistDerivedInsight(ctx, runID, job, operator, scope.DomainID, "Planning summary", body, "planning_summary", map[string]any{
		"record_count": recordCount,
		"match_count":  len(matches),
		"window_days":  windowDays,
	})
}

// RunSupportTrendsExtraction builds a derived Insight summarizing repeated support-oriented themes from one feed.
func (d *DigestRunner) RunSupportTrendsExtraction(ctx context.Context, runID uuid.UUID, job *KnowledgeJob, operator uuid.UUID) error {
	var scope struct {
		SourceFeedID uuid.UUID `json:"source_feed_id"`
		DomainID     uuid.UUID `json:"domain_id"`
	}
	if err := json.Unmarshal(job.SourceScopeJSON, &scope); err != nil {
		return fmt.Errorf("support_trends_extraction: scope: %w", err)
	}
	if scope.SourceFeedID == uuid.Nil || scope.DomainID == uuid.Nil {
		return fmt.Errorf("support_trends_extraction: source_feed_id and domain_id required in source_scope_json")
	}
	ok, err := JobAllowsSourceFeed(ctx, d.pool, job.ID, scope.SourceFeedID)
	if err != nil {
		return fmt.Errorf("support_trends_extraction: source allow check: %w", err)
	}
	if !ok {
		return fmt.Errorf("support_trends_extraction: source feed %s is not declared for this job", scope.SourceFeedID)
	}

	windowDays := 14
	if len(job.ConfigJSON) > 0 {
		var cfg struct {
			WindowDays *int `json:"window_days"`
		}
		_ = json.Unmarshal(job.ConfigJSON, &cfg)
		if cfg.WindowDays != nil && *cfg.WindowDays > 0 && *cfg.WindowDays <= 90 {
			windowDays = *cfg.WindowDays
		}
	}
	since := time.Now().UTC().Add(-time.Duration(windowDays) * 24 * time.Hour)
	rows, err := d.pool.Query(ctx, `
		SELECT structured_payload_json FROM normalized_records
		WHERE source_feed_id = $1 AND created_at >= $2
		ORDER BY created_at ASC
		LIMIT 1000`,
		scope.SourceFeedID, since)
	if err != nil {
		return err
	}
	defer rows.Close()

	themeKeywords := map[string][]string{
		"access":      {"login", "access", "permission", "auth", "signin"},
		"billing":     {"billing", "invoice", "payment", "refund", "charge"},
		"reliability": {"bug", "error", "outage", "broken", "fail"},
		"performance": {"slow", "latency", "timeout", "performance"},
		"onboarding":  {"setup", "install", "onboard", "configure"},
		"requests":    {"feature request", "wishlist", "could you", "would love"},
	}
	themeCounts := map[string]int{}
	recordCount := 0
	for rows.Next() {
		recordCount++
		var raw json.RawMessage
		if err := rows.Scan(&raw); err != nil {
			return err
		}
		s := strings.ToLower(string(raw))
		for theme, kws := range themeKeywords {
			for _, kw := range kws {
				if strings.Contains(s, kw) {
					themeCounts[theme]++
					break
				}
			}
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	type themeCount struct {
		Theme string
		Count int
	}
	ordered := make([]themeCount, 0, len(themeCounts))
	for theme, count := range themeCounts {
		if count > 0 {
			ordered = append(ordered, themeCount{Theme: theme, Count: count})
		}
	}
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].Count == ordered[j].Count {
			return ordered[i].Theme < ordered[j].Theme
		}
		return ordered[i].Count > ordered[j].Count
	})
	lines := make([]string, 0, len(ordered))
	for _, item := range ordered {
		lines = append(lines, fmt.Sprintf("- %s: %d mentions", item.Theme, item.Count))
	}
	body := fmt.Sprintf(
		"Support trends extraction (%d records scanned, %d recurring themes, window_days=%d)\n---\n%s",
		recordCount,
		len(ordered),
		windowDays,
		joinLimit(lines, 20),
	)
	return d.persistDerivedInsight(ctx, runID, job, operator, scope.DomainID, "Support trends extraction", body, "support_trends_extraction", map[string]any{
		"record_count": recordCount,
		"theme_counts": themeCounts,
		"window_days":  windowDays,
	})
}

func (d *DigestRunner) persistDerivedInsight(
	ctx context.Context,
	runID uuid.UUID,
	job *KnowledgeJob,
	operator uuid.UUID,
	domainID uuid.UUID,
	title string,
	body string,
	outputType string,
	payload map[string]any,
) error {
	plan := PlanWeeklyDigestOutput(job)
	payload["kind"] = outputType
	payloadBytes, _ := json.Marshal(payload)
	ent, err := d.entities.Create(ctx, knowledge_core.CreateEntityInput{
		Type:             "Insight",
		Title:            title,
		Body:             &body,
		OwnerID:          &operator,
		DomainID:         domainID,
		SensitivityLevel: job.OutputSensitivity,
		TruthMode:        "derived",
		LifecycleState:   plan.EntityLifecycle,
		PayloadJSON:      payloadBytes,
	})
	if err != nil {
		return err
	}

	ref := outputType + "-" + runID.String()
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

	outputPayload, _ := json.Marshal(map[string]any{"entity_id": ent.ID.String(), "kind": outputType})
	_, err = d.pool.Exec(ctx, `
		INSERT INTO job_outputs (job_run_id, output_type, structured_payload_json, target_entity_id, target_entity_type, review_task_id, publication_status)
		VALUES ($1,$2,$3,$4,'entity',$5,$6)`,
		runID, outputType, outputPayload, ent.ID, reviewTaskID, plan.JobOutputPublicationStatus)
	return err
}
