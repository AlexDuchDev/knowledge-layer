package governance

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UpkeepSuggestion is a deterministic, explainable stewardship hint (not an auto-fix).
type UpkeepSuggestion struct {
	EntityID        uuid.UUID `json:"entity_id"`
	Title           string    `json:"title"`
	DomainID        uuid.UUID `json:"domain_id"`
	Type            string    `json:"type"`
	LifecycleState  string    `json:"lifecycle_state"`
	FreshnessStatus string    `json:"freshness_status"`
	Reason          string    `json:"reason"`
	Evidence        string    `json:"evidence"`
}

// ListUpkeepSuggestions returns narrow heuristic hints for operators (no LLM).
func ListUpkeepSuggestions(ctx context.Context, pool *pgxpool.Pool, domainIDs []uuid.UUID, limit int) ([]UpkeepSuggestion, error) {
	if len(domainIDs) == 0 {
		return nil, nil
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	seen := make(map[uuid.UUID]struct{})
	var out []UpkeepSuggestion

	add := func(s UpkeepSuggestion) {
		if len(out) >= limit {
			return
		}
		if _, ok := seen[s.EntityID]; ok {
			return
		}
		seen[s.EntityID] = struct{}{}
		out = append(out, s)
	}

	stale, err := ListStaleEntities(ctx, pool, domainIDs, limit)
	if err != nil {
		return nil, err
	}
	for _, e := range stale {
		add(UpkeepSuggestion{
			EntityID:        e.ID,
			Title:           e.Title,
			DomainID:        e.DomainID,
			Type:            "",
			LifecycleState:  e.LifecycleState,
			FreshnessStatus: e.FreshnessStatus,
			Reason:          "freshness_not_fresh",
			Evidence:        "freshness_status=" + e.FreshnessStatus,
		})
	}

	rows, err := pool.Query(ctx, `
		SELECT id, title, domain_id, type, lifecycle_state, freshness_status, COALESCE(summary,'')
		FROM entities
		WHERE archived_at IS NULL AND domain_id = ANY($1)
		  AND lifecycle_state = 'published'
		  AND (summary IS NULL OR length(trim(summary)) < 8)
		ORDER BY updated_at ASC
		LIMIT $2`, domainIDs, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, domain uuid.UUID
		var title, typ, life, fresh, sum string
		if err := rows.Scan(&id, &title, &domain, &typ, &life, &fresh, &sum); err != nil {
			return nil, err
		}
		ev := "summary_empty_or_short"
		if strings.TrimSpace(sum) != "" {
			ev = "summary_short"
		}
		add(UpkeepSuggestion{
			EntityID:        id,
			Title:           title,
			DomainID:        domain,
			Type:            typ,
			LifecycleState:  life,
			FreshnessStatus: fresh,
			Reason:          "weak_summary",
			Evidence:        ev,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	rows2, err := pool.Query(ctx, `
		SELECT e.id, e.title, e.domain_id, e.type, e.lifecycle_state, e.freshness_status
		FROM entities e
		WHERE e.archived_at IS NULL AND e.domain_id = ANY($1)
		  AND e.type IN ('decision','policy')
		  AND e.lifecycle_state = 'published'
		  AND NOT EXISTS (
		    SELECT 1 FROM entity_links l WHERE l.from_entity_id = e.id OR l.to_entity_id = e.id
		  )
		ORDER BY e.updated_at ASC
		LIMIT $2`, domainIDs, limit)
	if err != nil {
		return nil, err
	}
	defer rows2.Close()
	for rows2.Next() {
		var id, domain uuid.UUID
		var title, typ, life, fresh string
		if err := rows2.Scan(&id, &title, &domain, &typ, &life, &fresh); err != nil {
			return nil, err
		}
		add(UpkeepSuggestion{
			EntityID:        id,
			Title:           title,
			DomainID:        domain,
			Type:            typ,
			LifecycleState:  life,
			FreshnessStatus: fresh,
			Reason:          "no_explicit_links",
			Evidence:        "expected relations for type " + typ,
		})
	}
	return out, rows2.Err()
}
