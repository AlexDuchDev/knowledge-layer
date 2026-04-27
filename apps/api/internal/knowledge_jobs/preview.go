package knowledge_jobs

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

// JobPreview is the structured effective configuration for admins (Job Builder preview).
type JobPreview struct {
	Identity struct {
		ID          uuid.UUID `json:"id"`
		Name        string    `json:"name"`
		Description *string   `json:"description,omitempty"`
		Purpose     *string   `json:"purpose,omitempty"`
		OwnerID     uuid.UUID `json:"owner_id"`
		Status      string    `json:"status"`
	} `json:"identity"`
	TemplateKey        *string         `json:"template_key,omitempty"`
	JobType            string          `json:"job_type"`
	ProcessingMode     string          `json:"processing_mode"`
	PrimaryTriggerType string          `json:"primary_trigger_type"`
	Triggers           []JobTriggerRow `json:"triggers"`
	SourceScopeJSON    json.RawMessage `json:"source_scope_json"`
	SourceBindings     []struct {
		SourceType string    `json:"source_type"`
		SourceID   uuid.UUID `json:"source_id"`
	} `json:"source_bindings"`
	OutputPolicy struct {
		OutputDomainID     *uuid.UUID `json:"output_domain_id,omitempty"`
		OutputSensitivity  int        `json:"output_sensitivity_level"`
		PublicationMode    string     `json:"publication_mode"`
		ReviewRequired     bool       `json:"review_required"`
		CitationsRequired  bool       `json:"citations_required"`
		ProvenanceRequired bool       `json:"provenance_required"`
		OutputType         string     `json:"output_type"`
	} `json:"output_policy"`
	Governance struct {
		ScenarioOnlyExposure bool `json:"scenario_only_exposure"`
		AllowDomainRunJob    bool `json:"allow_domain_run_job"`
	} `json:"governance"`
	Operators []struct {
		PrincipalType string    `json:"principal_type"`
		PrincipalID   uuid.UUID `json:"principal_id"`
	} `json:"operators"`
	ScenarioBindings []struct {
		ScenarioID   uuid.UUID `json:"scenario_id"`
		ScenarioCode string    `json:"scenario_code"`
		Relationship string    `json:"relationship"`
	} `json:"scenario_bindings"`
	ConfigJSON          json.RawMessage `json:"config_json"`
	OperatorScopeJSON   json.RawMessage `json:"operator_scope_json"`
	ValidationErrors    []string        `json:"validation_errors,omitempty"`
	ValidationWarnings  []string        `json:"validation_warnings,omitempty"`
	RolePermissionsNote string          `json:"role_permissions_note"`
}

// BuildJobPreview assembles effective job config and runs definition validation checks.
func (s *JobService) BuildJobPreview(ctx context.Context, jobID uuid.UUID) (*JobPreview, error) {
	j, err := s.Get(ctx, jobID)
	if err != nil {
		return nil, err
	}
	triggers, err := s.ListTriggers(ctx, jobID)
	if err != nil {
		return nil, err
	}
	opRows, err := s.listOperatorRows(ctx, jobID)
	if err != nil {
		return nil, err
	}
	scenRows, err := s.listScenarioBindingRows(ctx, jobID)
	if err != nil {
		return nil, err
	}
	srcRows, err := s.listSourceBindingRows(ctx, jobID)
	if err != nil {
		return nil, err
	}
	var outputType string
	_ = s.pool.QueryRow(ctx, `SELECT output_type FROM knowledge_jobs WHERE id=$1`, jobID).Scan(&outputType)

	in := jobToCreateInput(j)
	var errs []string
	var warns []string
	if e := ValidateUpdateJobInput(in); e != nil {
		errs = append(errs, e.Error())
	}
	if e := ValidateTriggerRowsForPrimaryType(j.TriggerType, triggers); e != nil {
		errs = append(errs, e.Error())
	}
	if j.ScenarioOnlyExposure && len(scenRows) == 0 {
		warns = append(warns, "scenario_only_exposure is true but no scenario bindings exist")
	}

	p := &JobPreview{}
	p.Identity.ID = j.ID
	p.Identity.Name = j.Name
	p.Identity.Description = j.Description
	p.Identity.Purpose = j.Purpose
	p.Identity.OwnerID = j.OwnerID
	p.Identity.Status = j.Status
	if j.TemplateKey != nil && *j.TemplateKey != "" {
		p.TemplateKey = j.TemplateKey
	}
	p.JobType = j.JobType
	p.ProcessingMode = j.ProcessingMode
	p.PrimaryTriggerType = j.TriggerType
	p.Triggers = triggers
	p.SourceScopeJSON = j.SourceScopeJSON
	for _, r := range srcRows {
		p.SourceBindings = append(p.SourceBindings, struct {
			SourceType string    `json:"source_type"`
			SourceID   uuid.UUID `json:"source_id"`
		}{SourceType: r.SourceType, SourceID: r.SourceID})
	}
	p.OutputPolicy.OutputDomainID = j.OutputDomainID
	p.OutputPolicy.OutputSensitivity = j.OutputSensitivity
	p.OutputPolicy.PublicationMode = j.PublicationMode
	p.OutputPolicy.ReviewRequired = j.ReviewRequired
	p.OutputPolicy.CitationsRequired = j.CitationsRequired
	p.OutputPolicy.ProvenanceRequired = j.ProvenanceRequired
	p.OutputPolicy.OutputType = outputType
	p.Governance.ScenarioOnlyExposure = j.ScenarioOnlyExposure
	p.Governance.AllowDomainRunJob = j.AllowDomainRunJob
	for _, o := range opRows {
		p.Operators = append(p.Operators, struct {
			PrincipalType string    `json:"principal_type"`
			PrincipalID   uuid.UUID `json:"principal_id"`
		}{PrincipalType: o.PrincipalType, PrincipalID: o.PrincipalID})
	}
	for _, sb := range scenRows {
		p.ScenarioBindings = append(p.ScenarioBindings, struct {
			ScenarioID   uuid.UUID `json:"scenario_id"`
			ScenarioCode string    `json:"scenario_code"`
			Relationship string    `json:"relationship"`
		}{ScenarioID: sb.ScenarioID, ScenarioCode: sb.ScenarioCode, Relationship: sb.Relationship})
	}
	p.ConfigJSON = j.ConfigJSON
	p.OperatorScopeJSON = j.OperatorScopeJSON
	p.ValidationErrors = errs
	p.ValidationWarnings = warns
	p.RolePermissionsNote = "Role-based job permissions (role_job_permissions) may further restrict run/configure/review; not expanded in this preview."
	return p, nil
}

func jobToCreateInput(j *KnowledgeJob) CreateJobInput {
	tk := ""
	if j.TemplateKey != nil {
		tk = *j.TemplateKey
	}
	pr := j.ProvenanceRequired
	ad := j.AllowDomainRunJob
	return CreateJobInput{
		TemplateID:           tk,
		Name:                 j.Name,
		JobType:              j.JobType,
		Purpose:              j.Purpose,
		Description:          j.Description,
		OwnerID:              j.OwnerID,
		SourceScopeJSON:      j.SourceScopeJSON,
		OperatorScopeJSON:    j.OperatorScopeJSON,
		TriggerType:          j.TriggerType,
		OutputDomainID:       j.OutputDomainID,
		OutputSensitivity:    j.OutputSensitivity,
		PublicationMode:      j.PublicationMode,
		ReviewRequired:       j.ReviewRequired,
		ConfigJSON:           j.ConfigJSON,
		ProcessingMode:       j.ProcessingMode,
		CitationsRequired:    j.CitationsRequired,
		ProvenanceRequired:   &pr,
		ScenarioOnlyExposure: j.ScenarioOnlyExposure,
		AllowDomainRunJob:    &ad,
	}
}

type sourceBindingRow struct {
	SourceType string
	SourceID   uuid.UUID
}

func (s *JobService) listSourceBindingRows(ctx context.Context, jobID uuid.UUID) ([]sourceBindingRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT source_type, source_id FROM knowledge_job_sources WHERE knowledge_job_id=$1 ORDER BY source_type, source_id`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []sourceBindingRow
	for rows.Next() {
		var r sourceBindingRow
		if err := rows.Scan(&r.SourceType, &r.SourceID); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

type scenarioBindingRow struct {
	ScenarioID   uuid.UUID
	ScenarioCode string
	Relationship string
}

func (s *JobService) listScenarioBindingRows(ctx context.Context, jobID uuid.UUID) ([]scenarioBindingRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT sjb.scenario_id, sd.code, sjb.relationship
		FROM scenario_job_bindings sjb
		JOIN scenario_definitions sd ON sd.id = sjb.scenario_id
		WHERE sjb.knowledge_job_id = $1
		ORDER BY sd.code`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []scenarioBindingRow
	for rows.Next() {
		var r scenarioBindingRow
		if err := rows.Scan(&r.ScenarioID, &r.ScenarioCode, &r.Relationship); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

type operatorRow struct {
	PrincipalType string
	PrincipalID   uuid.UUID
}

func (s *JobService) listOperatorRows(ctx context.Context, jobID uuid.UUID) ([]operatorRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT principal_type, principal_id FROM knowledge_job_operators
		WHERE knowledge_job_id=$1 ORDER BY principal_type, principal_id`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []operatorRow
	for rows.Next() {
		var r operatorRow
		if err := rows.Scan(&r.PrincipalType, &r.PrincipalID); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
