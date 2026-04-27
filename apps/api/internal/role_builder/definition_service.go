package role_builder

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

// DefinitionService orchestrates role definition lifecycle.
type DefinitionService struct {
	repo *DefinitionRepository
}

func NewDefinitionService(repo *DefinitionRepository) *DefinitionService {
	return &DefinitionService{repo: repo}
}

func (s *DefinitionService) List(ctx context.Context) ([]RoleSummary, error) {
	return s.repo.ListSummaries(ctx)
}

func (s *DefinitionService) Get(ctx context.Context, id uuid.UUID) (*RoleFull, error) {
	return s.repo.GetFull(ctx, id)
}

func (s *DefinitionService) Create(ctx context.Context, in RoleWriteInput) (uuid.UUID, error) {
	if err := ValidateWriteInput(in, true); err != nil {
		return uuid.Nil, err
	}
	id, err := s.repo.Create(ctx, in)
	if err != nil {
		if isUniqueViolation(err) {
			return uuid.Nil, fmt.Errorf("duplicate role code or constraint violation: %w", err)
		}
		return uuid.Nil, err
	}
	return id, nil
}

// Patch updates metadata; if bindings is non-nil, replaces all bindings (full replace).
func (s *DefinitionService) Patch(ctx context.Context, id uuid.UUID, name, code *string, description *string, category *string, active *bool, scopeModel *string, bindings *RoleWriteInput) error {
	full, err := s.repo.GetFull(ctx, id)
	if err != nil {
		return err
	}
	if full.IsSystem && active != nil && !*active {
		return fmt.Errorf("cannot deactivate system role")
	}

	if bindings != nil {
		bindings.Code = full.Code
		if code != nil {
			bindings.Code = *code
		}
		bindings.Name = full.Name
		if name != nil {
			bindings.Name = *name
		}
		if bindings.Category == "" {
			bindings.Category = full.Category
		}
		if category != nil {
			bindings.Category = *category
		}
		if bindings.ScopeModel == "" {
			bindings.ScopeModel = full.ScopeModel
		}
		if scopeModel != nil {
			bindings.ScopeModel = *scopeModel
		}
		if bindings.Governance == nil {
			bindings.Governance = &full.Governance
		}
		if err := ValidateReplaceBindings(*bindings); err != nil {
			return err
		}
	}

	return s.repo.Patch(ctx, id, name, code, description, category, active, scopeModel, bindings)
}

func (s *DefinitionService) Delete(ctx context.Context, id uuid.UUID) error {
	full, err := s.repo.GetFull(ctx, id)
	if err != nil {
		return err
	}
	if full.IsSystem {
		return fmt.Errorf("cannot delete system role")
	}
	n, err := s.repo.CountAssignments(ctx, id)
	if err != nil {
		return err
	}
	if n > 0 {
		return s.repo.SetInactive(ctx, id)
	}
	return s.repo.DeleteHard(ctx, id)
}

func (s *DefinitionService) Clone(ctx context.Context, sourceID uuid.UUID, newCode, newName string, description *string, category string, sourcePresetCode *string) (uuid.UUID, error) {
	if strings.TrimSpace(newCode) == "" || strings.TrimSpace(newName) == "" {
		return uuid.Nil, fmt.Errorf("newCode and newName are required")
	}
	if category == "" {
		category = "domain"
	}
	id, err := s.repo.Clone(ctx, sourceID, newCode, newName, description, category, sourcePresetCode)
	if err != nil {
		if isUniqueViolation(err) {
			return uuid.Nil, fmt.Errorf("duplicate role code: %w", err)
		}
		return uuid.Nil, err
	}
	return id, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
