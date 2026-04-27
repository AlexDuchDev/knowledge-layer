package identity_access

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// entityACLBlocksPrincipal is true when an explicit deny row applies to the user (direct or via team).
func entityACLBlocksPrincipal(ctx context.Context, pool *pgxpool.Pool, entityID, userID uuid.UUID) (bool, error) {
	var n int
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM entity_acl ea
		WHERE ea.entity_id = $1 AND ea.effect = 'deny'
		  AND (
		    (ea.principal_type = 'user' AND ea.principal_id = $2)
		    OR (ea.principal_type = 'team' AND EXISTS (
		      SELECT 1 FROM user_team_memberships m
		      WHERE m.user_id = $2 AND m.team_id = ea.principal_id
		    ))
		  )`, entityID, userID).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("entity_acl: %w", err)
	}
	return n > 0, nil
}

// entityACLAllowsPrincipal is true when an explicit allow row applies to the user (direct or via team).
func entityACLAllowsPrincipal(ctx context.Context, pool *pgxpool.Pool, entityID, userID uuid.UUID) (bool, error) {
	var n int
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM entity_acl ea
		WHERE ea.entity_id = $1 AND ea.effect = 'allow'
		  AND (
		    (ea.principal_type = 'user' AND ea.principal_id = $2)
		    OR (ea.principal_type = 'team' AND EXISTS (
		      SELECT 1 FROM user_team_memberships m
		      WHERE m.user_id = $2 AND m.team_id = ea.principal_id
		    ))
		  )`, entityID, userID).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("entity_acl allow: %w", err)
	}
	return n > 0, nil
}

// entityTypeAllowedForDomain enforces access_policies.entity_type_scope when any active policy narrows by type.
func entityTypeAllowedForDomain(ctx context.Context, pool *pgxpool.Pool, domainID uuid.UUID, entityType string) (bool, error) {
	rows, err := pool.Query(ctx, `
		SELECT DISTINCT trim(entity_type_scope) AS ts FROM access_policies
		WHERE domain_id = $1 AND status = 'active'
		  AND entity_type_scope IS NOT NULL AND trim(entity_type_scope) <> ''`, domainID)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	var scopes []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return false, err
		}
		scopes = append(scopes, s)
	}
	if err := rows.Err(); err != nil {
		return false, err
	}
	if len(scopes) == 0 {
		return true, nil
	}
	for _, s := range scopes {
		if s == "*" || strings.EqualFold(s, entityType) {
			return true, nil
		}
	}
	return false, nil
}
