package contenthub

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Hub struct {
	ID          uuid.UUID  `json:"id"`
	DomainID    uuid.UUID  `json:"domain_id"`
	Slug        string     `json:"slug"`
	Title       string     `json:"title"`
	Description *string    `json:"description,omitempty"`
	Status      string     `json:"status"`
	CreatedByID *uuid.UUID `json:"created_by_id,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type HubItem struct {
	EntityID  uuid.UUID `json:"entity_id"`
	Role      string    `json:"role"`
	SortOrder int       `json:"sort_order"`
}

type Service struct{ pool *pgxpool.Pool }

func NewService(pool *pgxpool.Pool) *Service { return &Service{pool: pool} }

func (s *Service) ListInDomains(ctx context.Context, domainIDs []uuid.UUID, limit int) ([]Hub, error) {
	if len(domainIDs) == 0 {
		return nil, nil
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, domain_id, slug, title, description, status, created_by_id, created_at, updated_at
		FROM content_hubs
		WHERE domain_id = ANY($1) AND status = 'active'
		ORDER BY updated_at DESC
		LIMIT $2`, domainIDs, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Hub
	for rows.Next() {
		var h Hub
		if err := rows.Scan(&h.ID, &h.DomainID, &h.Slug, &h.Title, &h.Description, &h.Status, &h.CreatedByID, &h.CreatedAt, &h.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, h)
	}
	return list, rows.Err()
}

func (s *Service) GetBySlug(ctx context.Context, domainID uuid.UUID, slug string) (*Hub, []HubItem, error) {
	var h Hub
	err := s.pool.QueryRow(ctx, `
		SELECT id, domain_id, slug, title, description, status, created_by_id, created_at, updated_at
		FROM content_hubs WHERE domain_id=$1 AND slug=$2 AND status != 'archived'`, domainID, slug,
	).Scan(&h.ID, &h.DomainID, &h.Slug, &h.Title, &h.Description, &h.Status, &h.CreatedByID, &h.CreatedAt, &h.UpdatedAt)
	if err != nil {
		return nil, nil, err
	}
	ir, err := s.pool.Query(ctx, `
		SELECT entity_id, role, sort_order FROM content_hub_items WHERE hub_id=$1 ORDER BY sort_order ASC, entity_id`, h.ID)
	if err != nil {
		return nil, nil, err
	}
	defer ir.Close()
	var items []HubItem
	for ir.Next() {
		var it HubItem
		if err := ir.Scan(&it.EntityID, &it.Role, &it.SortOrder); err != nil {
			return nil, nil, err
		}
		items = append(items, it)
	}
	return &h, items, ir.Err()
}

func (s *Service) Create(ctx context.Context, domainID uuid.UUID, slug, title string, description *string, createdBy uuid.UUID) (*Hub, error) {
	id := uuid.New()
	_, err := s.pool.Exec(ctx, `
		INSERT INTO content_hubs (id, domain_id, slug, title, description, status, created_by_id)
		VALUES ($1,$2,$3,$4,$5,'active',$6)`, id, domainID, slug, title, description, createdBy)
	if err != nil {
		return nil, err
	}
	return s.getByID(ctx, id)
}

// GetByID returns hub metadata.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Hub, error) {
	return s.getByID(ctx, id)
}

func (s *Service) getByID(ctx context.Context, id uuid.UUID) (*Hub, error) {
	var h Hub
	err := s.pool.QueryRow(ctx, `
		SELECT id, domain_id, slug, title, description, status, created_by_id, created_at, updated_at
		FROM content_hubs WHERE id=$1`, id,
	).Scan(&h.ID, &h.DomainID, &h.Slug, &h.Title, &h.Description, &h.Status, &h.CreatedByID, &h.CreatedAt, &h.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &h, nil
}

// ListItems returns ordered hub items.
func (s *Service) ListItems(ctx context.Context, hubID uuid.UUID) ([]HubItem, error) {
	ir, err := s.pool.Query(ctx, `
		SELECT entity_id, role, sort_order FROM content_hub_items WHERE hub_id=$1 ORDER BY sort_order ASC, entity_id`, hubID)
	if err != nil {
		return nil, err
	}
	defer ir.Close()
	var items []HubItem
	for ir.Next() {
		var it HubItem
		if err := ir.Scan(&it.EntityID, &it.Role, &it.SortOrder); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, ir.Err()
}

func (s *Service) AddItem(ctx context.Context, hubID, entityID uuid.UUID, role string, sortOrder int) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO content_hub_items (id, hub_id, entity_id, role, sort_order)
		VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT (hub_id, entity_id) DO UPDATE SET role=EXCLUDED.role, sort_order=EXCLUDED.sort_order`,
		uuid.New(), hubID, entityID, role, sortOrder)
	return err
}

// ListHubIDsContainingEntity returns active hub IDs that include the entity.
func (s *Service) ListHubIDsContainingEntity(ctx context.Context, entityID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT DISTINCT i.hub_id
		FROM content_hub_items i
		JOIN content_hubs h ON h.id = i.hub_id AND h.status = 'active'
		WHERE i.entity_id = $1`, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// ListPeerEntityIDsInHubs returns other entity IDs in the same hubs (curated co-members), ordered by hub sort order.
func (s *Service) ListPeerEntityIDsInHubs(ctx context.Context, hubIDs []uuid.UUID, excludeEntity uuid.UUID, limit int) ([]uuid.UUID, error) {
	if len(hubIDs) == 0 || limit <= 0 {
		return nil, nil
	}
	if limit > 50 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		SELECT i.entity_id FROM content_hub_items i
		JOIN content_hubs h ON h.id = i.hub_id AND h.status = 'active'
		WHERE i.hub_id = ANY($1) AND i.entity_id != $2
		ORDER BY i.sort_order ASC, i.entity_id ASC`, hubIDs, excludeEntity)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	seen := make(map[uuid.UUID]struct{})
	var out []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
		if len(out) >= limit {
			break
		}
	}
	return out, rows.Err()
}
