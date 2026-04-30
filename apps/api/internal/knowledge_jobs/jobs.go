package knowledge_jobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/knowledgelayer/api/internal/platform/queue"
)

// ErrJobMissingName is returned when name is empty after template resolution.
var ErrJobMissingName = errors.New("job name required")

// ErrUnimplementedJobType is returned when job_type has no runtime processor (fail-closed on create/update).
var ErrUnimplementedJobType = errors.New("job_type has no runtime processor")

// ErrJobMissingType is returned when job_type is empty after template resolution.
var ErrJobMissingType = errors.New("job_type required")

type KnowledgeJob struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	// JobType is the processing class (e.g. weekly_digest); catalog templates set this via template_id.
	JobType              string          `json:"job_type"`
	ProcessorImplemented bool            `json:"processor_implemented"`
	Purpose              *string         `json:"purpose,omitempty"`
	Description          *string         `json:"description,omitempty"`
	OwnerID              uuid.UUID       `json:"owner_id"`
	SourceScopeJSON      json.RawMessage `json:"source_scope_json"`
	OperatorScopeJSON    json.RawMessage `json:"operator_scope_json"`
	TriggerType          string          `json:"trigger_type"`
	OutputDomainID       *uuid.UUID      `json:"output_domain_id,omitempty"`
	OutputSensitivity    int             `json:"output_sensitivity_level"`
	PublicationMode      string          `json:"publication_mode"`
	ReviewRequired       bool            `json:"review_required"`
	CitationsRequired    bool            `json:"citations_required"`
	ProvenanceRequired   bool            `json:"provenance_required"`
	ScenarioOnlyExposure bool            `json:"scenario_only_exposure"`
	AllowDomainRunJob    bool            `json:"allow_domain_run_job"`
	TemplateKey          *string         `json:"template_key,omitempty"`
	ProcessingMode       string          `json:"processing_mode"`
	ClonedFromJobID      *uuid.UUID      `json:"cloned_from_job_id,omitempty"`
	SourcePresetCode     *string         `json:"source_preset_code,omitempty"`
	ConfigJSON           json.RawMessage `json:"config_json"`
	Status               string          `json:"status"`
}

// JobListItem is a job row plus optional scenario summary for admin lists.
type JobListItem struct {
	KnowledgeJob
	ScenarioBindingCount int      `json:"scenario_binding_count,omitempty"`
	ScenarioCodes        []string `json:"scenario_codes,omitempty"`
}

type JobRun struct {
	ID                     uuid.UUID       `json:"id"`
	KnowledgeJobID         uuid.UUID       `json:"knowledge_job_id"`
	Status                 string          `json:"status"`
	InputScopeSnapshotJSON json.RawMessage `json:"input_scope_snapshot_json"`
}

type JobService struct {
	pool       *pgxpool.Pool
	digest     *DigestRunner
	summarizer *EntitySummarizer
	publish    *queue.Publisher
	orch       *runOrchestrator
}

// NewJobService wires the orchestrator. summarizer may be nil when no LLM
// provider is configured (local dev without OPENAI_API_KEY); the orchestrator
// returns a clear error in that case rather than silently no-op'ing.
func NewJobService(pool *pgxpool.Pool, digest *DigestRunner, summarizer *EntitySummarizer, publish *queue.Publisher) *JobService {
	return &JobService{
		pool:       pool,
		digest:     digest,
		summarizer: summarizer,
		publish:    publish,
		orch:       newRunOrchestrator(pool, digest, summarizer),
	}
}

func (s *JobService) List(ctx context.Context) ([]KnowledgeJob, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, job_type, purpose, description, owner_id, operator_scope_json, source_scope_json, trigger_type,
		       output_domain_id, output_sensitivity_level, publication_mode, review_required, citations_required, provenance_required,
		       scenario_only_exposure, allow_domain_run_job, template_key, processing_mode, cloned_from_job_id, source_preset_code, config_json, status
		FROM knowledge_jobs ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []KnowledgeJob
	for rows.Next() {
		j, err := scanKnowledgeJobRow(rows)
		if err != nil {
			return nil, err
		}
		normalizeJobForAPI(&j)
		list = append(list, j)
	}
	return list, rows.Err()
}

// ListWithScenarioSummary returns jobs with scenario binding counts and codes (one extra query).
func (s *JobService) ListWithScenarioSummary(ctx context.Context) ([]JobListItem, error) {
	jobs, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	if len(jobs) == 0 {
		return nil, nil
	}
	ids := make([]uuid.UUID, len(jobs))
	for i := range jobs {
		ids[i] = jobs[i].ID
	}
	summary, err := s.scenarioCodesByJobIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]JobListItem, len(jobs))
	for i := range jobs {
		out[i].KnowledgeJob = jobs[i]
		if sc, ok := summary[jobs[i].ID]; ok {
			out[i].ScenarioCodes = sc
			out[i].ScenarioBindingCount = len(sc)
		}
	}
	return out, nil
}

func (s *JobService) scenarioCodesByJobIDs(ctx context.Context, jobIDs []uuid.UUID) (map[uuid.UUID][]string, error) {
	if len(jobIDs) == 0 {
		return map[uuid.UUID][]string{}, nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT sjb.knowledge_job_id, array_agg(sd.code ORDER BY sd.code) AS codes
		FROM scenario_job_bindings sjb
		JOIN scenario_definitions sd ON sd.id = sjb.scenario_id
		WHERE sjb.knowledge_job_id = ANY($1::uuid[])
		GROUP BY sjb.knowledge_job_id`, jobIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[uuid.UUID][]string)
	for rows.Next() {
		var jid uuid.UUID
		var codes []string
		if err := rows.Scan(&jid, &codes); err != nil {
			return nil, err
		}
		if codes == nil {
			codes = []string{}
		}
		out[jid] = codes
	}
	return out, rows.Err()
}

func (s *JobService) Get(ctx context.Context, id uuid.UUID) (*KnowledgeJob, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, name, job_type, purpose, description, owner_id, operator_scope_json, source_scope_json, trigger_type,
		       output_domain_id, output_sensitivity_level, publication_mode, review_required, citations_required, provenance_required,
		       scenario_only_exposure, allow_domain_run_job, template_key, processing_mode, cloned_from_job_id, source_preset_code, config_json, status
		FROM knowledge_jobs WHERE id=$1`, id)
	j, err := scanKnowledgeJobRow(row)
	if err != nil {
		return nil, err
	}
	normalizeJobForAPI(&j)
	return &j, nil
}

type jobRowScanner interface {
	Scan(dest ...any) error
}

func scanKnowledgeJobRow(row jobRowScanner) (KnowledgeJob, error) {
	var j KnowledgeJob
	var tk sql.NullString
	var cid pgtype.UUID
	var spc sql.NullString
	err := row.Scan(&j.ID, &j.Name, &j.JobType, &j.Purpose, &j.Description, &j.OwnerID, &j.OperatorScopeJSON, &j.SourceScopeJSON, &j.TriggerType,
		&j.OutputDomainID, &j.OutputSensitivity, &j.PublicationMode, &j.ReviewRequired, &j.CitationsRequired, &j.ProvenanceRequired,
		&j.ScenarioOnlyExposure, &j.AllowDomainRunJob, &tk, &j.ProcessingMode, &cid, &spc, &j.ConfigJSON, &j.Status)
	if err != nil {
		return j, err
	}
	if tk.Valid {
		v := tk.String
		j.TemplateKey = &v
	}
	if cid.Valid {
		u := uuid.UUID(cid.Bytes)
		j.ClonedFromJobID = &u
	}
	if spc.Valid && spc.String != "" {
		v := spc.String
		j.SourcePresetCode = &v
	}
	if len(j.OperatorScopeJSON) == 0 {
		j.OperatorScopeJSON = []byte("{}")
	}
	return j, nil
}

func normalizeJobForAPI(j *KnowledgeJob) {
	j.PublicationMode = MergeNormalizedPublicationMode(j.PublicationMode)
	j.ProcessorImplemented = IsKnowledgeJobProcessorImplemented(j.JobType)
}

func boolPtrDefault(p *bool, def bool) bool {
	if p == nil {
		return def
	}
	return *p
}

type CreateJobInput struct {
	TemplateID           string          `json:"template_id"`
	Name                 string          `json:"name"`
	JobType              string          `json:"job_type"`
	Purpose              *string         `json:"purpose"`
	Description          *string         `json:"description"`
	OwnerID              uuid.UUID       `json:"owner_id"`
	SourceScopeJSON      json.RawMessage `json:"source_scope_json"`
	OperatorScopeJSON    json.RawMessage `json:"operator_scope_json"`
	TriggerType          string          `json:"trigger_type"`
	OutputDomainID       *uuid.UUID      `json:"output_domain_id"`
	OutputSensitivity    int             `json:"output_sensitivity_level"`
	PublicationMode      string          `json:"publication_mode"`
	ReviewRequired       bool            `json:"review_required"`
	CitationsRequired    bool            `json:"citations_required"`
	ProvenanceRequired   *bool           `json:"provenance_required,omitempty"`
	ScenarioOnlyExposure bool            `json:"scenario_only_exposure"`
	AllowDomainRunJob    *bool           `json:"allow_domain_run_job,omitempty"`
	ProcessingMode       string          `json:"processing_mode"`
	ConfigJSON           json.RawMessage `json:"config_json"`
	// SourcePresetCode is set when creating from job_builder_presets (not a generic API field).
	SourcePresetCode *string `json:"source_preset_code,omitempty"`
}

// PatchJobInput is a partial update payload (nil / empty means omit).
type PatchJobInput struct {
	Name                 *string         `json:"name"`
	Purpose              *string         `json:"purpose"`
	Description          *string         `json:"description"`
	SourceScopeJSON      json.RawMessage `json:"source_scope_json"`
	OperatorScopeJSON    json.RawMessage `json:"operator_scope_json"`
	TriggerType          *string         `json:"trigger_type"`
	OutputDomainID       *uuid.UUID      `json:"output_domain_id"`
	OutputSensitivity    *int            `json:"output_sensitivity_level"`
	PublicationMode      *string         `json:"publication_mode"`
	ReviewRequired       *bool           `json:"review_required"`
	CitationsRequired    *bool           `json:"citations_required"`
	ProvenanceRequired   *bool           `json:"provenance_required"`
	ScenarioOnlyExposure *bool           `json:"scenario_only_exposure"`
	AllowDomainRunJob    *bool           `json:"allow_domain_run_job"`
	ProcessingMode       *string         `json:"processing_mode"`
	ConfigJSON           json.RawMessage `json:"config_json"`
	Status               *string         `json:"status"`
}

func (s *JobService) Create(ctx context.Context, in CreateJobInput) (*KnowledgeJob, error) {
	var err error
	in, err = ApplyJobTemplate(in)
	if err != nil {
		return nil, err
	}
	if in.Name == "" {
		return nil, ErrJobMissingName
	}
	if in.JobType == "" {
		return nil, ErrJobMissingType
	}
	if strings.TrimSpace(in.PublicationMode) == "" {
		if in.ReviewRequired {
			in.PublicationMode = PublicationModeReviewedPublish
		} else {
			in.PublicationMode = PublicationModeDraftOnly
		}
	}
	if strings.TrimSpace(in.TriggerType) == "" {
		in.TriggerType = "manual"
	}
	if strings.TrimSpace(in.ProcessingMode) == "" {
		in.ProcessingMode = "summarize"
	}
	if len(in.ConfigJSON) == 0 {
		in.ConfigJSON = []byte("{}")
	}
	if len(in.OperatorScopeJSON) == 0 {
		in.OperatorScopeJSON = []byte("{}")
	}
	if err := ValidateCreateJobInput(in); err != nil {
		return nil, err
	}
	pub := NormalizePublicationMode(NormalizePublicationModeForStorage(in.PublicationMode))
	prov := boolPtrDefault(in.ProvenanceRequired, true)
	allow := boolPtrDefault(in.AllowDomainRunJob, true)
	id := uuid.New()
	var templateKey interface{}
	if strings.TrimSpace(in.TemplateID) != "" {
		templateKey = strings.TrimSpace(in.TemplateID)
	}
	var spc interface{}
	if in.SourcePresetCode != nil && strings.TrimSpace(*in.SourcePresetCode) != "" {
		spc = strings.TrimSpace(*in.SourcePresetCode)
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO knowledge_jobs (id, name, job_type, purpose, description, owner_id, operator_scope_json, source_scope_json, trigger_type,
			output_type, output_domain_id, output_sensitivity_level, publication_mode, review_required, approval_required,
			sanitization_rules_json, config_json, status,
			citations_required, provenance_required, scenario_only_exposure, allow_domain_run_job, template_key, processing_mode, source_preset_code)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,'artifact',$10,$11,$12,$13,false,'{}',$14,'draft',
			$15,$16,$17,$18,$19,$20,$21)`,
		id, in.Name, in.JobType, in.Purpose, in.Description, in.OwnerID, in.OperatorScopeJSON, in.SourceScopeJSON, in.TriggerType,
		in.OutputDomainID, in.OutputSensitivity, pub, in.ReviewRequired, in.ConfigJSON,
		in.CitationsRequired, prov, in.ScenarioOnlyExposure, allow, templateKey, in.ProcessingMode, spc)
	if err != nil {
		return nil, err
	}
	if err := s.syncJobSourcesFromScope(ctx, id, in.SourceScopeJSON); err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

func (s *JobService) syncJobSourcesFromScope(ctx context.Context, jobID uuid.UUID, sourceScope json.RawMessage) error {
	if len(sourceScope) == 0 {
		return nil
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(sourceScope, &m); err != nil {
		return fmt.Errorf("job source scope: %w", err)
	}
	raw, ok := m["source_feed_id"]
	if !ok {
		return nil
	}
	var idStr string
	if err := json.Unmarshal(raw, &idStr); err != nil {
		return fmt.Errorf("job source scope source_feed_id: %w", err)
	}
	feedID, err := uuid.Parse(strings.TrimSpace(idStr))
	if err != nil || feedID == uuid.Nil {
		return nil
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO knowledge_job_sources (knowledge_job_id, source_type, source_id)
		VALUES ($1,'source_feed',$2)
		ON CONFLICT (knowledge_job_id, source_type, source_id) DO NOTHING`,
		jobID, feedID)
	return err
}

func (s *JobService) Patch(ctx context.Context, id uuid.UUID, status *string) (*KnowledgeJob, error) {
	if status != nil {
		_, err := s.pool.Exec(ctx, `UPDATE knowledge_jobs SET status=$2, updated_at=now() WHERE id=$1`, id, *status)
		if err != nil {
			return nil, err
		}
	}
	return s.Get(ctx, id)
}

func (s *JobService) resyncJobSources(ctx context.Context, jobID uuid.UUID, sourceScope json.RawMessage) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM knowledge_job_sources WHERE knowledge_job_id = $1`, jobID)
	if err != nil {
		return err
	}
	return s.syncJobSourcesFromScope(ctx, jobID, sourceScope)
}

func (s *JobService) GetRun(ctx context.Context, id uuid.UUID) (*JobRun, error) {
	var r JobRun
	err := s.pool.QueryRow(ctx, `
		SELECT id, knowledge_job_id, status, input_scope_snapshot_json FROM job_runs WHERE id=$1`, id,
	).Scan(&r.ID, &r.KnowledgeJobID, &r.Status, &r.InputScopeSnapshotJSON)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *JobService) Run(ctx context.Context, jobID uuid.UUID, initiatedBy uuid.UUID) (*JobRun, error) {
	j, err := s.Get(ctx, jobID)
	if err != nil {
		return nil, err
	}
	runID := uuid.New()
	snapshot := j.SourceScopeJSON
	st := "running"
	if s.publish != nil && s.publish.Enabled() {
		st = "queued"
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO job_runs (id, knowledge_job_id, initiated_by_type, initiated_by_id, trigger_type, status, input_scope_snapshot_json, started_at)
		VALUES ($1,$2,'user',$3,'manual',$4,$5,now())`, runID, jobID, initiatedBy, st, snapshot)
	if err != nil {
		return nil, err
	}

	if s.publish != nil && s.publish.Enabled() {
		if err := s.publish.EnqueueKnowledgeJobRun(ctx, runID, jobID); err != nil {
			_, _ = s.pool.Exec(ctx, `UPDATE job_runs SET status='failed', completed_at=now(), error_count=error_count+1 WHERE id=$1`, runID)
			return nil, err
		}
		return s.getRun(ctx, runID)
	}

	if err := s.executeRun(ctx, runID, j, initiatedBy); err != nil {
		_, _ = s.pool.Exec(ctx, `UPDATE job_runs SET status='failed', completed_at=now(), error_count=error_count+1 WHERE id=$1`, runID)
		return s.getRun(ctx, runID)
	}
	_, _ = s.pool.Exec(ctx, `UPDATE job_runs SET status='completed', completed_at=now() WHERE id=$1`, runID)
	return s.getRun(ctx, runID)
}

func (s *JobService) executeRun(ctx context.Context, runID uuid.UUID, j *KnowledgeJob, operator uuid.UUID) error {
	if s.orch == nil {
		return nil
	}
	return s.orch.execute(ctx, runID, j, operator)
}

// ProcessQueuedRun executes a run that was inserted with status queued (worker entrypoint).
func (s *JobService) ProcessQueuedRun(ctx context.Context, runID uuid.UUID) error {
	var jobID uuid.UUID
	var initiatedBy *uuid.UUID
	var st string
	err := s.pool.QueryRow(ctx, `
		SELECT knowledge_job_id, initiated_by_id, status FROM job_runs WHERE id=$1`, runID).Scan(&jobID, &initiatedBy, &st)
	if err != nil {
		return err
	}
	if st != "queued" {
		return nil
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE job_runs SET status='running', started_at=now() WHERE id=$1 AND status='queued'`, runID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return nil
	}
	j, err := s.Get(ctx, jobID)
	if err != nil {
		_, _ = s.pool.Exec(ctx, `UPDATE job_runs SET status='failed', completed_at=now(), error_count=error_count+1 WHERE id=$1`, runID)
		return err
	}
	op := uuid.Nil
	if initiatedBy != nil {
		op = *initiatedBy
	}
	if err := s.executeRun(ctx, runID, j, op); err != nil {
		_, _ = s.pool.Exec(ctx, `UPDATE job_runs SET status='failed', completed_at=now(), error_count=error_count+1 WHERE id=$1`, runID)
		return err
	}
	_, _ = s.pool.Exec(ctx, `UPDATE job_runs SET status='completed', completed_at=now() WHERE id=$1`, runID)
	return nil
}

func (s *JobService) getRun(ctx context.Context, id uuid.UUID) (*JobRun, error) {
	var r JobRun
	err := s.pool.QueryRow(ctx, `
		SELECT id, knowledge_job_id, status, input_scope_snapshot_json FROM job_runs WHERE id=$1`, id,
	).Scan(&r.ID, &r.KnowledgeJobID, &r.Status, &r.InputScopeSnapshotJSON)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// OperatorCanRun allows the job owner or an explicit knowledge_job_operators row (user principal).
func (s *JobService) OperatorCanRun(ctx context.Context, job *KnowledgeJob, userID uuid.UUID) bool {
	if job.OwnerID == userID {
		return true
	}
	var n int
	_ = s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM knowledge_job_operators
		WHERE knowledge_job_id = $1 AND principal_type = 'user' AND principal_id = $2`,
		job.ID, userID).Scan(&n)
	return n > 0
}

// JobTriggerRow is a persisted job trigger.
type JobTriggerRow struct {
	ID               uuid.UUID       `json:"id"`
	KnowledgeJobID   uuid.UUID       `json:"knowledge_job_id"`
	TriggerType      string          `json:"trigger_type"`
	ScheduleExpr     *string         `json:"schedule_expr,omitempty"`
	EventFilterJSON  json.RawMessage `json:"event_filter_json"`
	WindowConfigJSON json.RawMessage `json:"window_config_json"`
	Status           string          `json:"status"`
}

// JobOutputRow is output from a job run.
type JobOutputRow struct {
	ID                    uuid.UUID       `json:"id"`
	JobRunID              uuid.UUID       `json:"job_run_id"`
	OutputType            string          `json:"output_type"`
	StructuredPayloadJSON json.RawMessage `json:"structured_payload_json"`
	TargetEntityID        *uuid.UUID      `json:"target_entity_id,omitempty"`
	PublicationStatus     string          `json:"publication_status"`
}

func (s *JobService) ListTriggers(ctx context.Context, jobID uuid.UUID) ([]JobTriggerRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, knowledge_job_id, trigger_type, schedule_expr, event_filter_json, window_config_json, status
		FROM job_triggers WHERE knowledge_job_id=$1 ORDER BY created_at ASC`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []JobTriggerRow
	for rows.Next() {
		var t JobTriggerRow
		if err := rows.Scan(&t.ID, &t.KnowledgeJobID, &t.TriggerType, &t.ScheduleExpr, &t.EventFilterJSON, &t.WindowConfigJSON, &t.Status); err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	return list, rows.Err()
}

func (s *JobService) GetTrigger(ctx context.Context, id uuid.UUID) (*JobTriggerRow, error) {
	var t JobTriggerRow
	err := s.pool.QueryRow(ctx, `
		SELECT id, knowledge_job_id, trigger_type, schedule_expr, event_filter_json, window_config_json, status
		FROM job_triggers WHERE id=$1`, id,
	).Scan(&t.ID, &t.KnowledgeJobID, &t.TriggerType, &t.ScheduleExpr, &t.EventFilterJSON, &t.WindowConfigJSON, &t.Status)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

type CreateTriggerInput struct {
	TriggerType      string
	ScheduleExpr     *string
	EventFilterJSON  json.RawMessage
	WindowConfigJSON json.RawMessage
}

func (s *JobService) CreateTrigger(ctx context.Context, jobID uuid.UUID, in CreateTriggerInput) (*JobTriggerRow, error) {
	if err := ValidateTriggerInput(in); err != nil {
		return nil, err
	}
	if len(in.EventFilterJSON) == 0 {
		in.EventFilterJSON = []byte("{}")
	}
	if len(in.WindowConfigJSON) == 0 {
		in.WindowConfigJSON = []byte("{}")
	}
	id := uuid.New()
	_, err := s.pool.Exec(ctx, `
		INSERT INTO job_triggers (id, knowledge_job_id, trigger_type, schedule_expr, event_filter_json, window_config_json, status)
		VALUES ($1,$2,$3,$4,$5,$6,'active')`, id, jobID, in.TriggerType, in.ScheduleExpr, in.EventFilterJSON, in.WindowConfigJSON)
	if err != nil {
		return nil, err
	}
	return s.GetTrigger(ctx, id)
}

func (s *JobService) PatchTrigger(ctx context.Context, id uuid.UUID, status *string, scheduleExpr *string) (*JobTriggerRow, error) {
	if status != nil {
		_, err := s.pool.Exec(ctx, `UPDATE job_triggers SET status=$2, updated_at=now() WHERE id=$1`, id, *status)
		if err != nil {
			return nil, err
		}
	}
	if scheduleExpr != nil {
		_, err := s.pool.Exec(ctx, `UPDATE job_triggers SET schedule_expr=$2, updated_at=now() WHERE id=$1`, id, *scheduleExpr)
		if err != nil {
			return nil, err
		}
	}
	return s.GetTrigger(ctx, id)
}

func (s *JobService) DeleteTrigger(ctx context.Context, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM job_triggers WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *JobService) ListRunsForJob(ctx context.Context, jobID uuid.UUID, limit int) ([]JobRun, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, knowledge_job_id, status, input_scope_snapshot_json
		FROM job_runs WHERE knowledge_job_id=$1 ORDER BY started_at DESC LIMIT $2`, jobID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []JobRun
	for rows.Next() {
		var r JobRun
		if err := rows.Scan(&r.ID, &r.KnowledgeJobID, &r.Status, &r.InputScopeSnapshotJSON); err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

func (s *JobService) ListOutputsForRun(ctx context.Context, runID uuid.UUID) ([]JobOutputRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, job_run_id, output_type, structured_payload_json, target_entity_id, publication_status
		FROM job_outputs WHERE job_run_id=$1 ORDER BY created_at ASC`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []JobOutputRow
	for rows.Next() {
		var o JobOutputRow
		if err := rows.Scan(&o.ID, &o.JobRunID, &o.OutputType, &o.StructuredPayloadJSON, &o.TargetEntityID, &o.PublicationStatus); err != nil {
			return nil, err
		}
		list = append(list, o)
	}
	return list, rows.Err()
}

func (s *JobService) GetJobOutput(ctx context.Context, id uuid.UUID) (*JobOutputRow, error) {
	var o JobOutputRow
	err := s.pool.QueryRow(ctx, `
		SELECT id, job_run_id, output_type, structured_payload_json, target_entity_id, publication_status
		FROM job_outputs WHERE id=$1`, id,
	).Scan(&o.ID, &o.JobRunID, &o.OutputType, &o.StructuredPayloadJSON, &o.TargetEntityID, &o.PublicationStatus)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

// FailedJobRun is a compact row for ops dashboards.
type FailedJobRun struct {
	ID             uuid.UUID `json:"id"`
	KnowledgeJobID uuid.UUID `json:"knowledge_job_id"`
	Status         string    `json:"status"`
	StartedAt      time.Time `json:"started_at"`
}

// ListRecentFailedJobRuns returns latest failed job runs.
func (s *JobService) ListRecentFailedJobRuns(ctx context.Context, limit int) ([]FailedJobRun, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, knowledge_job_id, status, started_at FROM job_runs
		WHERE status = 'failed' ORDER BY started_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []FailedJobRun
	for rows.Next() {
		var r FailedJobRun
		if err := rows.Scan(&r.ID, &r.KnowledgeJobID, &r.Status, &r.StartedAt); err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

// JobRunListing is a denormalized projection used by the CP "all runs" view.
// It joins job_runs with knowledge_jobs so the UI can display name + job_type
// without an N+1 fetch.
type JobRunListing struct {
	ID             uuid.UUID  `json:"id"`
	KnowledgeJobID uuid.UUID  `json:"knowledge_job_id"`
	JobName        string     `json:"job_name"`
	JobType        string     `json:"job_type"`
	Status         string     `json:"status"`
	StartedAt      time.Time  `json:"started_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
}

// ListRecentRuns returns the most recent runs across all jobs, optionally
// filtered by status and/or job_type. Used by the CP /control-plane/jobs/runs
// listing (Phase 2.1.3). Operator authorization is enforced at the route
// layer (identity admin or principal that may run jobs in the run's domain).
func (s *JobService) ListRecentRuns(ctx context.Context, limit int, statusFilter, jobTypeFilter string) ([]JobRunListing, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	q := `
		SELECT r.id, r.knowledge_job_id, j.name, j.job_type, r.status, r.started_at, r.completed_at
		FROM job_runs r
		JOIN knowledge_jobs j ON j.id = r.knowledge_job_id
		WHERE 1=1`
	args := []any{}
	n := 1
	if statusFilter != "" {
		q += ` AND r.status = $` + strconv.Itoa(n)
		args = append(args, statusFilter)
		n++
	}
	if jobTypeFilter != "" {
		q += ` AND j.job_type = $` + strconv.Itoa(n)
		args = append(args, jobTypeFilter)
		n++
	}
	q += ` ORDER BY r.started_at DESC LIMIT $` + strconv.Itoa(n)
	args = append(args, limit)

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []JobRunListing
	for rows.Next() {
		var r JobRunListing
		if err := rows.Scan(&r.ID, &r.KnowledgeJobID, &r.JobName, &r.JobType, &r.Status, &r.StartedAt, &r.CompletedAt); err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, rows.Err()
}
