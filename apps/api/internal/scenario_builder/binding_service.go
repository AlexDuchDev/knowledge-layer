package scenario_builder

import (
	"context"

	"github.com/google/uuid"
)

// BindingService manages scenario bindings and role mirror sync.
type BindingService struct {
	defRepo  *DefinitionRepository
	bindRepo *BindingsRepository
}

func NewBindingService(defRepo *DefinitionRepository, bindRepo *BindingsRepository) *BindingService {
	return &BindingService{defRepo: defRepo, bindRepo: bindRepo}
}

func (s *BindingService) ReplaceRoleBindings(ctx context.Context, scenarioID uuid.UUID, rows []RoleBindingWrite) error {
	code, err := s.defRepo.GetCode(ctx, scenarioID)
	if err != nil {
		return err
	}
	return s.bindRepo.ReplaceRoleBindings(ctx, scenarioID, code, rows)
}

func (s *BindingService) ReplaceSourceBindings(ctx context.Context, scenarioID uuid.UUID, rows []SourceBindingRow) error {
	return s.bindRepo.ReplaceSourceBindings(ctx, scenarioID, rows)
}

func (s *BindingService) ReplaceJobBindings(ctx context.Context, scenarioID uuid.UUID, rows []JobBindingRow) error {
	return s.bindRepo.ReplaceJobBindings(ctx, scenarioID, rows)
}

func (s *BindingService) ReplaceUIBindings(ctx context.Context, scenarioID uuid.UUID, rows []UIBindingRow) error {
	return s.bindRepo.ReplaceUIBindings(ctx, scenarioID, rows)
}
