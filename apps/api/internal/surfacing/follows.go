package surfacing

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ScopeFollow is a surfacing preference (not an access grant).
type ScopeFollow struct {
	ID         uuid.UUID `json:"id"`
	UserID     uuid.UUID `json:"user_id"`
	ScopeType  string    `json:"scope_type"`
	RefID      uuid.UUID `json:"ref_id"`
	EntityType string    `json:"entity_type,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type FollowRepo struct{ pool *pgxpool.Pool }

func NewFollowRepo(pool *pgxpool.Pool) *FollowRepo { return &FollowRepo{pool: pool} }

func (r *FollowRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]ScopeFollow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, scope_type, ref_id, entity_type, created_at
		FROM user_scope_follows WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ScopeFollow
	for rows.Next() {
		var f ScopeFollow
		if err := rows.Scan(&f.ID, &f.UserID, &f.ScopeType, &f.RefID, &f.EntityType, &f.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (r *FollowRepo) Add(ctx context.Context, userID uuid.UUID, scopeType string, refID uuid.UUID, entityType string) (*ScopeFollow, error) {
	if entityType == "" {
		entityType = ""
	}
	var f ScopeFollow
	err := r.pool.QueryRow(ctx, `
		INSERT INTO user_scope_follows (id, user_id, scope_type, ref_id, entity_type)
		VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT (user_id, scope_type, ref_id, entity_type) DO UPDATE SET user_id = EXCLUDED.user_id
		RETURNING id, user_id, scope_type, ref_id, entity_type, created_at`,
		uuid.New(), userID, scopeType, refID, entityType,
	).Scan(&f.ID, &f.UserID, &f.ScopeType, &f.RefID, &f.EntityType, &f.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("surfacing follow add: %w", err)
	}
	return &f, nil
}

func (r *FollowRepo) Remove(ctx context.Context, userID uuid.UUID, scopeType string, refID uuid.UUID, entityType string) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM user_scope_follows
		WHERE user_id=$1 AND scope_type=$2 AND ref_id=$3 AND entity_type=$4`,
		userID, scopeType, refID, entityType)
	return err
}
