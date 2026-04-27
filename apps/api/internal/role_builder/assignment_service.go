package role_builder

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/knowledgelayer/api/internal/identity_access"
)

// AccessEvaluatorSubset is the access surface needed for assignment gates.
type AccessEvaluatorSubset interface {
	DomainIDsWithGrant(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	Evaluate(ctx context.Context, in identity_access.EvaluateInput) (*identity_access.AccessDecision, error)
}

// AssignmentService validates and creates role assignments.
type AssignmentService struct {
	assign *AssignmentRepository
	def    *DefinitionRepository
	access AccessEvaluatorSubset
}

func NewAssignmentService(assign *AssignmentRepository, def *DefinitionRepository, access AccessEvaluatorSubset) *AssignmentService {
	return &AssignmentService{assign: assign, def: def, access: access}
}

// PrincipalAllowsScenario enforces role-bound scenario keys for Ask and GET /search (fail-closed when a code is supplied).
func (s *AssignmentService) PrincipalAllowsScenario(ctx context.Context, userID uuid.UUID, scenarioKey string) (bool, error) {
	if strings.TrimSpace(scenarioKey) == "" {
		return true, nil
	}
	return s.assign.PrincipalHasScenarioKey(ctx, userID, scenarioKey)
}

func (s *AssignmentService) ListByRole(ctx context.Context, roleID uuid.UUID, limit, offset int) ([]UserRoleAssignment, error) {
	return s.assign.ListByRole(ctx, roleID, limit, offset)
}

func (s *AssignmentService) Assign(ctx context.Context, assignerID, userID, roleID uuid.UUID, scopeType string, scopeID *uuid.UUID, expiresAt *time.Time) (*UserRoleAssignment, error) {
	active, err := s.def.IsActive(ctx, roleID)
	if err != nil {
		return nil, err
	}
	if !active {
		return nil, fmt.Errorf("cannot assign inactive role")
	}

	priv, err := s.def.IsPrivilegedRole(ctx, roleID)
	if err != nil {
		return nil, err
	}
	if priv {
		if err := s.ensureManagePermissions(ctx, assignerID, scopeType, scopeID); err != nil {
			return nil, err
		}
	}

	if scopeType == "" {
		scopeType = "global"
	}

	a, err := s.assign.CreateBinding(ctx, userID, roleID, scopeType, scopeID, &assignerID, expiresAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, fmt.Errorf("duplicate assignment for this user, role, and scope")
		}
		return nil, err
	}
	return a, nil
}

func (s *AssignmentService) ensureManagePermissions(ctx context.Context, assignerID uuid.UUID, scopeType string, scopeID *uuid.UUID) error {
	if scopeType == "domain" && scopeID != nil {
		dec, err := s.access.Evaluate(ctx, identity_access.EvaluateInput{
			PrincipalID:  assignerID,
			Action:       "manage_permissions",
			ResourceType: "domain",
			DomainID:     scopeID,
		})
		if err != nil {
			return err
		}
		if !dec.Allow || !dec.SensitivityOK {
			return fmt.Errorf("manage_permissions required on target domain to assign privileged role")
		}
		return nil
	}

	doms, err := s.access.DomainIDsWithGrant(ctx, assignerID)
	if err != nil {
		return err
	}
	for _, dom := range doms {
		d := dom
		dec, err := s.access.Evaluate(ctx, identity_access.EvaluateInput{
			PrincipalID:  assignerID,
			Action:       "manage_permissions",
			ResourceType: "domain",
			DomainID:     &d,
		})
		if err != nil {
			return err
		}
		if dec.Allow && dec.SensitivityOK {
			return nil
		}
	}
	return fmt.Errorf("manage_permissions on at least one granted domain required to assign privileged role")
}
