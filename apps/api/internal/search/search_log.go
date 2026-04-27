package search

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

// LogSearchInteraction records a search for operator analytics (fail-open).
func (s *Service) LogSearchInteraction(ctx context.Context, principal uuid.UUID, filters map[string]string, hitCount int) {
	if s.pool == nil {
		return
	}
	b, err := json.Marshal(filters)
	if err != nil {
		b = []byte("{}")
	}
	q := filters["q"]
	_, _ = s.pool.Exec(ctx, `
		INSERT INTO search_interaction_log (id, principal_id, q, filters_json, hit_count)
		VALUES ($1, $2, $3, $4::jsonb, $5)`,
		uuid.New(), principal, q, b, hitCount)
}
