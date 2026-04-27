package governance

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AnswerFeedback struct {
	ID           uuid.UUID `json:"id"`
	PrincipalID  uuid.UUID `json:"principal_id"`
	TraceID      string    `json:"trace_id"`
	FeedbackKind string    `json:"feedback_kind"`
	Comment      *string   `json:"comment,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type AnswerFeedbackService struct{ pool *pgxpool.Pool }

func NewAnswerFeedbackService(pool *pgxpool.Pool) *AnswerFeedbackService {
	return &AnswerFeedbackService{pool: pool}
}

func (a *AnswerFeedbackService) Submit(ctx context.Context, principal uuid.UUID, traceID, kind string, comment *string) (*AnswerFeedback, error) {
	id := uuid.New()
	_, err := a.pool.Exec(ctx, `
		INSERT INTO answer_feedback (id, principal_id, trace_id, feedback_kind, comment)
		VALUES ($1,$2,$3,$4,$5)`, id, principal, traceID, kind, comment)
	if err != nil {
		return nil, err
	}
	return a.Get(ctx, id)
}

func (a *AnswerFeedbackService) Get(ctx context.Context, id uuid.UUID) (*AnswerFeedback, error) {
	var f AnswerFeedback
	err := a.pool.QueryRow(ctx, `
		SELECT id, principal_id, trace_id, feedback_kind, comment, created_at
		FROM answer_feedback WHERE id=$1`, id,
	).Scan(&f.ID, &f.PrincipalID, &f.TraceID, &f.FeedbackKind, &f.Comment, &f.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (a *AnswerFeedbackService) ListByTrace(ctx context.Context, traceID string, limit int) ([]AnswerFeedback, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := a.pool.Query(ctx, `
		SELECT id, principal_id, trace_id, feedback_kind, comment, created_at
		FROM answer_feedback WHERE trace_id=$1 ORDER BY created_at DESC LIMIT $2`, traceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []AnswerFeedback
	for rows.Next() {
		var f AnswerFeedback
		if err := rows.Scan(&f.ID, &f.PrincipalID, &f.TraceID, &f.FeedbackKind, &f.Comment, &f.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, f)
	}
	return list, rows.Err()
}

func (a *AnswerFeedbackService) ListRecent(ctx context.Context, limit int) ([]AnswerFeedback, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := a.pool.Query(ctx, `
		SELECT id, principal_id, trace_id, feedback_kind, comment, created_at
		FROM answer_feedback ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []AnswerFeedback
	for rows.Next() {
		var f AnswerFeedback
		if err := rows.Scan(&f.ID, &f.PrincipalID, &f.TraceID, &f.FeedbackKind, &f.Comment, &f.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, f)
	}
	return list, rows.Err()
}
