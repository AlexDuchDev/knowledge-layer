package scenario_builder

import (
	"context"

	"github.com/google/uuid"
)

// PreviewService builds effective scenario shape for admins.
type PreviewService struct {
	repo *DefinitionRepository
}

func NewPreviewService(repo *DefinitionRepository) *PreviewService {
	return &PreviewService{repo: repo}
}

func (s *PreviewService) Build(ctx context.Context, id uuid.UUID) (*ScenarioPreview, error) {
	full, err := s.repo.GetFull(ctx, id)
	if err != nil {
		return nil, err
	}
	p := &ScenarioPreview{
		ScenarioID:      full.ID,
		Code:            full.Code,
		Name:            full.Name,
		ScenarioType:    full.ScenarioType,
		Active:          full.Active,
		TargetRoleScope: full.TargetRoleScopeJSON,
		InputScope:      full.InputScopeJSON,
		TriggerType:     full.TriggerType,
		TriggerConfig:   full.TriggerConfigJSON,
		ProcessingMode:  full.ProcessingMode,
		OutputMode:      full.OutputMode,
		UISurface:       full.UISurface,
		OutputPolicy:    full.OutputPolicy,
		VisibleRoles:    full.RoleBindings,
		SourceBindings:  full.SourceBindings,
		JobBindings:     full.JobBindings,
		UIBindings:      full.UIBindings,
		ConfigJSON:      full.ConfigJSON,
		PreviewConfig:   full.PreviewConfig,
	}
	if full.OutputPolicy != nil {
		p.GovernanceSummary = GovernanceSummary{
			ReviewRequired:     full.OutputPolicy.ReviewRequired,
			PublicationMode:    full.OutputPolicy.PublicationMode,
			CitationsRequired:  full.OutputPolicy.CitationsRequired,
			ProvenanceRequired: full.OutputPolicy.ProvenanceRequired,
			OutputSensitivity:  full.OutputPolicy.OutputSensitivity,
			HasOutputPolicy:    true,
		}
	}
	return p, nil
}
