package presetcatalog

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CatalogRepository reads preset_catalog_entries and categories.
type CatalogRepository struct {
	pool *pgxpool.Pool
}

func NewCatalogRepository(pool *pgxpool.Pool) *CatalogRepository {
	return &CatalogRepository{pool: pool}
}

// List filters by preset_type and optional category axis+code.
func (r *CatalogRepository) List(ctx context.Context, presetType, categoryAxis, categoryCode string) ([]ListRow, error) {
	presetType = strings.TrimSpace(presetType)
	categoryAxis = strings.TrimSpace(categoryAxis)
	categoryCode = strings.TrimSpace(categoryCode)

	var q string
	var args []any
	if categoryAxis != "" && categoryCode != "" {
		q = `
			SELECT e.id, e.preset_type, e.code, e.name, e.description, e.active, e.metadata_json, e.created_at, e.updated_at
			FROM preset_catalog_entries e
			INNER JOIN preset_catalog_category_assignments a ON a.preset_catalog_entry_id = e.id
			INNER JOIN preset_categories c ON c.id = a.category_id AND c.axis = $2 AND c.code = $3
			WHERE e.active = true AND ($1::text = '' OR e.preset_type = $1)
			ORDER BY e.preset_type, e.code`
		args = []any{presetType, categoryAxis, categoryCode}
	} else {
		q = `
			SELECT id, preset_type, code, name, description, active, metadata_json, created_at, updated_at
			FROM preset_catalog_entries
			WHERE active = true AND ($1::text = '' OR preset_type = $1)
			ORDER BY preset_type, code`
		args = []any{presetType}
	}

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []CatalogEntry
	for rows.Next() {
		var e CatalogEntry
		var desc *string
		if err := rows.Scan(&e.ID, &e.PresetType, &e.Code, &e.Name, &desc, &e.Active, &e.MetadataJSON, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		e.Description = desc
		if len(e.MetadataJSON) == 0 {
			e.MetadataJSON = []byte("{}")
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]ListRow, 0, len(entries))
	for _, e := range entries {
		cats, err := r.categoriesForEntry(ctx, e.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, ListRow{CatalogEntry: e, Categories: cats})
	}
	return out, nil
}

func (r *CatalogRepository) categoriesForEntry(ctx context.Context, entryID uuid.UUID) ([]CategoryRef, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT c.axis, c.code, c.label
		FROM preset_catalog_category_assignments a
		JOIN preset_categories c ON c.id = a.category_id
		WHERE a.preset_catalog_entry_id = $1
		ORDER BY c.axis, c.sort_order, c.code`, entryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CategoryRef
	for rows.Next() {
		var c CategoryRef
		if err := rows.Scan(&c.Axis, &c.Code, &c.Label); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// GetByID returns one entry or ErrNoRows.
func (r *CatalogRepository) GetByID(ctx context.Context, id uuid.UUID) (*CatalogEntry, error) {
	var e CatalogEntry
	var desc *string
	err := r.pool.QueryRow(ctx, `
		SELECT id, preset_type, code, name, description, active, metadata_json, created_at, updated_at
		FROM preset_catalog_entries WHERE id = $1`, id,
	).Scan(&e.ID, &e.PresetType, &e.Code, &e.Name, &desc, &e.Active, &e.MetadataJSON, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return nil, err
	}
	e.Description = desc
	if len(e.MetadataJSON) == 0 {
		e.MetadataJSON = []byte("{}")
	}
	return &e, nil
}

// GetByTypeAndCode loads by natural key.
func (r *CatalogRepository) GetByTypeAndCode(ctx context.Context, presetType, code string) (*CatalogEntry, error) {
	var e CatalogEntry
	var desc *string
	err := r.pool.QueryRow(ctx, `
		SELECT id, preset_type, code, name, description, active, metadata_json, created_at, updated_at
		FROM preset_catalog_entries WHERE preset_type = $1 AND code = $2`, presetType, code,
	).Scan(&e.ID, &e.PresetType, &e.Code, &e.Name, &desc, &e.Active, &e.MetadataJSON, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return nil, err
	}
	e.Description = desc
	if len(e.MetadataJSON) == 0 {
		e.MetadataJSON = []byte("{}")
	}
	return &e, nil
}

// RoleTemplateID resolves the template role UUID for a role preset code (roles.preset_key).
func (r *CatalogRepository) RoleTemplateID(ctx context.Context, presetKey string) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `
		SELECT id FROM roles WHERE preset_key = $1 AND is_preset = true AND active = true LIMIT 1`, presetKey,
	).Scan(&id)
	return id, err
}

// ScenarioPresetTemplate returns scenario_presets row JSON for preview.
func (r *CatalogRepository) ScenarioPresetTemplate(ctx context.Context, presetKey string) (name string, description *string, scenarioType string, templateJSON json.RawMessage, err error) {
	err = r.pool.QueryRow(ctx, `
		SELECT name, description, scenario_type, template_json FROM scenario_presets WHERE preset_key = $1`, presetKey,
	).Scan(&name, &description, &scenarioType, &templateJSON)
	return
}

// JobBuilderPresetRow loads job_builder_presets for preview.
func (r *CatalogRepository) JobBuilderPresetRow(ctx context.Context, presetKey string) (name string, description *string, templateKey string, defaultsJSON json.RawMessage, err error) {
	err = r.pool.QueryRow(ctx, `
		SELECT name, description, template_key, defaults_json FROM job_builder_presets WHERE preset_key = $1`, presetKey,
	).Scan(&name, &description, &templateKey, &defaultsJSON)
	return
}

// ValidateRelationshipPair checks allowed from/to type combinations.
func ValidateRelationshipPair(relType, fromType, toType string) error {
	switch relType {
	case "role_recommends_scenario":
		if fromType != "role" || toType != "scenario" {
			return fmt.Errorf("role_recommends_scenario requires role -> scenario")
		}
	case "role_recommends_job":
		if fromType != "role" || toType != "job" {
			return fmt.Errorf("role_recommends_job requires role -> job")
		}
	case "scenario_recommends_job":
		if fromType != "scenario" || toType != "job" {
			return fmt.Errorf("scenario_recommends_job requires scenario -> job")
		}
	case "job_pairs_with_scenario":
		if fromType != "job" || toType != "scenario" {
			return fmt.Errorf("job_pairs_with_scenario requires job -> scenario")
		}
	default:
		return fmt.Errorf("unknown relationship_type %q", relType)
	}
	return nil
}

// ErrNotFound wraps pgx.ErrNoRows for callers.
func IsNotFound(err error) bool {
	return err == pgx.ErrNoRows
}
