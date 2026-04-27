package governance

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ApprovalQueueRow joins review_tasks with entity trust metadata for approvers.
type ApprovalQueueRow struct {
	ReviewTaskID     uuid.UUID  `json:"review_task_id"`
	TargetType       string     `json:"target_type"`
	TargetID         uuid.UUID  `json:"target_id"`
	OwnerID          uuid.UUID  `json:"owner_id"`
	ReviewerID       *uuid.UUID `json:"reviewer_id,omitempty"`
	TaskStatus       string     `json:"task_status"`
	DueAt            *time.Time `json:"due_at,omitempty"`
	Title            string     `json:"title"`
	EntityType       string     `json:"entity_type"`
	DomainID         uuid.UUID  `json:"domain_id"`
	DomainName       string     `json:"domain_name"`
	SensitivityLevel int        `json:"sensitivity_level"`
	TruthMode        string     `json:"truth_mode"`
	LifecycleState   string     `json:"lifecycle_state"`
	ApprovalStatus   string     `json:"approval_status"`
}

type ApprovalQueue struct{ pool *pgxpool.Pool }

func NewApprovalQueue(pool *pgxpool.Pool) *ApprovalQueue {
	return &ApprovalQueue{pool: pool}
}

func (a *ApprovalQueue) ListEntityReviews(ctx context.Context, limit int) ([]ApprovalQueueRow, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	rows, err := a.pool.Query(ctx, `
		SELECT rt.id, rt.target_type, rt.target_id, rt.owner_id, rt.reviewer_id, rt.status, rt.due_at,
			e.title, e.type, e.domain_id, d.name, e.sensitivity_level, e.truth_mode, e.lifecycle_state, e.approval_status
		FROM review_tasks rt
		JOIN entities e ON e.id = rt.target_id AND rt.target_type = 'entity'
		JOIN domains d ON d.id = e.domain_id
		WHERE e.archived_at IS NULL
		  AND rt.status IN ('pending','in_progress')
		ORDER BY rt.due_at NULLS LAST, rt.created_at ASC
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []ApprovalQueueRow
	for rows.Next() {
		var r ApprovalQueueRow
		if err := rows.Scan(&r.ReviewTaskID, &r.TargetType, &r.TargetID, &r.OwnerID, &r.ReviewerID, &r.TaskStatus, &r.DueAt,
			&r.Title, &r.EntityType, &r.DomainID, &r.DomainName, &r.SensitivityLevel, &r.TruthMode, &r.LifecycleState, &r.ApprovalStatus); err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

func (a *ApprovalQueue) ListEntityReviewsInDomains(ctx context.Context, limit int, domainIDs []uuid.UUID) ([]ApprovalQueueRow, error) {
	if len(domainIDs) == 0 {
		return []ApprovalQueueRow{}, nil
	}
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	rows, err := a.pool.Query(ctx, `
		SELECT rt.id, rt.target_type, rt.target_id, rt.owner_id, rt.reviewer_id, rt.status, rt.due_at,
			e.title, e.type, e.domain_id, d.name, e.sensitivity_level, e.truth_mode, e.lifecycle_state, e.approval_status
		FROM review_tasks rt
		JOIN entities e ON e.id = rt.target_id AND rt.target_type = 'entity'
		JOIN domains d ON d.id = e.domain_id
		WHERE e.archived_at IS NULL
		  AND e.domain_id = ANY($2)
		  AND rt.status IN ('pending','in_progress')
		ORDER BY rt.due_at NULLS LAST, rt.created_at ASC
		LIMIT $1`, limit, domainIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []ApprovalQueueRow
	for rows.Next() {
		var r ApprovalQueueRow
		if err := rows.Scan(&r.ReviewTaskID, &r.TargetType, &r.TargetID, &r.OwnerID, &r.ReviewerID, &r.TaskStatus, &r.DueAt,
			&r.Title, &r.EntityType, &r.DomainID, &r.DomainName, &r.SensitivityLevel, &r.TruthMode, &r.LifecycleState, &r.ApprovalStatus); err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, rows.Err()
}
