package role_builder

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// PresetService lists catalog presets and materializes editable roles from them.
type PresetService struct {
	repo *DefinitionRepository
	def  *DefinitionService
}

func NewPresetService(repo *DefinitionRepository, def *DefinitionService) *PresetService {
	return &PresetService{repo: repo, def: def}
}

func (s *PresetService) List(ctx context.Context) ([]RoleSummary, error) {
	return s.repo.ListPresets(ctx)
}

// CreateFromPreset clones a preset into a new editable role.
func (s *PresetService) CreateFromPreset(ctx context.Context, presetKey, newCode, newName string, description *string) (uuid.UUID, error) {
	presetKey = strings.TrimSpace(presetKey)
	if presetKey == "" {
		return uuid.Nil, fmt.Errorf("preset_key is required")
	}
	sum, err := s.repo.GetByPresetKey(ctx, presetKey)
	if err != nil {
		return uuid.Nil, fmt.Errorf("preset not found: %w", err)
	}
	pk := presetKey
	return s.def.Clone(ctx, sum.ID, newCode, newName, description, sum.Category, &pk)
}
