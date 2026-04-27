package role_builder

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AssignmentRepository wraps user_role_bindings for Role Builder.
type AssignmentRepository struct {
	pool *pgxpool.Pool
}

func NewAssignmentRepository(pool *pgxpool.Pool) *AssignmentRepository {
	return &AssignmentRepository{pool: pool}
}

func (r *AssignmentRepository) ListByRole(ctx context.Context, roleID uuid.UUID, limit, offset int) ([]UserRoleAssignment, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, role_id, scope_type, scope_id, granted_by, granted_at, expires_at
		FROM user_role_bindings WHERE role_id = $1
		ORDER BY granted_at DESC LIMIT $2 OFFSET $3`, roleID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []UserRoleAssignment
	for rows.Next() {
		var a UserRoleAssignment
		if err := rows.Scan(&a.ID, &a.UserID, &a.RoleID, &a.ScopeType, &a.ScopeID, &a.GrantedBy, &a.GrantedAt, &a.ExpiresAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// CreateBinding inserts a user_role_bindings row.
func (r *AssignmentRepository) CreateBinding(ctx context.Context, userID, roleID uuid.UUID, scopeType string, scopeID *uuid.UUID, grantedBy *uuid.UUID, expiresAt *time.Time) (*UserRoleAssignment, error) {
	id := uuid.New()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO user_role_bindings (id, user_id, role_id, scope_type, scope_id, granted_by, granted_at, expires_at)
		VALUES ($1,$2,$3,$4,$5,$6,now(),$7)`,
		id, userID, roleID, scopeType, scopeID, grantedBy, expiresAt)
	if err != nil {
		return nil, err
	}
	var a UserRoleAssignment
	err = r.pool.QueryRow(ctx, `
		SELECT id, user_id, role_id, scope_type, scope_id, granted_by, granted_at, expires_at
		FROM user_role_bindings WHERE id = $1`, id,
	).Scan(&a.ID, &a.UserID, &a.RoleID, &a.ScopeType, &a.ScopeID, &a.GrantedBy, &a.GrantedAt, &a.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// PrincipalHasScenarioKey is true when the user has a non-expired role assignment whose role lists scenario_key in role_scenario_bindings.
func (r *AssignmentRepository) PrincipalHasScenarioKey(ctx context.Context, userID uuid.UUID, scenarioKey string) (bool, error) {
	sk := strings.TrimSpace(scenarioKey)
	if sk == "" {
		return false, nil
	}
	var ok bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM user_role_bindings urb
			INNER JOIN role_scenario_bindings rsb ON rsb.role_id = urb.role_id
			WHERE urb.user_id = $1
			  AND (urb.expires_at IS NULL OR urb.expires_at > now())
			  AND rsb.scenario_key = $2
		)`, userID, sk).Scan(&ok)
	if err != nil {
		return false, err
	}
	return ok, nil
}
