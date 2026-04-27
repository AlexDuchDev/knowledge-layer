package scenario_builder

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// DefinitionService orchestrates scenario definition lifecycle.
type DefinitionService struct {
	repo     *DefinitionRepository
	bindRepo *BindingsRepository
}

func NewDefinitionService(repo *DefinitionRepository, bindRepo *BindingsRepository) *DefinitionService {
	return &DefinitionService{repo: repo, bindRepo: bindRepo}
}

func (s *DefinitionService) List(ctx context.Context) ([]ScenarioSummary, error) {
	return s.repo.ListSummaries(ctx)
}

func (s *DefinitionService) Get(ctx context.Context, id uuid.UUID) (*ScenarioFull, error) {
	return s.repo.GetFull(ctx, id)
}

func (s *DefinitionService) Create(ctx context.Context, in ScenarioWriteInput) (uuid.UUID, error) {
	return s.repo.Create(ctx, in)
}

// Patch updates definition fields; if code changes, re-keys role_scenario_bindings mirror.
func (s *DefinitionService) Patch(ctx context.Context, id uuid.UUID,
	name, code *string,
	description *string,
	scenarioType *string,
	active *bool,
	targetScope, inputScope, triggerCfg *json.RawMessage,
	triggerType *string,
	procMode, outMode, uiSurface *string,
	configJSON, previewConfig *json.RawMessage,
	notes *string,
	ownerUser, ownerTeam *uuid.UUID,
	policy *OutputPolicyWrite,
) error {
	oldCode, err := s.repo.GetCode(ctx, id)
	if err != nil {
		return err
	}
	if code != nil && strings.TrimSpace(*code) != "" && strings.TrimSpace(*code) != oldCode {
		if err := s.bindRepo.RemoveRoleScenarioMirrorForCode(ctx, oldCode); err != nil {
			return err
		}
	}
	if err := s.repo.Patch(ctx, id, name, code, description, scenarioType, active,
		targetScope, inputScope, triggerCfg, triggerType, procMode, outMode, uiSurface,
		configJSON, previewConfig, notes, ownerUser, ownerTeam, policy); err != nil {
		return err
	}
	newCode, err := s.repo.GetCode(ctx, id)
	if err != nil {
		return err
	}
	if code != nil && strings.TrimSpace(*code) != oldCode {
		return s.bindRepo.ResyncRoleScenarioMirror(ctx, id, newCode)
	}
	return nil
}

func (s *DefinitionService) Delete(ctx context.Context, id uuid.UUID) error {
	isPreset, err := s.repo.IsPreset(ctx, id)
	if err != nil {
		return err
	}
	if isPreset {
		return fmt.Errorf("cannot delete system preset scenario")
	}
	code, err := s.repo.GetCode(ctx, id)
	if err != nil {
		return err
	}
	if err := s.bindRepo.RemoveRoleScenarioMirrorForCode(ctx, code); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}
