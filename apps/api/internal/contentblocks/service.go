package contentblocks

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Block struct {
	ID             uuid.UUID  `json:"id"`
	DomainID       uuid.UUID  `json:"domain_id"`
	OwnerID        *uuid.UUID `json:"owner_id,omitempty"`
	Title          string     `json:"title"`
	Body           string     `json:"body"`
	TruthMode      string     `json:"truth_mode"`
	LifecycleState string     `json:"lifecycle_state"`
	ApprovalStatus string     `json:"approval_status"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type Service struct{ pool *pgxpool.Pool }

func NewService(pool *pgxpool.Pool) *Service { return &Service{pool: pool} }

func (s *Service) Create(ctx context.Context, domainID uuid.UUID, ownerID *uuid.UUID, title, body string) (*Block, error) {
	id := uuid.New()
	_, err := s.pool.Exec(ctx, `
		INSERT INTO content_blocks (id, domain_id, owner_id, title, body, truth_mode, lifecycle_state, approval_status)
		VALUES ($1,$2,$3,$4,$5,'derived','draft','none')`, id, domainID, ownerID, title, body)
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*Block, error) {
	var b Block
	err := s.pool.QueryRow(ctx, `
		SELECT id, domain_id, owner_id, title, body, truth_mode, lifecycle_state, approval_status, created_at, updated_at
		FROM content_blocks WHERE id=$1`, id,
	).Scan(&b.ID, &b.DomainID, &b.OwnerID, &b.Title, &b.Body, &b.TruthMode, &b.LifecycleState, &b.ApprovalStatus, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (s *Service) AttachToEntity(ctx context.Context, entityID, blockID uuid.UUID, placement string, sortOrder int) error {
	if placement == "" {
		placement = "inline"
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO entity_content_block_refs (entity_id, block_id, placement, sort_order)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT (entity_id, block_id) DO UPDATE SET placement=EXCLUDED.placement, sort_order=EXCLUDED.sort_order`,
		entityID, blockID, placement, sortOrder)
	return err
}

func (s *Service) ListForEntity(ctx context.Context, entityID uuid.UUID) ([]Block, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT b.id, b.domain_id, b.owner_id, b.title, b.body, b.truth_mode, b.lifecycle_state, b.approval_status, b.created_at, b.updated_at
		FROM content_blocks b
		JOIN entity_content_block_refs r ON r.block_id = b.id
		WHERE r.entity_id = $1
		ORDER BY r.sort_order ASC, b.title`, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Block
	for rows.Next() {
		var b Block
		if err := rows.Scan(&b.ID, &b.DomainID, &b.OwnerID, &b.Title, &b.Body, &b.TruthMode, &b.LifecycleState, &b.ApprovalStatus, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, b)
	}
	return list, rows.Err()
}
