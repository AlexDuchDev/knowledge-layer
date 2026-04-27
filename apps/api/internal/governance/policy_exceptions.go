package governance

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrReasonRequired is returned when creating a policy exception without a reason.
var ErrReasonRequired = errors.New("reason is required for policy exceptions")

type PolicyException struct {
	ID              uuid.UUID       `json:"id"`
	TargetType      string          `json:"target_type"`
	TargetID        uuid.UUID       `json:"target_id"`
	OverrideType    string          `json:"override_type"`
	PolicyPayload   json.RawMessage `json:"policy_payload"`
	Reason          *string         `json:"reason,omitempty"`
	CreatedBy       *uuid.UUID      `json:"created_by,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	ExpiresAt       *time.Time      `json:"expires_at,omitempty"`
	Status          string          `json:"status"`
	EffectiveStatus string          `json:"effective_status"`
	ReviewedAt      *time.Time      `json:"reviewed_at,omitempty"`
	ReviewedBy      *uuid.UUID      `json:"reviewed_by,omitempty"`
	RevokedAt       *time.Time      `json:"revoked_at,omitempty"`
	RevokedBy       *uuid.UUID      `json:"revoked_by,omitempty"`
}

type PolicyExceptionService struct{ pool *pgxpool.Pool }

func NewPolicyExceptionService(pool *pgxpool.Pool) *PolicyExceptionService {
	return &PolicyExceptionService{pool: pool}
}

func effectiveExceptionStatus(status string, expiresAt *time.Time, now time.Time) string {
	if status == "revoked" {
		return "revoked"
	}
	if expiresAt != nil && !expiresAt.After(now) {
		return "expired"
	}
	return status
}

func (s *PolicyExceptionService) List(ctx context.Context, limit int) ([]PolicyException, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, target_type, target_id, override_type, policy_payload, reason, created_by, created_at, expires_at,
			COALESCE(NULLIF(status,''), 'active'), reviewed_at, reviewed_by, revoked_at, revoked_by
		FROM policy_overrides ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	now := time.Now().UTC()
	var list []PolicyException
	for rows.Next() {
		var p PolicyException
		if err := rows.Scan(&p.ID, &p.TargetType, &p.TargetID, &p.OverrideType, &p.PolicyPayload, &p.Reason,
			&p.CreatedBy, &p.CreatedAt, &p.ExpiresAt, &p.Status, &p.ReviewedAt, &p.ReviewedBy, &p.RevokedAt, &p.RevokedBy); err != nil {
			return nil, err
		}
		p.EffectiveStatus = effectiveExceptionStatus(p.Status, p.ExpiresAt, now)
		list = append(list, p)
	}
	return list, rows.Err()
}

type CreatePolicyExceptionInput struct {
	TargetType    string          `json:"target_type"`
	TargetID      uuid.UUID       `json:"target_id"`
	OverrideType  string          `json:"override_type"`
	PolicyPayload json.RawMessage `json:"policy_payload"`
	Reason        string          `json:"reason"`
	ExpiresAt     *time.Time      `json:"expires_at,omitempty"`
}

func (s *PolicyExceptionService) Create(ctx context.Context, principal uuid.UUID, in CreatePolicyExceptionInput) (*PolicyException, error) {
	if in.Reason == "" {
		return nil, ErrReasonRequired
	}
	if len(in.PolicyPayload) == 0 {
		in.PolicyPayload = []byte("{}")
	}
	id := uuid.New()
	_, err := s.pool.Exec(ctx, `
		INSERT INTO policy_overrides (id, target_type, target_id, override_type, policy_payload, reason, created_by, expires_at, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'pending_review')`,
		id, in.TargetType, in.TargetID, in.OverrideType, in.PolicyPayload, in.Reason, principal, in.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

func (s *PolicyExceptionService) Get(ctx context.Context, id uuid.UUID) (*PolicyException, error) {
	var p PolicyException
	err := s.pool.QueryRow(ctx, `
		SELECT id, target_type, target_id, override_type, policy_payload, reason, created_by, created_at, expires_at,
			COALESCE(NULLIF(status,''), 'active'), reviewed_at, reviewed_by, revoked_at, revoked_by
		FROM policy_overrides WHERE id=$1`, id,
	).Scan(&p.ID, &p.TargetType, &p.TargetID, &p.OverrideType, &p.PolicyPayload, &p.Reason,
		&p.CreatedBy, &p.CreatedAt, &p.ExpiresAt, &p.Status, &p.ReviewedAt, &p.ReviewedBy, &p.RevokedAt, &p.RevokedBy)
	if err != nil {
		return nil, err
	}
	p.EffectiveStatus = effectiveExceptionStatus(p.Status, p.ExpiresAt, time.Now().UTC())
	return &p, nil
}

func (s *PolicyExceptionService) MarkReviewed(ctx context.Context, id uuid.UUID, reviewer uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE policy_overrides SET status='active', reviewed_at=now(), reviewed_by=$2 WHERE id=$1 AND status='pending_review'`, id, reviewer)
	return err
}

func (s *PolicyExceptionService) Revoke(ctx context.Context, id uuid.UUID, actor uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE policy_overrides SET status='revoked', revoked_at=now(), revoked_by=$2 WHERE id=$1`, id, actor)
	return err
}
