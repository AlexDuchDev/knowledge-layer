package onboarding

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository persists onboarding sessions and related rows.
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// ListTemplates returns seeded templates.
func (r *Repository) ListTemplates(ctx context.Context) ([]Template, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, code, title, description, metadata_json FROM onboarding_templates ORDER BY code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Template
	for rows.Next() {
		var t Template
		if err := rows.Scan(&t.ID, &t.Code, &t.Title, &t.Description, &t.MetadataJSON); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// GetTemplateByCode loads one template.
func (r *Repository) GetTemplateByCode(ctx context.Context, code string) (*Template, error) {
	var t Template
	err := r.pool.QueryRow(ctx, `
		SELECT id, code, title, description, metadata_json FROM onboarding_templates WHERE code = $1`, code,
	).Scan(&t.ID, &t.Code, &t.Title, &t.Description, &t.MetadataJSON)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// CreateSession inserts a draft session.
func (r *Repository) CreateSession(ctx context.Context, createdBy uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `
		INSERT INTO onboarding_sessions (created_by_user_id) VALUES ($1) RETURNING id`, createdBy,
	).Scan(&id)
	return id, err
}

// SessionCreatedBy returns the user who created the session.
func (r *Repository) SessionCreatedBy(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
	var u uuid.UUID
	err := r.pool.QueryRow(ctx, `
		SELECT created_by_user_id FROM onboarding_sessions WHERE id = $1`, id,
	).Scan(&u)
	return u, err
}

// ListSessionsByCreator returns recent sessions for a user.
func (r *Repository) ListSessionsByCreator(ctx context.Context, userID uuid.UUID, limit int) ([]SessionSummary, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, status, template_code, updated_at
		FROM onboarding_sessions
		WHERE created_by_user_id = $1
		ORDER BY updated_at DESC
		LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SessionSummary
	for rows.Next() {
		var s SessionSummary
		var tc *string
		if err := rows.Scan(&s.ID, &s.Status, &tc, &s.UpdatedAt); err != nil {
			return nil, err
		}
		s.TemplateCode = tc
		out = append(out, s)
	}
	return out, rows.Err()
}

// LoadSessionView aggregates session for API.
func (r *Repository) LoadSessionView(ctx context.Context, id uuid.UUID) (*SessionView, error) {
	var v SessionView
	var tc *string
	err := r.pool.QueryRow(ctx, `
		SELECT id, status, template_code, org_profile_json, created_at, updated_at
		FROM onboarding_sessions WHERE id = $1`, id,
	).Scan(&v.ID, &v.Status, &tc, &v.OrgProfileJSON, &v.CreatedAt, &v.UpdatedAt)
	if err != nil {
		return nil, err
	}
	v.TemplateCode = tc
	if len(v.OrgProfileJSON) == 0 {
		v.OrgProfileJSON = []byte("{}")
	}

	v.Steps, err = r.loadSteps(ctx, id)
	if err != nil {
		return nil, err
	}
	v.SelectedPresets, err = r.loadSelectedPresets(ctx, id)
	if err != nil {
		return nil, err
	}
	v.Connectors, err = r.loadConnectors(ctx, id)
	if err != nil {
		return nil, err
	}
	v.FeedDrafts, err = r.loadFeedDrafts(ctx, id)
	if err != nil {
		return nil, err
	}
	v.Assignment, err = r.loadAssignment(ctx, id)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *Repository) loadSteps(ctx context.Context, sessionID uuid.UUID) (map[string]json.RawMessage, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT step_key, payload_json FROM onboarding_session_steps WHERE session_id = $1`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]json.RawMessage{}
	for rows.Next() {
		var k string
		var v json.RawMessage
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		out[k] = v
	}
	return out, rows.Err()
}

func (r *Repository) loadSelectedPresets(ctx context.Context, sessionID uuid.UUID) ([]SelectedPresetRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT osp.id, osp.preset_catalog_entry_id, p.preset_type, p.code, osp.slot, osp.customizations_json
		FROM onboarding_selected_presets osp
		JOIN preset_catalog_entries p ON p.id = osp.preset_catalog_entry_id
		WHERE osp.session_id = $1 ORDER BY osp.slot`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []SelectedPresetRow
	for rows.Next() {
		var row SelectedPresetRow
		if err := rows.Scan(&row.ID, &row.PresetCatalogEntryID, &row.PresetType, &row.PresetCode, &row.Slot, &row.CustomizationsJSON); err != nil {
			return nil, err
		}
		list = append(list, row)
	}
	return list, rows.Err()
}

func (r *Repository) loadConnectors(ctx context.Context, sessionID uuid.UUID) ([]ConnectorRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT connector_family_code, enabled FROM onboarding_connector_selections WHERE session_id = $1 ORDER BY connector_family_code`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []ConnectorRow
	for rows.Next() {
		var c ConnectorRow
		if err := rows.Scan(&c.FamilyCode, &c.Enabled); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

func (r *Repository) loadFeedDrafts(ctx context.Context, sessionID uuid.UUID) ([]json.RawMessage, error) {
	rows, err := r.pool.Query(ctx, `SELECT draft_json FROM onboarding_source_feed_drafts WHERE session_id = $1 ORDER BY created_at`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []json.RawMessage
	for rows.Next() {
		var d json.RawMessage
		if err := rows.Scan(&d); err != nil {
			return nil, err
		}
		list = append(list, d)
	}
	return list, rows.Err()
}

func (r *Repository) loadAssignment(ctx context.Context, sessionID uuid.UUID) (*AssignmentRow, error) {
	var a AssignmentRow
	var admin, dom *uuid.UUID
	err := r.pool.QueryRow(ctx, `
		SELECT initial_admin_user_id, domain_owner_user_id, assignments_json
		FROM onboarding_assignment_drafts WHERE session_id = $1`, sessionID,
	).Scan(&admin, &dom, &a.AssignmentsJSON)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	a.InitialAdminUserID = admin
	a.DomainOwnerUserID = dom
	if len(a.AssignmentsJSON) == 0 {
		a.AssignmentsJSON = []byte("{}")
	}
	return &a, nil
}

// UpdateSessionMeta updates org profile and/or status.
func (r *Repository) UpdateSessionMeta(ctx context.Context, id uuid.UUID, orgProfile json.RawMessage, status *string) error {
	if len(orgProfile) > 0 && string(orgProfile) != "null" {
		if status != nil {
			_, err := r.pool.Exec(ctx, `
				UPDATE onboarding_sessions SET org_profile_json = $2, status = $3, updated_at = now() WHERE id = $1`,
				id, orgProfile, *status)
			return err
		}
		_, err := r.pool.Exec(ctx, `
			UPDATE onboarding_sessions SET org_profile_json = $2, updated_at = now() WHERE id = $1`, id, orgProfile)
		return err
	}
	if status != nil {
		_, err := r.pool.Exec(ctx, `
			UPDATE onboarding_sessions SET status = $2, updated_at = now() WHERE id = $1`, id, *status)
		return err
	}
	return nil
}

// SetTemplateCode sets template fk on session.
func (r *Repository) SetTemplateCode(ctx context.Context, id uuid.UUID, code string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE onboarding_sessions SET template_code = $2, updated_at = now() WHERE id = $1`, id, code)
	return err
}

// UpsertStep merges one step payload.
func (r *Repository) UpsertStep(ctx context.Context, sessionID uuid.UUID, stepKey string, payload json.RawMessage) error {
	if len(payload) == 0 {
		payload = []byte("{}")
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO onboarding_session_steps (session_id, step_key, payload_json, updated_at)
		VALUES ($1,$2,$3,now())
		ON CONFLICT (session_id, step_key) DO UPDATE SET payload_json = EXCLUDED.payload_json, updated_at = now()`,
		sessionID, stepKey, payload)
	return err
}

// ReplaceSelectedPresets removes old selections and inserts new rows.
func (r *Repository) ReplaceSelectedPresets(ctx context.Context, sessionID uuid.UUID, presets []SelectedPresetPatch) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	_, err = tx.Exec(ctx, `DELETE FROM onboarding_selected_presets WHERE session_id = $1`, sessionID)
	if err != nil {
		return err
	}
	for _, p := range presets {
		slot := p.Slot
		if slot == "" {
			slot = "default"
		}
		cj := p.CustomizationsJSON
		if len(cj) == 0 {
			cj = []byte("{}")
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO onboarding_selected_presets (session_id, preset_catalog_entry_id, slot, customizations_json)
			VALUES ($1,$2,$3,$4)`, sessionID, p.PresetCatalogEntryID, slot, cj)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// UpsertConnector sets one connector toggle.
func (r *Repository) UpsertConnector(ctx context.Context, sessionID uuid.UUID, family string, enabled bool) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO onboarding_connector_selections (session_id, connector_family_code, enabled)
		VALUES ($1,$2,$3)
		ON CONFLICT (session_id, connector_family_code) DO UPDATE SET enabled = EXCLUDED.enabled`,
		sessionID, family, enabled)
	return err
}

// ReplaceAssignment upserts assignment draft (full row).
func (r *Repository) ReplaceAssignment(ctx context.Context, sessionID uuid.UUID, a *AssignmentPatch) error {
	if a == nil {
		return nil
	}
	aj := a.AssignmentsJSON
	if len(aj) == 0 {
		aj = []byte("{}")
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO onboarding_assignment_drafts (session_id, initial_admin_user_id, domain_owner_user_id, assignments_json, updated_at)
		VALUES ($1,$2,$3,$4,now())
		ON CONFLICT (session_id) DO UPDATE SET
			initial_admin_user_id = EXCLUDED.initial_admin_user_id,
			domain_owner_user_id = EXCLUDED.domain_owner_user_id,
			assignments_json = EXCLUDED.assignments_json,
			updated_at = now()`,
		sessionID, a.InitialAdminUserID, a.DomainOwnerUserID, aj)
	return err
}

// MarkSessionLaunched sets status and timestamp.
func (r *Repository) MarkSessionLaunched(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE onboarding_sessions SET status = 'launched', updated_at = now() WHERE id = $1`, id)
	return err
}

// InsertLaunchLog creates a log row; returns id.
func (r *Repository) InsertLaunchLog(ctx context.Context, sessionID uuid.UUID) (uuid.UUID, error) {
	var lid uuid.UUID
	err := r.pool.QueryRow(ctx, `
		INSERT INTO onboarding_launch_logs (session_id, status) VALUES ($1, 'running') RETURNING id`, sessionID,
	).Scan(&lid)
	return lid, err
}

// FinishLaunchLog updates log outcome.
func (r *Repository) FinishLaunchLog(ctx context.Context, logID uuid.UUID, status string, result json.RawMessage, errText *string) error {
	if len(result) == 0 {
		result = []byte("{}")
	}
	_, err := r.pool.Exec(ctx, `
		UPDATE onboarding_launch_logs SET finished_at = now(), status = $2, result_json = $3, error_text = $4 WHERE id = $1`,
		logID, status, result, errText)
	return err
}

// SessionStatus returns current status or ErrNoRows.
func (r *Repository) SessionStatus(ctx context.Context, id uuid.UUID) (string, error) {
	var st string
	err := r.pool.QueryRow(ctx, `SELECT status FROM onboarding_sessions WHERE id = $1`, id).Scan(&st)
	return st, err
}

// CatalogIDsByTypeAndCodes resolves preset catalog UUIDs for template metadata.
func (r *Repository) CatalogIDsByTypeAndCodes(ctx context.Context, presetType string, codes []string) (map[string]uuid.UUID, error) {
	if len(codes) == 0 {
		return map[string]uuid.UUID{}, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT code, id FROM preset_catalog_entries
		WHERE preset_type = $1 AND code = ANY($2::text[])`, presetType, codes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]uuid.UUID)
	for rows.Next() {
		var code string
		var id uuid.UUID
		if err := rows.Scan(&code, &id); err != nil {
			return nil, err
		}
		out[code] = id
	}
	return out, rows.Err()
}
