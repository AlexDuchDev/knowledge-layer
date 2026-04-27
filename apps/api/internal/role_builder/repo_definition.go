package role_builder

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DefinitionRepository persists role definitions and bindings.
type DefinitionRepository struct {
	pool *pgxpool.Pool
}

func NewDefinitionRepository(pool *pgxpool.Pool) *DefinitionRepository {
	return &DefinitionRepository{pool: pool}
}

func (r *DefinitionRepository) ListSummaries(ctx context.Context) ([]RoleSummary, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, code, name, description, category, active, scope_model, is_preset, preset_key, is_system, created_at, updated_at
		FROM roles ORDER BY category, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RoleSummary
	for rows.Next() {
		var s RoleSummary
		if err := rows.Scan(&s.ID, &s.Code, &s.Name, &s.Description, &s.Category, &s.Active, &s.ScopeModel, &s.IsPreset, &s.PresetKey, &s.IsSystem, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *DefinitionRepository) GetByPresetKey(ctx context.Context, key string) (*RoleSummary, error) {
	var s RoleSummary
	err := r.pool.QueryRow(ctx, `
		SELECT id, code, name, description, category, active, scope_model, is_preset, preset_key, is_system, created_at, updated_at
		FROM roles WHERE preset_key = $1 AND active = true`, key,
	).Scan(&s.ID, &s.Code, &s.Name, &s.Description, &s.Category, &s.Active, &s.ScopeModel, &s.IsPreset, &s.PresetKey, &s.IsSystem, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *DefinitionRepository) ListPresets(ctx context.Context) ([]RoleSummary, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, code, name, description, category, active, scope_model, is_preset, preset_key, is_system, created_at, updated_at
		FROM roles WHERE is_preset = true AND active = true ORDER BY preset_key NULLS LAST, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RoleSummary
	for rows.Next() {
		var s RoleSummary
		if err := rows.Scan(&s.ID, &s.Code, &s.Name, &s.Description, &s.Category, &s.Active, &s.ScopeModel, &s.IsPreset, &s.PresetKey, &s.IsSystem, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *DefinitionRepository) GetFull(ctx context.Context, id uuid.UUID) (*RoleFull, error) {
	var f RoleFull
	var spc sql.NullString
	err := r.pool.QueryRow(ctx, `
		SELECT id, code, name, description, category, active, scope_model, is_preset, preset_key, is_system,
		       cloned_from_role_id, source_preset_code, created_at, updated_at
		FROM roles WHERE id = $1`, id,
	).Scan(&f.ID, &f.Code, &f.Name, &f.Description, &f.Category, &f.Active, &f.ScopeModel, &f.IsPreset, &f.PresetKey, &f.IsSystem, &f.ClonedFromRoleID, &spc, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if spc.Valid && spc.String != "" {
		v := spc.String
		f.SourcePresetCode = &v
	}

	rows, err := r.pool.Query(ctx, `SELECT ap.code FROM role_action_permissions rap JOIN action_permissions ap ON ap.id = rap.action_permission_id WHERE rap.role_id = $1 ORDER BY ap.code`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		f.ActionCodes = append(f.ActionCodes, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	rows2, err := r.pool.Query(ctx, `SELECT domain_id FROM role_domain_bindings WHERE role_id = $1 ORDER BY domain_id`, id)
	if err != nil {
		return nil, err
	}
	defer rows2.Close()
	for rows2.Next() {
		var d uuid.UUID
		if err := rows2.Scan(&d); err != nil {
			return nil, err
		}
		f.DomainIDs = append(f.DomainIDs, d)
	}
	if err := rows2.Err(); err != nil {
		return nil, err
	}

	rows3, err := r.pool.Query(ctx, `SELECT entity_type FROM role_entity_type_bindings WHERE role_id = $1 ORDER BY entity_type`, id)
	if err != nil {
		return nil, err
	}
	defer rows3.Close()
	for rows3.Next() {
		var et string
		if err := rows3.Scan(&et); err != nil {
			return nil, err
		}
		f.EntityTypes = append(f.EntityTypes, et)
	}
	if err := rows3.Err(); err != nil {
		return nil, err
	}

	rows4, err := r.pool.Query(ctx, `SELECT scope_kind, scope_ref FROM role_source_scope_bindings WHERE role_id = $1 ORDER BY scope_kind, scope_ref`, id)
	if err != nil {
		return nil, err
	}
	defer rows4.Close()
	for rows4.Next() {
		var s SourceScopeRef
		if err := rows4.Scan(&s.ScopeKind, &s.ScopeRef); err != nil {
			return nil, err
		}
		f.SourceScopes = append(f.SourceScopes, s)
	}
	if err := rows4.Err(); err != nil {
		return nil, err
	}

	rows5, err := r.pool.Query(ctx, `SELECT scenario_key FROM role_scenario_bindings WHERE role_id = $1 ORDER BY scenario_key`, id)
	if err != nil {
		return nil, err
	}
	defer rows5.Close()
	for rows5.Next() {
		var sk string
		if err := rows5.Scan(&sk); err != nil {
			return nil, err
		}
		f.ScenarioKeys = append(f.ScenarioKeys, sk)
	}
	if err := rows5.Err(); err != nil {
		return nil, err
	}

	rows6, err := r.pool.Query(ctx, `SELECT dashboard_key FROM role_dashboard_bindings WHERE role_id = $1 ORDER BY dashboard_key`, id)
	if err != nil {
		return nil, err
	}
	defer rows6.Close()
	for rows6.Next() {
		var dk string
		if err := rows6.Scan(&dk); err != nil {
			return nil, err
		}
		f.DashboardKeys = append(f.DashboardKeys, dk)
	}
	if err := rows6.Err(); err != nil {
		return nil, err
	}

	rows7, err := r.pool.Query(ctx, `
		SELECT knowledge_job_id, can_run, can_configure, can_review_job_output
		FROM role_job_permissions WHERE role_id = $1 ORDER BY knowledge_job_id`, id)
	if err != nil {
		return nil, err
	}
	defer rows7.Close()
	for rows7.Next() {
		var j JobPermissionRow
		if err := rows7.Scan(&j.JobID, &j.CanRun, &j.CanConfigure, &j.CanReviewJobOutput); err != nil {
			return nil, err
		}
		f.JobPermissions = append(f.JobPermissions, j)
	}
	if err := rows7.Err(); err != nil {
		return nil, err
	}

	err = r.pool.QueryRow(ctx, `
		SELECT COALESCE(can_review_outputs,false), COALESCE(can_approve_outputs,false), COALESCE(can_publish_outputs,false),
		       COALESCE(can_override_policies,false), COALESCE(can_manage_assignments,false)
		FROM role_governance_permissions WHERE role_id = $1`, id,
	).Scan(&f.Governance.CanReviewOutputs, &f.Governance.CanApproveOutputs, &f.Governance.CanPublishOutputs,
		&f.Governance.CanOverridePolicies, &f.Governance.CanManageAssignments)
	if err != nil && err != pgx.ErrNoRows {
		return nil, err
	}
	if err == pgx.ErrNoRows {
		f.Governance = GovernanceRow{}
	}

	return &f, nil
}

func (r *DefinitionRepository) CountAssignments(ctx context.Context, roleID uuid.UUID) (int64, error) {
	var n int64
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM user_role_bindings WHERE role_id = $1 AND (expires_at IS NULL OR expires_at > now())`, roleID,
	).Scan(&n)
	return n, err
}

func (r *DefinitionRepository) IsPrivilegedRole(ctx context.Context, roleID uuid.UUID) (bool, error) {
	var gov bool
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(can_override_policies, false) FROM role_governance_permissions WHERE role_id = $1`, roleID,
	).Scan(&gov)
	if err != nil && err != pgx.ErrNoRows {
		return false, err
	}
	if gov {
		return true, nil
	}
	var n int
	err = r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM role_action_permissions rap
		JOIN action_permissions ap ON ap.id = rap.action_permission_id
		WHERE rap.role_id = $1 AND ap.code IN ('manage_permissions','manage_policies')`, roleID,
	).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *DefinitionRepository) IsActive(ctx context.Context, roleID uuid.UUID) (bool, error) {
	var active bool
	err := r.pool.QueryRow(ctx, `SELECT active FROM roles WHERE id = $1`, roleID).Scan(&active)
	if err != nil {
		return false, err
	}
	return active, nil
}

func (r *DefinitionRepository) createFullTx(ctx context.Context, tx pgx.Tx, in RoleWriteInput, presetKey *string, isPreset, isSystem bool, clonedFrom *uuid.UUID, sourcePresetCode *string) (uuid.UUID, error) {
	id := uuid.New()
	active := true
	if in.Active != nil {
		active = *in.Active
	}
	scopeModel := in.ScopeModel
	if scopeModel == "" {
		scopeModel = "global"
	}
	cat := in.Category
	if cat == "" {
		cat = "domain"
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO roles (id, code, name, description, category, active, scope_model, is_preset, preset_key, is_system, cloned_from_role_id, source_preset_code, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,now(),now())`,
		id, in.Code, in.Name, in.Description, cat, active, scopeModel, isPreset, presetKey, isSystem, clonedFrom, sourcePresetCode)
	if err != nil {
		return uuid.Nil, err
	}

	gov := GovernanceRow{}
	if in.Governance != nil {
		gov = *in.Governance
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO role_governance_permissions (role_id, can_review_outputs, can_approve_outputs, can_publish_outputs, can_override_policies, can_manage_assignments)
		VALUES ($1,$2,$3,$4,$5,$6)`,
		id, gov.CanReviewOutputs, gov.CanApproveOutputs, gov.CanPublishOutputs, gov.CanOverridePolicies, gov.CanManageAssignments)
	if err != nil {
		return uuid.Nil, err
	}

	for _, code := range in.ActionCodes {
		code = strings.TrimSpace(code)
		if code == "" {
			continue
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO role_action_permissions (id, role_id, action_permission_id)
			SELECT gen_random_uuid(), $1, ap.id FROM action_permissions ap WHERE ap.code = $2
			ON CONFLICT (role_id, action_permission_id) DO NOTHING`, id, code)
		if err != nil {
			return uuid.Nil, fmt.Errorf("action %q: %w", code, err)
		}
	}

	for _, did := range in.DomainIDs {
		_, err = tx.Exec(ctx, `INSERT INTO role_domain_bindings (role_id, domain_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`, id, did)
		if err != nil {
			return uuid.Nil, err
		}
	}
	for _, et := range in.EntityTypes {
		et = strings.TrimSpace(et)
		if et == "" {
			continue
		}
		_, err = tx.Exec(ctx, `INSERT INTO role_entity_type_bindings (role_id, entity_type) VALUES ($1,$2) ON CONFLICT DO NOTHING`, id, et)
		if err != nil {
			return uuid.Nil, err
		}
	}
	for _, sc := range in.SourceScopes {
		_, err = tx.Exec(ctx, `INSERT INTO role_source_scope_bindings (role_id, scope_kind, scope_ref) VALUES ($1,$2,$3) ON CONFLICT DO NOTHING`,
			id, sc.ScopeKind, sc.ScopeRef)
		if err != nil {
			return uuid.Nil, err
		}
	}
	for _, sk := range in.ScenarioKeys {
		sk = strings.TrimSpace(sk)
		if sk == "" {
			continue
		}
		_, err = tx.Exec(ctx, `INSERT INTO role_scenario_bindings (role_id, scenario_key) VALUES ($1,$2) ON CONFLICT DO NOTHING`, id, sk)
		if err != nil {
			return uuid.Nil, err
		}
	}
	for _, dk := range in.DashboardKeys {
		dk = strings.TrimSpace(dk)
		if dk == "" {
			continue
		}
		_, err = tx.Exec(ctx, `INSERT INTO role_dashboard_bindings (role_id, dashboard_key) VALUES ($1,$2) ON CONFLICT DO NOTHING`, id, dk)
		if err != nil {
			return uuid.Nil, err
		}
	}
	for _, jp := range in.JobPermissions {
		_, err = tx.Exec(ctx, `
			INSERT INTO role_job_permissions (role_id, knowledge_job_id, can_run, can_configure, can_review_job_output)
			VALUES ($1,$2,$3,$4,$5) ON CONFLICT (role_id, knowledge_job_id) DO UPDATE SET
				can_run = EXCLUDED.can_run, can_configure = EXCLUDED.can_configure, can_review_job_output = EXCLUDED.can_review_job_output`,
			id, jp.JobID, jp.CanRun, jp.CanConfigure, jp.CanReviewJobOutput)
		if err != nil {
			return uuid.Nil, err
		}
	}

	return id, nil
}

func (r *DefinitionRepository) Create(ctx context.Context, in RoleWriteInput) (uuid.UUID, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback(ctx)

	id, err := r.createFullTx(ctx, tx, in, nil, false, false, nil, nil)
	if err != nil {
		return uuid.Nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

func (r *DefinitionRepository) replaceBindings(ctx context.Context, tx pgx.Tx, roleID uuid.UUID, in RoleWriteInput) error {
	_, err := tx.Exec(ctx, `DELETE FROM role_domain_bindings WHERE role_id = $1`, roleID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `DELETE FROM role_entity_type_bindings WHERE role_id = $1`, roleID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `DELETE FROM role_source_scope_bindings WHERE role_id = $1`, roleID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `DELETE FROM role_scenario_bindings WHERE role_id = $1`, roleID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `DELETE FROM role_dashboard_bindings WHERE role_id = $1`, roleID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `DELETE FROM role_job_permissions WHERE role_id = $1`, roleID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `DELETE FROM role_action_permissions WHERE role_id = $1`, roleID)
	if err != nil {
		return err
	}

	for _, code := range in.ActionCodes {
		code = strings.TrimSpace(code)
		if code == "" {
			continue
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO role_action_permissions (id, role_id, action_permission_id)
			SELECT gen_random_uuid(), $1, ap.id FROM action_permissions ap WHERE ap.code = $2
			ON CONFLICT (role_id, action_permission_id) DO NOTHING`, roleID, code)
		if err != nil {
			return fmt.Errorf("action %q: %w", code, err)
		}
	}

	for _, did := range in.DomainIDs {
		_, err = tx.Exec(ctx, `INSERT INTO role_domain_bindings (role_id, domain_id) VALUES ($1,$2)`, roleID, did)
		if err != nil {
			return err
		}
	}
	for _, et := range in.EntityTypes {
		et = strings.TrimSpace(et)
		if et == "" {
			continue
		}
		_, err = tx.Exec(ctx, `INSERT INTO role_entity_type_bindings (role_id, entity_type) VALUES ($1,$2)`, roleID, et)
		if err != nil {
			return err
		}
	}
	for _, sc := range in.SourceScopes {
		_, err = tx.Exec(ctx, `INSERT INTO role_source_scope_bindings (role_id, scope_kind, scope_ref) VALUES ($1,$2,$3)`, roleID, sc.ScopeKind, sc.ScopeRef)
		if err != nil {
			return err
		}
	}
	for _, sk := range in.ScenarioKeys {
		sk = strings.TrimSpace(sk)
		if sk == "" {
			continue
		}
		_, err = tx.Exec(ctx, `INSERT INTO role_scenario_bindings (role_id, scenario_key) VALUES ($1,$2)`, roleID, sk)
		if err != nil {
			return err
		}
	}
	for _, dk := range in.DashboardKeys {
		dk = strings.TrimSpace(dk)
		if dk == "" {
			continue
		}
		_, err = tx.Exec(ctx, `INSERT INTO role_dashboard_bindings (role_id, dashboard_key) VALUES ($1,$2)`, roleID, dk)
		if err != nil {
			return err
		}
	}
	for _, jp := range in.JobPermissions {
		_, err = tx.Exec(ctx, `
			INSERT INTO role_job_permissions (role_id, knowledge_job_id, can_run, can_configure, can_review_job_output)
			VALUES ($1,$2,$3,$4,$5)`,
			roleID, jp.JobID, jp.CanRun, jp.CanConfigure, jp.CanReviewJobOutput)
		if err != nil {
			return err
		}
	}

	gov := GovernanceRow{}
	if in.Governance != nil {
		gov = *in.Governance
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO role_governance_permissions (role_id, can_review_outputs, can_approve_outputs, can_publish_outputs, can_override_policies, can_manage_assignments)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (role_id) DO UPDATE SET
			can_review_outputs = EXCLUDED.can_review_outputs,
			can_approve_outputs = EXCLUDED.can_approve_outputs,
			can_publish_outputs = EXCLUDED.can_publish_outputs,
			can_override_policies = EXCLUDED.can_override_policies,
			can_manage_assignments = EXCLUDED.can_manage_assignments`,
		roleID, gov.CanReviewOutputs, gov.CanApproveOutputs, gov.CanPublishOutputs, gov.CanOverridePolicies, gov.CanManageAssignments)
	return err
}

// PatchMeta updates scalar columns; if replaceBindings is non-nil, replaces all bindings.
func (r *DefinitionRepository) Patch(ctx context.Context, roleID uuid.UUID, name, code *string, description *string, category *string, active *bool, scopeModel *string, replaceBindings *RoleWriteInput) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var curName, curCode, curCat, curScope string
	var curDesc *string
	var curActive bool
	err = tx.QueryRow(ctx, `SELECT name, code, description, category, active, scope_model FROM roles WHERE id = $1`, roleID).
		Scan(&curName, &curCode, &curDesc, &curCat, &curActive, &curScope)
	if err != nil {
		return err
	}

	nm := curName
	if name != nil {
		nm = *name
	}
	cd := curCode
	if code != nil {
		cd = *code
	}
	desc := curDesc
	if description != nil {
		desc = description
	}
	cat := curCat
	if category != nil {
		cat = *category
	}
	act := curActive
	if active != nil {
		act = *active
	}
	sm := curScope
	if scopeModel != nil {
		sm = *scopeModel
	}

	_, err = tx.Exec(ctx, `
		UPDATE roles SET name = $2, code = $3, description = $4, category = $5, active = $6, scope_model = $7, updated_at = now() WHERE id = $1`,
		roleID, nm, cd, desc, cat, act, sm)
	if err != nil {
		return err
	}

	if replaceBindings != nil {
		if err := r.replaceBindings(ctx, tx, roleID, *replaceBindings); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *DefinitionRepository) SetInactive(ctx context.Context, roleID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE roles SET active = false, updated_at = now() WHERE id = $1`, roleID)
	return err
}

func (r *DefinitionRepository) DeleteHard(ctx context.Context, roleID uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `DELETE FROM roles WHERE id = $1`, roleID)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *DefinitionRepository) Clone(ctx context.Context, sourceID uuid.UUID, newCode, newName string, description *string, category string, sourcePresetCode *string) (uuid.UUID, error) {
	src, err := r.GetFull(ctx, sourceID)
	if err != nil {
		return uuid.Nil, err
	}

	in := RoleWriteInput{
		Code:          newCode,
		Name:          newName,
		Description:   description,
		Category:      category,
		Active:        boolPtr(true),
		ScopeModel:    src.ScopeModel,
		ActionCodes:   append([]string(nil), src.ActionCodes...),
		DomainIDs:     append([]uuid.UUID(nil), src.DomainIDs...),
		EntityTypes:   append([]string(nil), src.EntityTypes...),
		SourceScopes:  append([]SourceScopeRef(nil), src.SourceScopes...),
		ScenarioKeys:  append([]string(nil), src.ScenarioKeys...),
		DashboardKeys: append([]string(nil), src.DashboardKeys...),
		Governance:    &GovernanceRow{src.Governance.CanReviewOutputs, src.Governance.CanApproveOutputs, src.Governance.CanPublishOutputs, src.Governance.CanOverridePolicies, src.Governance.CanManageAssignments},
	}
	for _, j := range src.JobPermissions {
		in.JobPermissions = append(in.JobPermissions, JobPermissionWrite{
			JobID: j.JobID, CanRun: j.CanRun, CanConfigure: j.CanConfigure, CanReviewJobOutput: j.CanReviewJobOutput,
		})
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback(ctx)

	id, err := r.createFullTx(ctx, tx, in, nil, false, false, &sourceID, sourcePresetCode)
	if err != nil {
		return uuid.Nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

func boolPtr(b bool) *bool { return &b }
