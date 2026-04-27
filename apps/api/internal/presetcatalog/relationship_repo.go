package presetcatalog

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RelationshipRepository reads preset_relationships.
type RelationshipRepository struct {
	pool *pgxpool.Pool
}

func NewRelationshipRepository(pool *pgxpool.Pool) *RelationshipRepository {
	return &RelationshipRepository{pool: pool}
}

// ListFrom returns related entries (outgoing edges).
func (r *RelationshipRepository) ListFrom(ctx context.Context, fromID uuid.UUID) ([]RelatedEntry, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT r.relationship_type,
		       te.id, te.preset_type, te.code, te.name, te.description, te.active, te.metadata_json, te.created_at, te.updated_at
		FROM preset_relationships r
		JOIN preset_catalog_entries te ON te.id = r.to_preset_id
		WHERE r.from_preset_id = $1
		ORDER BY r.relationship_type, te.preset_type, te.code`, fromID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RelatedEntry
	for rows.Next() {
		var rel RelatedEntry
		var desc *string
		if err := rows.Scan(&rel.RelationshipType,
			&rel.Entry.ID, &rel.Entry.PresetType, &rel.Entry.Code, &rel.Entry.Name, &desc, &rel.Entry.Active, &rel.Entry.MetadataJSON, &rel.Entry.CreatedAt, &rel.Entry.UpdatedAt); err != nil {
			return nil, err
		}
		rel.Entry.Description = desc
		if len(rel.Entry.MetadataJSON) == 0 {
			rel.Entry.MetadataJSON = []byte("{}")
		}
		out = append(out, rel)
	}
	return out, rows.Err()
}
