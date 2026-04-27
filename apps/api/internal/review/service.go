package review

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Task struct {
	ID             uuid.UUID  `json:"id"`
	TargetType     string     `json:"target_type"`
	TargetID       uuid.UUID  `json:"target_id"`
	ReviewerID     *uuid.UUID `json:"reviewer_id,omitempty"`
	OwnerID        uuid.UUID  `json:"owner_id"`
	Status         string     `json:"status"`
	DueAt          *time.Time `json:"due_at,omitempty"`
	ResolutionNote *string    `json:"resolution_note,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
}

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

func (s *Service) Create(ctx context.Context, targetType string, targetID, ownerID uuid.UUID, reviewerID *uuid.UUID, due *time.Time) (*Task, error) {
	id := uuid.New()
	_, err := s.pool.Exec(ctx, `
		INSERT INTO review_tasks (id, target_type, target_id, reviewer_id, owner_id, status, due_at)
		VALUES ($1,$2,$3,$4,$5,'pending',$6)`, id, targetType, targetID, reviewerID, ownerID, due)
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*Task, error) {
	var t Task
	err := s.pool.QueryRow(ctx, `
		SELECT id, target_type, target_id, reviewer_id, owner_id, status, due_at, resolution_note, created_at, completed_at
		FROM review_tasks WHERE id=$1`, id,
	).Scan(&t.ID, &t.TargetType, &t.TargetID, &t.ReviewerID, &t.OwnerID, &t.Status, &t.DueAt, &t.ResolutionNote, &t.CreatedAt, &t.CompletedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// OverdueReviewTask includes trust metadata for governance queues.
type OverdueReviewTask struct {
	Task
	Title            string    `json:"title"`
	EntityType       string    `json:"entity_type,omitempty"`
	DomainID         uuid.UUID `json:"domain_id,omitempty"`
	SensitivityLevel int       `json:"sensitivity_level,omitempty"`
	TruthMode        string    `json:"truth_mode,omitempty"`
	OverdueSeconds   int64     `json:"overdue_seconds"`
}

// ListOverdue returns open tasks past due_at with entity metadata when target is an entity.
func (s *Service) ListOverdue(ctx context.Context, limit int) ([]OverdueReviewTask, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	now := time.Now().UTC()
	rows, err := s.pool.Query(ctx, `
		SELECT rt.id, rt.target_type, rt.target_id, rt.reviewer_id, rt.owner_id, rt.status, rt.due_at, rt.resolution_note, rt.created_at, rt.completed_at,
			COALESCE(e.title, ''), COALESCE(e.type, ''), e.domain_id, COALESCE(e.sensitivity_level, 0), COALESCE(e.truth_mode, ''),
			EXTRACT(EPOCH FROM ($1::timestamptz - rt.due_at))::bigint
		FROM review_tasks rt
		LEFT JOIN entities e ON e.id = rt.target_id AND rt.target_type = 'entity' AND e.archived_at IS NULL
		WHERE rt.status IN ('pending','in_progress')
		  AND rt.due_at IS NOT NULL AND rt.due_at < $1
		ORDER BY rt.due_at ASC
		LIMIT $2`, now, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []OverdueReviewTask
	for rows.Next() {
		var o OverdueReviewTask
		var domain *uuid.UUID
		if err := rows.Scan(
			&o.ID, &o.TargetType, &o.TargetID, &o.ReviewerID, &o.OwnerID, &o.Status, &o.DueAt, &o.ResolutionNote, &o.CreatedAt, &o.CompletedAt,
			&o.Title, &o.EntityType, &domain, &o.SensitivityLevel, &o.TruthMode, &o.OverdueSeconds,
		); err != nil {
			return nil, err
		}
		if domain != nil {
			o.DomainID = *domain
		}
		list = append(list, o)
	}
	return list, rows.Err()
}

// ListOverdueInDomains returns overdue tasks scoped to entity domains in the given set.
// Non-entity targets (or missing entity joins) are excluded to avoid cross-domain leakage in governance queues.
func (s *Service) ListOverdueInDomains(ctx context.Context, limit int, domainIDs []uuid.UUID) ([]OverdueReviewTask, error) {
	if len(domainIDs) == 0 {
		return []OverdueReviewTask{}, nil
	}
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	now := time.Now().UTC()
	rows, err := s.pool.Query(ctx, `
		SELECT rt.id, rt.target_type, rt.target_id, rt.reviewer_id, rt.owner_id, rt.status, rt.due_at, rt.resolution_note, rt.created_at, rt.completed_at,
			e.title, e.type, e.domain_id, e.sensitivity_level, e.truth_mode,
			EXTRACT(EPOCH FROM ($1::timestamptz - rt.due_at))::bigint
		FROM review_tasks rt
		JOIN entities e ON e.id = rt.target_id AND rt.target_type = 'entity' AND e.archived_at IS NULL
		WHERE rt.status IN ('pending','in_progress')
		  AND rt.due_at IS NOT NULL AND rt.due_at < $1
		  AND e.domain_id = ANY($3)
		ORDER BY rt.due_at ASC
		LIMIT $2`, now, limit, domainIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []OverdueReviewTask
	for rows.Next() {
		var o OverdueReviewTask
		var domain uuid.UUID
		if err := rows.Scan(
			&o.ID, &o.TargetType, &o.TargetID, &o.ReviewerID, &o.OwnerID, &o.Status, &o.DueAt, &o.ResolutionNote, &o.CreatedAt, &o.CompletedAt,
			&o.Title, &o.EntityType, &domain, &o.SensitivityLevel, &o.TruthMode, &o.OverdueSeconds,
		); err != nil {
			return nil, err
		}
		o.DomainID = domain
		list = append(list, o)
	}
	return list, rows.Err()
}

func (s *Service) List(ctx context.Context) ([]Task, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, target_type, target_id, reviewer_id, owner_id, status, due_at, resolution_note, created_at, completed_at
		FROM review_tasks ORDER BY created_at DESC LIMIT 200`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Task
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.TargetType, &t.TargetID, &t.ReviewerID, &t.OwnerID, &t.Status, &t.DueAt, &t.ResolutionNote, &t.CreatedAt, &t.CompletedAt); err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	return list, rows.Err()
}

func (s *Service) transition(ctx context.Context, id uuid.UUID, status string, note *string) error {
	if note != nil {
		_, err := s.pool.Exec(ctx, `
			UPDATE review_tasks SET status=$2, resolution_note=$3,
				completed_at = CASE WHEN $2 IN ('approved','rejected','changes_requested') THEN now() ELSE completed_at END
			WHERE id=$1`, id, status, *note)
		return err
	}
	_, err := s.pool.Exec(ctx, `
		UPDATE review_tasks SET status=$2,
			completed_at = CASE WHEN $2 IN ('approved','rejected','changes_requested') THEN now() ELSE completed_at END
		WHERE id=$1`, id, status)
	return err
}

func (s *Service) Start(ctx context.Context, id uuid.UUID) error {
	return s.transition(ctx, id, "in_progress", nil)
}

func (s *Service) Approve(ctx context.Context, id uuid.UUID, note *string) error {
	return s.transition(ctx, id, "approved", note)
}

func (s *Service) RequestChanges(ctx context.Context, id uuid.UUID, note *string) error {
	return s.transition(ctx, id, "changes_requested", note)
}

func (s *Service) Reject(ctx context.Context, id uuid.UUID, note *string) error {
	return s.transition(ctx, id, "rejected", note)
}
