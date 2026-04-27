package governance

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// StaleEntity is a lightweight row for stale / unknown freshness queue.
type StaleEntity struct {
	ID              uuid.UUID `json:"id"`
	Title           string    `json:"title"`
	DomainID        uuid.UUID `json:"domain_id"`
	FreshnessStatus string    `json:"freshness_status"`
	LifecycleState  string    `json:"lifecycle_state"`
	TruthMode       string    `json:"truth_mode"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// ListStaleEntities returns entities in the given domains that are not fresh.
func ListStaleEntities(ctx context.Context, pool *pgxpool.Pool, domainIDs []uuid.UUID, limit int) ([]StaleEntity, error) {
	if len(domainIDs) == 0 {
		return []StaleEntity{}, nil
	}
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	rows, err := pool.Query(ctx, `
		SELECT id, title, domain_id, freshness_status, lifecycle_state, truth_mode, updated_at
		FROM entities
		WHERE archived_at IS NULL
		  AND domain_id = ANY($1)
		  AND freshness_status IS DISTINCT FROM 'fresh'
		ORDER BY updated_at ASC
		LIMIT $2`, domainIDs, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []StaleEntity
	for rows.Next() {
		var e StaleEntity
		if err := rows.Scan(&e.ID, &e.Title, &e.DomainID, &e.FreshnessStatus, &e.LifecycleState, &e.TruthMode, &e.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, e)
	}
	return list, rows.Err()
}
