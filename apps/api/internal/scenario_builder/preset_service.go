package scenario_builder

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// PresetService lists catalog presets and creates scenarios from templates.
type PresetService struct {
	repo *DefinitionRepository
	def  *DefinitionService
}

func NewPresetService(repo *DefinitionRepository, def *DefinitionService) *PresetService {
	return &PresetService{repo: repo, def: def}
}

func (s *PresetService) List(ctx context.Context) ([]ScenarioPresetCatalogRow, error) {
	return s.repo.ListPresetsCatalog(ctx)
}

// CreateFromPreset builds a ScenarioWriteInput from scenario_presets.template_json and calls Create.
func (s *PresetService) CreateFromPreset(ctx context.Context, in FromPresetInput) (uuid.UUID, error) {
	key := strings.TrimSpace(in.PresetKey)
	if key == "" {
		return uuid.Nil, fmt.Errorf("preset_key required")
	}
	row, err := s.repo.GetPresetCatalogRow(ctx, key)
	if err != nil {
		return uuid.Nil, fmt.Errorf("preset not found: %w", err)
	}
	if strings.TrimSpace(in.Code) == "" {
		return uuid.Nil, fmt.Errorf("code required")
	}
	if strings.TrimSpace(in.Name) == "" {
		return uuid.Nil, fmt.Errorf("name required")
	}

	var tpl map[string]interface{}
	if err := json.Unmarshal(row.TemplateJSON, &tpl); err != nil {
		return uuid.Nil, fmt.Errorf("preset template: %w", err)
	}
	for k, v := range in.Overrides {
		tpl[k] = v
	}

	write := ScenarioWriteInput{
		Code:                strings.TrimSpace(in.Code),
		Name:                strings.TrimSpace(in.Name),
		ScenarioType:        row.ScenarioType,
		TargetRoleScopeJSON: emptyJSON(),
		InputScopeJSON:      emptyJSON(),
		TriggerConfigJSON:   emptyJSON(),
		ConfigJSON:          emptyJSON(),
		PreviewConfig:       emptyJSON(),
		UISurface:           "admin_only",
	}
	active := true
	write.Active = &active

	if v, ok := tpl["description"].(string); ok {
		write.Description = &v
	}
	if v, ok := tpl["scenario_type"].(string); ok && v != "" {
		write.ScenarioType = v
	}
	if v, ok := tpl["trigger_type"].(string); ok {
		write.TriggerType = v
	}
	if v, ok := tpl["processing_mode"].(string); ok {
		write.ProcessingMode = v
	}
	if v, ok := tpl["output_mode"].(string); ok {
		write.OutputMode = v
	}
	if v, ok := tpl["ui_surface"].(string); ok {
		write.UISurface = v
	}
	marshalInto := func(key string, dest *json.RawMessage) {
		if raw, ok := tpl[key]; ok && raw != nil {
			b, err := json.Marshal(raw)
			if err == nil {
				*dest = b
			}
		}
	}
	marshalInto("target_role_scope_json", &write.TargetRoleScopeJSON)
	marshalInto("input_scope_json", &write.InputScopeJSON)
	marshalInto("trigger_config_json", &write.TriggerConfigJSON)
	marshalInto("config_json", &write.ConfigJSON)
	marshalInto("preview_config", &write.PreviewConfig)

	tplID, err := s.repo.GetSystemScenarioIDByPresetKey(ctx, key)
	if err == nil && tplID != uuid.Nil {
		write.ClonedFromScenarioID = &tplID
	}
	sk := key
	write.SourcePresetCode = &sk
	write.PresetKey = nil

	if op, ok := tpl["output_policy"].(map[string]interface{}); ok {
		pw := &OutputPolicyWrite{
			PublicationMode:    "draft",
			ExtraJSON:          emptyJSON(),
			OutputSensitivity:  0,
			ReviewRequired:     false,
			CitationsRequired:  false,
			ProvenanceRequired: false,
		}
		if v, ok := op["publication_mode"].(string); ok {
			pw.PublicationMode = v
		}
		if v, ok := op["output_sensitivity_level"].(float64); ok {
			pw.OutputSensitivity = int(v)
		}
		if v, ok := op["review_required"].(bool); ok {
			pw.ReviewRequired = v
		}
		if v, ok := op["citations_required"].(bool); ok {
			pw.CitationsRequired = v
		}
		if v, ok := op["provenance_required"].(bool); ok {
			pw.ProvenanceRequired = v
		}
		if ex, ok := op["extra_json"]; ok {
			if b, err := json.Marshal(ex); err == nil {
				pw.ExtraJSON = b
			}
		}
		write.OutputPolicy = pw
	}

	write.IsPreset = false

	id, err := s.def.Create(ctx, write)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}
