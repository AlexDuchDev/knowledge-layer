package role_builder

import (
	"context"

	"github.com/google/uuid"
)

// PreviewService builds effective-access previews for roles (and optionally users later).
type PreviewService struct {
	repo *DefinitionRepository
}

func NewPreviewService(repo *DefinitionRepository) *PreviewService {
	return &PreviewService{repo: repo}
}

func (s *PreviewService) PreviewRole(ctx context.Context, roleID uuid.UUID) (*EffectiveAccessPreview, error) {
	f, err := s.repo.GetFull(ctx, roleID)
	if err != nil {
		return nil, err
	}
	return &EffectiveAccessPreview{
		RoleID:       f.ID,
		Code:         f.Code,
		Name:         f.Name,
		Category:     f.Category,
		ScopeModel:   f.ScopeModel,
		Active:       f.Active,
		Domains:      append([]uuid.UUID(nil), f.DomainIDs...),
		EntityTypes:  append([]string(nil), f.EntityTypes...),
		SourceScopes: append([]SourceScopeRef(nil), f.SourceScopes...),
		Scenarios:    append([]string(nil), f.ScenarioKeys...),
		Dashboards:   append([]string(nil), f.DashboardKeys...),
		Actions:      append([]string(nil), f.ActionCodes...),
		Governance:   f.Governance,
		Jobs:         append([]JobPermissionRow(nil), f.JobPermissions...),
	}, nil
}
