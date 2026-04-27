package audit

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Event struct {
	ID             uuid.UUID       `json:"id"`
	EventType      string          `json:"event_type"`
	ActorType      string          `json:"actor_type"`
	ActorID        *uuid.UUID      `json:"actor_id,omitempty"`
	TargetType     string          `json:"target_type"`
	TargetID       *uuid.UUID      `json:"target_id,omitempty"`
	Decision       *string         `json:"decision,omitempty"`
	Reason         *string         `json:"reason,omitempty"`
	PolicyRefsJSON json.RawMessage `json:"policy_refs_json,omitempty"`
	TraceRef       *string         `json:"trace_ref,omitempty"`
	MetadataJSON   json.RawMessage `json:"metadata_json,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
}

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

type WriteInput struct {
	EventType      string
	ActorType      string
	ActorID        *uuid.UUID
	TargetType     string
	TargetID       *uuid.UUID
	Decision       *string
	Reason         *string
	PolicyRefsJSON json.RawMessage
	TraceRef       *string
	MetadataJSON   json.RawMessage
}

func (s *Service) Write(ctx context.Context, in WriteInput) error {
	if len(in.PolicyRefsJSON) == 0 {
		in.PolicyRefsJSON = []byte("[]")
	}
	if len(in.MetadataJSON) == 0 {
		in.MetadataJSON = []byte("{}")
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO audit_events (event_type, actor_type, actor_id, target_type, target_id, decision, reason, policy_refs_json, trace_ref, metadata_json)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		in.EventType, in.ActorType, in.ActorID, in.TargetType, in.TargetID, in.Decision, in.Reason, in.PolicyRefsJSON, in.TraceRef, in.MetadataJSON)
	return err
}

func (s *Service) List(ctx context.Context, eventType, targetType string, limit int) ([]Event, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	q := `SELECT id, event_type, actor_type, actor_id, target_type, target_id, decision, reason, policy_refs_json, trace_ref, metadata_json, created_at
		FROM audit_events WHERE 1=1`
	args := []any{}
	n := 1
	if eventType != "" {
		q += ` AND event_type = $` + strconv.Itoa(n)
		args = append(args, eventType)
		n++
	}
	if targetType != "" {
		q += ` AND target_type = $` + strconv.Itoa(n)
		args = append(args, targetType)
		n++
	}
	q += ` ORDER BY created_at DESC LIMIT $` + strconv.Itoa(n)
	args = append(args, limit)

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Event
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.ID, &e.EventType, &e.ActorType, &e.ActorID, &e.TargetType, &e.TargetID, &e.Decision, &e.Reason, &e.PolicyRefsJSON, &e.TraceRef, &e.MetadataJSON, &e.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, e)
	}
	return list, rows.Err()
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*Event, error) {
	var e Event
	err := s.pool.QueryRow(ctx, `
		SELECT id, event_type, actor_type, actor_id, target_type, target_id, decision, reason, policy_refs_json, trace_ref, metadata_json, created_at
		FROM audit_events WHERE id=$1`, id,
	).Scan(&e.ID, &e.EventType, &e.ActorType, &e.ActorID, &e.TargetType, &e.TargetID, &e.Decision, &e.Reason, &e.PolicyRefsJSON, &e.TraceRef, &e.MetadataJSON, &e.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &e, nil
}
