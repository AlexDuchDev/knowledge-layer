package knowledge_jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// JobOperatorPrincipal is a user (or future team) allowed to run the job explicitly.
type JobOperatorPrincipal struct {
	PrincipalType string    `json:"principal_type"`
	PrincipalID   uuid.UUID `json:"principal_id"`
}

// ScenarioJobBindingWrite links a scenario to this job.
type ScenarioJobBindingWrite struct {
	ScenarioID   uuid.UUID `json:"scenario_id"`
	Relationship string    `json:"relationship"`
}

// Update applies a partial patch to a job definition and re-syncs source bindings.
func (s *JobService) Update(ctx context.Context, id uuid.UUID, p PatchJobInput) (*KnowledgeJob, error) {
	cur, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	next := mergePatchKnowledgeJob(cur, p)
	vin := jobToCreateInput(next)
	if err := ValidateUpdateJobInput(vin); err != nil {
		return nil, err
	}
	triggers, err := s.ListTriggers(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := ValidateTriggerRowsForPrimaryType(next.TriggerType, triggers); err != nil {
		return nil, err
	}

	pub := NormalizePublicationMode(NormalizePublicationModeForStorage(next.PublicationMode))
	var tk interface{}
	if next.TemplateKey != nil && *next.TemplateKey != "" {
		tk = *next.TemplateKey
	}
	var cloneRef interface{}
	if next.ClonedFromJobID != nil {
		cloneRef = *next.ClonedFromJobID
	} else {
		cloneRef = nil
	}

	_, err = s.pool.Exec(ctx, `
		UPDATE knowledge_jobs SET
			name = $2, purpose = $3, description = $4,
			operator_scope_json = $5, source_scope_json = $6, trigger_type = $7,
			output_domain_id = $8, output_sensitivity_level = $9, publication_mode = $10,
			review_required = $11, citations_required = $12, provenance_required = $13,
			scenario_only_exposure = $14, allow_domain_run_job = $15, processing_mode = $16,
			config_json = $17, status = $18, template_key = $19, cloned_from_job_id = $20,
			updated_at = now()
		WHERE id = $1`,
		id, next.Name, next.Purpose, next.Description,
		next.OperatorScopeJSON, next.SourceScopeJSON, next.TriggerType,
		next.OutputDomainID, next.OutputSensitivity, pub,
		next.ReviewRequired, next.CitationsRequired, next.ProvenanceRequired,
		next.ScenarioOnlyExposure, next.AllowDomainRunJob, next.ProcessingMode,
		next.ConfigJSON, next.Status, tk, cloneRef)
	if err != nil {
		return nil, err
	}
	if err := s.resyncJobSources(ctx, id, next.SourceScopeJSON); err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

func mergePatchKnowledgeJob(cur *KnowledgeJob, p PatchJobInput) *KnowledgeJob {
	next := *cur
	if p.Name != nil {
		next.Name = strings.TrimSpace(*p.Name)
	}
	if p.Purpose != nil {
		next.Purpose = p.Purpose
	}
	if p.Description != nil {
		next.Description = p.Description
	}
	if len(p.SourceScopeJSON) > 0 && string(p.SourceScopeJSON) != "null" {
		next.SourceScopeJSON = p.SourceScopeJSON
	}
	if len(p.OperatorScopeJSON) > 0 && string(p.OperatorScopeJSON) != "null" {
		next.OperatorScopeJSON = p.OperatorScopeJSON
	}
	if p.TriggerType != nil {
		next.TriggerType = strings.TrimSpace(*p.TriggerType)
	}
	if p.OutputDomainID != nil {
		next.OutputDomainID = p.OutputDomainID
	}
	if p.OutputSensitivity != nil {
		next.OutputSensitivity = *p.OutputSensitivity
	}
	if p.PublicationMode != nil {
		next.PublicationMode = strings.TrimSpace(*p.PublicationMode)
	}
	if p.ReviewRequired != nil {
		next.ReviewRequired = *p.ReviewRequired
	}
	if p.CitationsRequired != nil {
		next.CitationsRequired = *p.CitationsRequired
	}
	if p.ProvenanceRequired != nil {
		next.ProvenanceRequired = *p.ProvenanceRequired
	}
	if p.ScenarioOnlyExposure != nil {
		next.ScenarioOnlyExposure = *p.ScenarioOnlyExposure
	}
	if p.AllowDomainRunJob != nil {
		next.AllowDomainRunJob = *p.AllowDomainRunJob
	}
	if p.ProcessingMode != nil {
		next.ProcessingMode = strings.TrimSpace(*p.ProcessingMode)
	}
	if len(p.ConfigJSON) > 0 && string(p.ConfigJSON) != "null" {
		next.ConfigJSON = p.ConfigJSON
	}
	if p.Status != nil {
		next.Status = strings.TrimSpace(*p.Status)
	}
	return &next
}

// Clone duplicates a job (no scenario bindings). New owner is principal.
func (s *JobService) Clone(ctx context.Context, srcID uuid.UUID, newOwner uuid.UUID) (*KnowledgeJob, error) {
	src, err := s.Get(ctx, srcID)
	if err != nil {
		return nil, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	newID := uuid.New()
	name := strings.TrimSpace(src.Name) + " (copy)"
	var tk interface{}
	if src.TemplateKey != nil {
		tk = *src.TemplateKey
	}
	cloneParent := src.ID
	pub := NormalizePublicationMode(NormalizePublicationModeForStorage(src.PublicationMode))
	var spc interface{}
	if src.SourcePresetCode != nil {
		spc = *src.SourcePresetCode
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO knowledge_jobs (
			id, name, job_type, purpose, description, owner_id, operator_scope_json, source_scope_json, trigger_type,
			output_type, output_domain_id, output_sensitivity_level, publication_mode, review_required, approval_required,
			sanitization_rules_json, config_json, status,
			citations_required, provenance_required, scenario_only_exposure, allow_domain_run_job, template_key, processing_mode,
			cloned_from_job_id, source_preset_code)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,'artifact',$10,$11,$12,$13,$14,false,'{}',$15,'draft',
			$16,$17,$18,$19,$20,$21,$22,$23,$24)`,
		newID, name, src.JobType, src.Purpose, src.Description, newOwner, src.OperatorScopeJSON, src.SourceScopeJSON, src.TriggerType,
		src.OutputDomainID, src.OutputSensitivity, pub, src.ReviewRequired, src.ConfigJSON,
		src.CitationsRequired, src.ProvenanceRequired, src.ScenarioOnlyExposure, src.AllowDomainRunJob, tk, src.ProcessingMode,
		cloneParent, spc)
	if err != nil {
		return nil, err
	}

	rows, err := tx.Query(ctx, `
		SELECT trigger_type, schedule_expr, event_filter_json, window_config_json, status
		FROM job_triggers WHERE knowledge_job_id = $1`, srcID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var tt, st string
		var sched *string
		var ev, win json.RawMessage
		if err := rows.Scan(&tt, &sched, &ev, &win, &st); err != nil {
			return nil, err
		}
		tid := uuid.New()
		_, err = tx.Exec(ctx, `
			INSERT INTO job_triggers (id, knowledge_job_id, trigger_type, schedule_expr, event_filter_json, window_config_json, status)
			VALUES ($1,$2,$3,$4,$5,$6,$7)`, tid, newID, tt, sched, ev, win, st)
		if err != nil {
			return nil, err
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	opRows, err := tx.Query(ctx, `
		SELECT principal_type, principal_id FROM knowledge_job_operators WHERE knowledge_job_id = $1`, srcID)
	if err != nil {
		return nil, err
	}
	defer opRows.Close()
	for opRows.Next() {
		var pt string
		var pid uuid.UUID
		if err := opRows.Scan(&pt, &pid); err != nil {
			return nil, err
		}
		oid := uuid.New()
		_, err = tx.Exec(ctx, `
			INSERT INTO knowledge_job_operators (id, knowledge_job_id, principal_type, principal_id)
			VALUES ($1,$2,$3,$4)`, oid, newID, pt, pid)
		if err != nil {
			return nil, err
		}
	}
	if err := opRows.Err(); err != nil {
		return nil, err
	}

	srcRows, err := tx.Query(ctx, `
		SELECT source_type, source_id FROM knowledge_job_sources WHERE knowledge_job_id = $1`, srcID)
	if err != nil {
		return nil, err
	}
	defer srcRows.Close()
	for srcRows.Next() {
		var st string
		var sid uuid.UUID
		if err := srcRows.Scan(&st, &sid); err != nil {
			return nil, err
		}
		kid := uuid.New()
		_, err = tx.Exec(ctx, `
			INSERT INTO knowledge_job_sources (id, knowledge_job_id, source_type, source_id)
			VALUES ($1,$2,$3,$4)`, kid, newID, st, sid)
		if err != nil {
			return nil, err
		}
	}
	if err := srcRows.Err(); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.Get(ctx, newID)
}

// ListOperators returns explicit operator principals for a job.
func (s *JobService) ListOperators(ctx context.Context, jobID uuid.UUID) ([]JobOperatorPrincipal, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT principal_type, principal_id FROM knowledge_job_operators
		WHERE knowledge_job_id = $1 ORDER BY principal_type, principal_id`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []JobOperatorPrincipal
	for rows.Next() {
		var o JobOperatorPrincipal
		if err := rows.Scan(&o.PrincipalType, &o.PrincipalID); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// ReplaceOperators replaces all operator rows for principal_type = user (v1).
func (s *JobService) ReplaceOperators(ctx context.Context, jobID uuid.UUID, users []uuid.UUID) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	_, err = tx.Exec(ctx, `DELETE FROM knowledge_job_operators WHERE knowledge_job_id = $1 AND principal_type = 'user'`, jobID)
	if err != nil {
		return err
	}
	for _, uid := range users {
		if uid == uuid.Nil {
			continue
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO knowledge_job_operators (id, knowledge_job_id, principal_type, principal_id)
			VALUES ($1,$2,'user',$3)`, uuid.New(), jobID, uid)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// ListScenarioBindingsForJob lists scenarios linked to this job.
func (s *JobService) ListScenarioBindingsForJob(ctx context.Context, jobID uuid.UUID) ([]ScenarioJobBindingWrite, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT scenario_id, relationship FROM scenario_job_bindings WHERE knowledge_job_id = $1`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ScenarioJobBindingWrite
	for rows.Next() {
		var r ScenarioJobBindingWrite
		if err := rows.Scan(&r.ScenarioID, &r.Relationship); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ReplaceScenarioBindings replaces all scenario_job_bindings for this job.
func (s *JobService) ReplaceScenarioBindings(ctx context.Context, jobID uuid.UUID, rowsIn []ScenarioJobBindingWrite) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, `DELETE FROM scenario_job_bindings WHERE knowledge_job_id = $1`, jobID)
	if err != nil {
		return err
	}
	for _, row := range rowsIn {
		rel := row.Relationship
		if rel == "" {
			rel = "supports"
		}
		switch rel {
		case "primary_support", "supports", "optional":
		default:
			return fmt.Errorf("invalid job relationship: %s", row.Relationship)
		}
		var n int
		if err := tx.QueryRow(ctx, `SELECT COUNT(1) FROM scenario_definitions WHERE id = $1`, row.ScenarioID).Scan(&n); err != nil {
			return err
		}
		if n == 0 {
			return fmt.Errorf("scenario not found: %s", row.ScenarioID)
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO scenario_job_bindings (scenario_id, knowledge_job_id, relationship)
			VALUES ($1,$2,$3)`, row.ScenarioID, jobID, rel)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// TestRunDry validates definition and returns preview without creating a job run.
func (s *JobService) TestRunDry(ctx context.Context, jobID uuid.UUID) (*JobPreview, error) {
	return s.BuildJobPreview(ctx, jobID)
}
