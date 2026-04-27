package knowledge_jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// CreateFromBuilderPreset creates a knowledge job from job_builder_presets + template catalog.
func (s *JobService) CreateFromBuilderPreset(ctx context.Context, presetKey string, ownerID uuid.UUID, nameOverride, purposeOverride *string) (*KnowledgeJob, error) {
	presetKey = strings.TrimSpace(presetKey)
	if presetKey == "" {
		return nil, fmt.Errorf("preset_key required")
	}
	var dbName string
	var dbDesc *string
	var templateKey string
	var defaultsJSON json.RawMessage
	err := s.pool.QueryRow(ctx, `
		SELECT name, description, template_key, defaults_json
		FROM job_builder_presets WHERE preset_key = $1`, presetKey,
	).Scan(&dbName, &dbDesc, &templateKey, &defaultsJSON)
	if err != nil {
		return nil, fmt.Errorf("job builder preset: %w", err)
	}

	var defs struct {
		ProcessingMode       string          `json:"processing_mode"`
		ConfigJSON           json.RawMessage `json:"config_json"`
		CitationsRequired    *bool           `json:"citations_required"`
		ReviewRequired       *bool           `json:"review_required"`
		PublicationMode      string          `json:"publication_mode"`
		OperatorScopeJSON    json.RawMessage `json:"operator_scope_json"`
		SourceScopeJSON      json.RawMessage `json:"source_scope_json"`
		TriggerType          string          `json:"trigger_type"`
		OutputSensitivity    *int            `json:"output_sensitivity_level"`
		ProvenanceRequired   *bool           `json:"provenance_required"`
		ScenarioOnlyExposure *bool           `json:"scenario_only_exposure"`
		AllowDomainRunJob    *bool           `json:"allow_domain_run_job"`
	}
	if len(defaultsJSON) > 0 && string(defaultsJSON) != "null" {
		_ = json.Unmarshal(defaultsJSON, &defs)
	}

	in := CreateJobInput{
		TemplateID:        templateKey,
		OwnerID:           ownerID,
		SourceScopeJSON:   defs.SourceScopeJSON,
		OperatorScopeJSON: defs.OperatorScopeJSON,
		TriggerType:       defs.TriggerType,
		PublicationMode:   defs.PublicationMode,
		ProcessingMode:    defs.ProcessingMode,
		ConfigJSON:        defs.ConfigJSON,
	}
	if nameOverride != nil && strings.TrimSpace(*nameOverride) != "" {
		in.Name = strings.TrimSpace(*nameOverride)
	} else {
		in.Name = dbName
	}
	if purposeOverride != nil {
		in.Purpose = purposeOverride
	} else if dbDesc != nil {
		p := *dbDesc
		in.Purpose = &p
	}
	if defs.CitationsRequired != nil {
		in.CitationsRequired = *defs.CitationsRequired
	}
	if defs.ReviewRequired != nil {
		in.ReviewRequired = *defs.ReviewRequired
	}
	if defs.OutputSensitivity != nil {
		in.OutputSensitivity = *defs.OutputSensitivity
	}
	if defs.ProvenanceRequired != nil {
		in.ProvenanceRequired = defs.ProvenanceRequired
	}
	if defs.ScenarioOnlyExposure != nil {
		in.ScenarioOnlyExposure = *defs.ScenarioOnlyExposure
	}
	if defs.AllowDomainRunJob != nil {
		in.AllowDomainRunJob = defs.AllowDomainRunJob
	}
	pk := presetKey
	in.SourcePresetCode = &pk

	return s.Create(ctx, in)
}
