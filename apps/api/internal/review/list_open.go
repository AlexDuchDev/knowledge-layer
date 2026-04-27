package review

import (
	"context"

	"github.com/google/uuid"
)

// OpenTaskPreview is a pending/in-progress review task scoped to an entity in allowed domains.
type OpenTaskPreview struct {
	Task
	Title      string    `json:"title,omitempty"`
	EntityType string    `json:"entity_type,omitempty"`
	DomainID   uuid.UUID `json:"domain_id,omitempty"`
}

// ListOpenInDomains returns open review tasks for entity targets in the given domains.
// If governanceView is true, returns all open tasks in those domains; otherwise only tasks owned by or assigned to principal.
func (s *Service) ListOpenInDomains(ctx context.Context, domainIDs []uuid.UUID, principal uuid.UUID, governanceView bool, limit int) ([]OpenTaskPreview, error) {
	if len(domainIDs) == 0 {
		return nil, nil
	}
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	q := `
		SELECT rt.id, rt.target_type, rt.target_id, rt.reviewer_id, rt.owner_id, rt.status, rt.due_at, rt.resolution_note, rt.created_at, rt.completed_at,
			COALESCE(e.title, ''), COALESCE(e.type, ''), e.domain_id
		FROM review_tasks rt
		JOIN entities e ON e.id = rt.target_id AND rt.target_type = 'entity' AND e.archived_at IS NULL
		WHERE rt.status IN ('pending','in_progress')
		  AND e.domain_id = ANY($1)`
	args := []any{domainIDs}
	if !governanceView {
		q += ` AND (rt.owner_id = $2 OR rt.reviewer_id = $2)`
		args = append(args, principal)
		q += ` ORDER BY rt.created_at DESC LIMIT $3`
		args = append(args, limit)
	} else {
		q += ` ORDER BY rt.created_at DESC LIMIT $2`
		args = append(args, limit)
	}
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []OpenTaskPreview
	for rows.Next() {
		var o OpenTaskPreview
		if err := rows.Scan(
			&o.ID, &o.TargetType, &o.TargetID, &o.ReviewerID, &o.OwnerID, &o.Status, &o.DueAt, &o.ResolutionNote, &o.CreatedAt, &o.CompletedAt,
			&o.Title, &o.EntityType, &o.DomainID,
		); err != nil {
			return nil, err
		}
		list = append(list, o)
	}
	return list, rows.Err()
}
