package permissions

import (
	"context"

	"github.com/google/uuid"

	"github.com/knowledgelayer/api/internal/identity_access"
	sharedperm "github.com/knowledgelayer/api/internal/shared/permissions"
)

// Resolver is the single entry point for permission evaluation from application and feature code.
// It delegates to identity_access.AccessEvaluator without duplicating SQL or policy rules.
type Resolver struct {
	eval *identity_access.AccessEvaluator
}

func NewResolver(eval *identity_access.AccessEvaluator) *Resolver {
	if eval == nil {
		return nil
	}
	return &Resolver{eval: eval}
}

// Evaluate runs the full access decision pipeline (grants, roles, sensitivity, overrides, entity_acl, entity type scope).
func (r *Resolver) Evaluate(ctx context.Context, in identity_access.EvaluateInput) (*identity_access.AccessDecision, error) {
	if r == nil || r.eval == nil {
		dec := &identity_access.AccessDecision{
			Allow:      false,
			ReasonCode: identity_access.ReasonDenyMissingPrincipal,
			Reasons:    []string{"deny: permission resolver unavailable"},
		}
		return dec, nil
	}
	return r.eval.Evaluate(ctx, in)
}

// Resolve returns a portable ResolutionResult (suitable for logging and internal APIs).
func (r *Resolver) Resolve(ctx context.Context, in identity_access.EvaluateInput) (*sharedperm.ResolutionResult, error) {
	dec, err := r.Evaluate(ctx, in)
	if err != nil {
		return nil, err
	}
	return sharedperm.FromAccessDecision(dec), nil
}

// DomainIDsWithGrant returns domain IDs where the user has a non-expired domain_grants row.
func (r *Resolver) DomainIDsWithGrant(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	if r == nil || r.eval == nil {
		return nil, nil
	}
	return r.eval.DomainIDsWithGrant(ctx, userID)
}

// Evaluator exposes the underlying evaluator for legacy call sites that need the concrete type.
func (r *Resolver) Evaluator() *identity_access.AccessEvaluator {
	if r == nil {
		return nil
	}
	return r.eval
}
