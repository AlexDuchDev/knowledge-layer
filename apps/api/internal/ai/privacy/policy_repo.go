package privacy

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PolicyRepo loads policy rules from Postgres.
type PolicyRepo struct {
	pool *pgxpool.Pool
}

// NewPolicyRepo constructs a policy repo.
func NewPolicyRepo(pool *pgxpool.Pool) *PolicyRepo {
	return &PolicyRepo{pool: pool}
}

// ListEnabled returns all enabled rules (cached resolution can be added later).
func (r *PolicyRepo) ListEnabled(ctx context.Context) ([]PolicyRule, error) {
	if r == nil || r.pool == nil {
		return nil, fmt.Errorf("privacy policy repo: nil pool")
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, scope_kind, scope_id, entity_type, action, rehydration_mode, priority, enabled
		FROM ai_privacy_policy_rules
		WHERE enabled = true
		ORDER BY scope_kind, priority DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []PolicyRule
	for rows.Next() {
		var ru PolicyRule
		var scopeID *string
		var et, sk, act, rh string
		if err := rows.Scan(&ru.ID, &sk, &scopeID, &et, &act, &rh, &ru.Priority, &ru.Enabled); err != nil {
			return nil, err
		}
		ru.ScopeKind = PolicyScopeKind(sk)
		ru.ScopeID = scopeID
		ru.EntityType = SensitiveEntityType(et)
		ru.Action = PolicyAction(act)
		ru.RehydrationMode = RehydrationMode(rh)
		list = append(list, ru)
	}
	return list, rows.Err()
}
