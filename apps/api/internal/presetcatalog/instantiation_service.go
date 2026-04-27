package presetcatalog

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/knowledgelayer/api/internal/knowledge_jobs"
	"github.com/knowledgelayer/api/internal/role_builder"
	"github.com/knowledgelayer/api/internal/scenario_builder"
)

// InstantiationService creates builder objects from catalog entries.
type InstantiationService struct {
	repo *CatalogRepository
	log  *LogRepository
	rb   *role_builder.Services
	sb   *scenario_builder.Services
	jobs *knowledge_jobs.JobService
}

func NewInstantiationService(
	repo *CatalogRepository,
	log *LogRepository,
	rb *role_builder.Services,
	sb *scenario_builder.Services,
	jobs *knowledge_jobs.JobService,
) *InstantiationService {
	return &InstantiationService{repo: repo, log: log, rb: rb, sb: sb, jobs: jobs}
}

// Instantiate creates a role, scenario, or job from a catalog entry.
func (s *InstantiationService) Instantiate(ctx context.Context, entryID, principal uuid.UUID, req InstantiateRequest) (*InstantiateResult, error) {
	entry, err := s.repo.GetByID(ctx, entryID)
	if err != nil {
		return nil, err
	}
	if !entry.Active {
		return nil, fmt.Errorf("preset is not active")
	}

	payload, _ := json.Marshal(req)

	switch entry.PresetType {
	case "role":
		newCode := entry.Code + "_instance"
		if req.Code != nil && strings.TrimSpace(*req.Code) != "" {
			newCode = strings.TrimSpace(*req.Code)
		}
		newName := entry.Name + " (instance)"
		if req.Name != nil && strings.TrimSpace(*req.Name) != "" {
			newName = strings.TrimSpace(*req.Name)
		}
		id, err := s.rb.Presets.CreateFromPreset(ctx, entry.Code, newCode, newName, req.Description)
		if err != nil {
			return nil, err
		}
		_ = s.log.Insert(ctx, entryID, principal, "role", id, payload)
		return &InstantiateResult{
			PresetCatalogEntryID: entryID,
			PresetType:           entry.PresetType,
			Code:                 entry.Code,
			TargetKind:           "role",
			TargetID:             id,
			EditPathHint:         fmt.Sprintf("/admin/roles/%s", id.String()),
		}, nil

	case "scenario":
		code := entry.Code + "_instance"
		if req.Code != nil && strings.TrimSpace(*req.Code) != "" {
			code = strings.TrimSpace(*req.Code)
		}
		name := entry.Name + " (instance)"
		if req.Name != nil && strings.TrimSpace(*req.Name) != "" {
			name = strings.TrimSpace(*req.Name)
		}
		overrides := map[string]interface{}{}
		if len(req.Overrides) > 0 && string(req.Overrides) != "null" {
			_ = json.Unmarshal(req.Overrides, &overrides)
		}
		id, err := s.sb.Presets.CreateFromPreset(ctx, scenario_builder.FromPresetInput{
			PresetKey: entry.Code,
			Code:      code,
			Name:      name,
			Overrides: overrides,
		})
		if err != nil {
			return nil, err
		}
		_ = s.log.Insert(ctx, entryID, principal, "scenario", id, payload)
		return &InstantiateResult{
			PresetCatalogEntryID: entryID,
			PresetType:           entry.PresetType,
			Code:                 entry.Code,
			TargetKind:           "scenario",
			TargetID:             id,
			EditPathHint:         fmt.Sprintf("/admin/scenarios/%s", id.String()),
		}, nil

	case "job":
		j, err := s.jobs.CreateFromBuilderPreset(ctx, entry.Code, principal, req.Name, req.Purpose)
		if err != nil {
			return nil, err
		}
		_ = s.log.Insert(ctx, entryID, principal, "job", j.ID, payload)
		return &InstantiateResult{
			PresetCatalogEntryID: entryID,
			PresetType:           entry.PresetType,
			Code:                 entry.Code,
			TargetKind:           "job",
			TargetID:             j.ID,
			EditPathHint:         fmt.Sprintf("/jobs/%s", j.ID.String()),
		}, nil

	default:
		return nil, fmt.Errorf("cannot instantiate preset_type %q", entry.PresetType)
	}
}
