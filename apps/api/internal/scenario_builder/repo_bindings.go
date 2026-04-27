package scenario_builder

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// BindingsRepository persists scenario_*_bindings.
type BindingsRepository struct {
	pool *pgxpool.Pool
}

func NewBindingsRepository(pool *pgxpool.Pool) *BindingsRepository {
	return &BindingsRepository{pool: pool}
}

// ReplaceRoleBindings replaces all role bindings and syncs role_scenario_bindings for scenario_key = code.
func (r *BindingsRepository) ReplaceRoleBindings(ctx context.Context, scenarioID uuid.UUID, scenarioCode string, rows []RoleBindingWrite) error {
	if err := ValidateRoleBindingWrites(rows); err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, row := range rows {
		var n int
		err = tx.QueryRow(ctx, `SELECT COUNT(1) FROM roles WHERE id = $1`, row.RoleID).Scan(&n)
		if err != nil {
			return err
		}
		if n == 0 {
			return fmt.Errorf("role not found: %s", row.RoleID)
		}
	}

	_, err = tx.Exec(ctx, `DELETE FROM scenario_role_bindings WHERE scenario_id = $1`, scenarioID)
	if err != nil {
		return err
	}
	for _, row := range rows {
		_, err = tx.Exec(ctx, `
			INSERT INTO scenario_role_bindings (scenario_id, role_id, can_see, can_run, can_manage, can_review_publish)
			VALUES ($1,$2,$3,$4,$5,$6)`,
			scenarioID, row.RoleID, row.CanSee, row.CanRun, row.CanManage, row.CanReviewPublish)
		if err != nil {
			return err
		}
	}

	_, err = tx.Exec(ctx, `DELETE FROM role_scenario_bindings WHERE scenario_key = $1`, scenarioCode)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if !row.CanSee {
			continue
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO role_scenario_bindings (role_id, scenario_key) VALUES ($1,$2)
			ON CONFLICT (role_id, scenario_key) DO NOTHING`,
			row.RoleID, scenarioCode)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// ReplaceSourceBindings replaces source feed bindings; validates feeds exist.
func (r *BindingsRepository) ReplaceSourceBindings(ctx context.Context, scenarioID uuid.UUID, rows []SourceBindingRow) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, row := range rows {
		var n int
		err = tx.QueryRow(ctx, `SELECT COUNT(1) FROM source_feeds WHERE id = $1`, row.SourceFeedID).Scan(&n)
		if err != nil {
			return err
		}
		if n == 0 {
			return fmt.Errorf("source_feed not found: %s", row.SourceFeedID)
		}
	}

	_, err = tx.Exec(ctx, `DELETE FROM scenario_source_bindings WHERE scenario_id = $1`, scenarioID)
	if err != nil {
		return err
	}
	for _, row := range rows {
		br := row.BindingRole
		if br == "" {
			br = "primary"
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO scenario_source_bindings (scenario_id, source_feed_id, binding_role) VALUES ($1,$2,$3)`,
			scenarioID, row.SourceFeedID, br)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// ReplaceJobBindings replaces job links; validates jobs exist.
func (r *BindingsRepository) ReplaceJobBindings(ctx context.Context, scenarioID uuid.UUID, rows []JobBindingRow) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, row := range rows {
		var n int
		err = tx.QueryRow(ctx, `SELECT COUNT(1) FROM knowledge_jobs WHERE id = $1`, row.KnowledgeJobID).Scan(&n)
		if err != nil {
			return err
		}
		if n == 0 {
			return fmt.Errorf("knowledge_job not found: %s", row.KnowledgeJobID)
		}
		rel := row.Relationship
		if rel == "" {
			rel = "supports"
		}
		switch rel {
		case "primary_support", "supports", "optional":
		default:
			return fmt.Errorf("invalid job relationship: %s", row.Relationship)
		}
	}

	_, err = tx.Exec(ctx, `DELETE FROM scenario_job_bindings WHERE scenario_id = $1`, scenarioID)
	if err != nil {
		return err
	}
	for _, row := range rows {
		rel := row.Relationship
		if rel == "" {
			rel = "supports"
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO scenario_job_bindings (scenario_id, knowledge_job_id, relationship) VALUES ($1,$2,$3)`,
			scenarioID, row.KnowledgeJobID, rel)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// ReplaceUIBindings replaces UI surfaces for a scenario.
func (r *BindingsRepository) ReplaceUIBindings(ctx context.Context, scenarioID uuid.UUID, rows []UIBindingRow) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, `DELETE FROM scenario_ui_bindings WHERE scenario_id = $1`, scenarioID)
	if err != nil {
		return err
	}
	for _, row := range rows {
		cfg := normJSON(row.ConfigJSON)
		_, err = tx.Exec(ctx, `
			INSERT INTO scenario_ui_bindings (id, scenario_id, surface_key, nav_group, sort_order, config_json)
			VALUES ($1,$2,$3,$4,$5,$6)`,
			uuid.New(), scenarioID, row.SurfaceKey, row.NavGroup, row.SortOrder, cfg)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// ResyncRoleScenarioMirror deletes and rebuilds role_scenario_bindings for one scenario from scenario_role_bindings.
func (r *BindingsRepository) ResyncRoleScenarioMirror(ctx context.Context, scenarioID uuid.UUID, scenarioCode string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, `DELETE FROM role_scenario_bindings WHERE scenario_key = $1`, scenarioCode)
	if err != nil {
		return err
	}
	rows, err := tx.Query(ctx, `
		SELECT role_id FROM scenario_role_bindings WHERE scenario_id = $1 AND can_see = true`, scenarioID)
	if err != nil {
		return err
	}
	defer rows.Close()
	var roleIDs []uuid.UUID
	for rows.Next() {
		var rid uuid.UUID
		if err := rows.Scan(&rid); err != nil {
			return err
		}
		roleIDs = append(roleIDs, rid)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, rid := range roleIDs {
		_, err = tx.Exec(ctx, `
			INSERT INTO role_scenario_bindings (role_id, scenario_key) VALUES ($1,$2)
			ON CONFLICT (role_id, scenario_key) DO NOTHING`, rid, scenarioCode)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// CodeByID returns scenario code (for mirror after code rename — caller should handle rekey).
func (r *BindingsRepository) CodeByID(ctx context.Context, id uuid.UUID) (string, error) {
	var code string
	err := r.pool.QueryRow(ctx, `SELECT code FROM scenario_definitions WHERE id = $1`, id).Scan(&code)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	return code, err
}

// RemoveRoleScenarioMirrorForCode removes mirror rows for a scenario key (e.g. before code change).
func (r *BindingsRepository) RemoveRoleScenarioMirrorForCode(ctx context.Context, scenarioCode string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM role_scenario_bindings WHERE scenario_key = $1`, scenarioCode)
	return err
}

// UpdateScenarioCode updates code; does not fix role_scenario_bindings keys — caller must re-sync with new code.
func (r *BindingsRepository) UpdateScenarioCode(ctx context.Context, id uuid.UUID, newCode string) error {
	tag, err := r.pool.Exec(ctx, `UPDATE scenario_definitions SET code = $2, updated_at = now() WHERE id = $1`, id, newCode)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
