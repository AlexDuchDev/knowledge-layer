package knowledge_jobs

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// JobAllowsSourceFeed returns whether the job may read from the given source feed.
// If knowledge_job_sources has any rows for the job, only those declared feeds are allowed (strict).
// If the table has no rows for the job, the JSON source_scope_json field source_feed_id is used (legacy / digest templates).
func JobAllowsSourceFeed(ctx context.Context, pool *pgxpool.Pool, jobID, feedID uuid.UUID) (bool, error) {
	var n int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM knowledge_job_sources WHERE knowledge_job_id = $1`, jobID).Scan(&n); err != nil {
		return false, err
	}
	if n > 0 {
		var m int
		err := pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM knowledge_job_sources
			WHERE knowledge_job_id = $1 AND source_type = 'source_feed' AND source_id = $2`,
			jobID, feedID).Scan(&m)
		return m > 0, err
	}

	var scopeJSON []byte
	err := pool.QueryRow(ctx, `SELECT source_scope_json FROM knowledge_jobs WHERE id = $1`, jobID).Scan(&scopeJSON)
	if err != nil {
		return false, err
	}
	var scope struct {
		SourceFeedID uuid.UUID `json:"source_feed_id"`
	}
	if err := json.Unmarshal(scopeJSON, &scope); err != nil {
		return false, fmt.Errorf("job scope: %w", err)
	}
	return scope.SourceFeedID == feedID, nil
}
