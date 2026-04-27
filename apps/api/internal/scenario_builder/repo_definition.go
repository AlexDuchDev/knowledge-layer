package scenario_builder

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DefinitionRepository persists scenario_definitions and output policies.
type DefinitionRepository struct {
	pool *pgxpool.Pool
}

func NewDefinitionRepository(pool *pgxpool.Pool) *DefinitionRepository {
	return &DefinitionRepository{pool: pool}
}

func emptyJSON() json.RawMessage {
	return json.RawMessage([]byte("{}"))
}

func normJSON(b json.RawMessage) json.RawMessage {
	if len(b) == 0 || string(b) == "null" {
		return emptyJSON()
	}
	return b
}

// ListSummaries returns scenarios with visible role codes aggregated.
func (r *DefinitionRepository) ListSummaries(ctx context.Context) ([]ScenarioSummary, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT s.id, s.code, s.name, s.description, s.scenario_type, s.active, s.is_preset, s.preset_key,
		       s.created_at, s.updated_at,
		       COALESCE(array_agg(DISTINCT ro.code) FILTER (WHERE srb.can_see = true), ARRAY[]::text[]) AS role_codes
		FROM scenario_definitions s
		LEFT JOIN scenario_role_bindings srb ON s.id = srb.scenario_id
		LEFT JOIN roles ro ON ro.id = srb.role_id
		GROUP BY s.id
		ORDER BY s.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ScenarioSummary
	for rows.Next() {
		var s ScenarioSummary
		var desc *string
		var pk *string
		var codes []string
		if err := rows.Scan(&s.ID, &s.Code, &s.Name, &desc, &s.ScenarioType, &s.Active, &s.IsPreset, &pk,
			&s.CreatedAt, &s.UpdatedAt, &codes); err != nil {
			return nil, err
		}
		s.Description = desc
		s.PresetKey = pk
		s.VisibleRoleCodes = codes
		out = append(out, s)
	}
	return out, rows.Err()
}

// GetFull loads definition, policy, and all bindings.
func (r *DefinitionRepository) GetFull(ctx context.Context, id uuid.UUID) (*ScenarioFull, error) {
	var s ScenarioFull
	var desc, notes, pk, spc *string
	var ownerUser, ownerTeam, clonedFrom, outPolID *uuid.UUID
	var tgtScope, inScope, trigCfg, cfg, prevCfg json.RawMessage
	var trigType, proc, outMode, uiSurf string
	err := r.pool.QueryRow(ctx, `
		SELECT id, code, name, description, scenario_type, active, is_preset, preset_key,
		       created_at, updated_at,
		       target_role_scope_json, input_scope_json, trigger_type, trigger_config_json,
		       processing_mode, output_mode, ui_surface, config_json, preview_config, notes,
		       owner_user_id, owner_team_id, cloned_from_scenario_id, source_preset_code, output_policy_id
		FROM scenario_definitions WHERE id = $1`, id,
	).Scan(&s.ID, &s.Code, &s.Name, &desc, &s.ScenarioType, &s.Active, &s.IsPreset, &pk,
		&s.CreatedAt, &s.UpdatedAt,
		&tgtScope, &inScope, &trigType, &trigCfg, &proc, &outMode, &uiSurf, &cfg, &prevCfg, &notes,
		&ownerUser, &ownerTeam, &clonedFrom, &spc, &outPolID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		return nil, err
	}
	s.Description = desc
	s.PresetKey = pk
	s.Notes = notes
	s.OwnerUserID = ownerUser
	s.OwnerTeamID = ownerTeam
	s.ClonedFromScenarioID = clonedFrom
	s.SourcePresetCode = spc
	s.OutputPolicyID = outPolID
	s.TargetRoleScopeJSON = normJSON(tgtScope)
	s.InputScopeJSON = normJSON(inScope)
	s.TriggerType = trigType
	s.TriggerConfigJSON = normJSON(trigCfg)
	s.ProcessingMode = proc
	s.OutputMode = outMode
	s.UISurface = uiSurf
	s.ConfigJSON = normJSON(cfg)
	s.PreviewConfig = normJSON(prevCfg)

	if outPolID != nil {
		pol, err := r.getOutputPolicyByScenarioID(ctx, id)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		if pol != nil {
			s.OutputPolicy = pol
		}
	}

	s.RoleBindings, err = r.listRoleBindings(ctx, id)
	if err != nil {
		return nil, err
	}
	s.SourceBindings, err = r.listSourceBindings(ctx, id)
	if err != nil {
		return nil, err
	}
	s.JobBindings, err = r.listJobBindings(ctx, id)
	if err != nil {
		return nil, err
	}
	s.UIBindings, err = r.listUIBindings(ctx, id)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *DefinitionRepository) getOutputPolicyByScenarioID(ctx context.Context, scenarioID uuid.UUID) (*OutputPolicyRow, error) {
	var p OutputPolicyRow
	var extra json.RawMessage
	var dom *uuid.UUID
	err := r.pool.QueryRow(ctx, `
		SELECT id, scenario_id, output_domain_id, output_sensitivity_level, review_required, publication_mode,
		       citations_required, provenance_required, extra_json
		FROM scenario_output_policies WHERE scenario_id = $1`, scenarioID,
	).Scan(&p.ID, &p.ScenarioID, &dom, &p.OutputSensitivity, &p.ReviewRequired, &p.PublicationMode,
		&p.CitationsRequired, &p.ProvenanceRequired, &extra)
	if err != nil {
		return nil, err
	}
	p.OutputDomainID = dom
	p.ExtraJSON = normJSON(extra)
	return &p, nil
}

func (r *DefinitionRepository) listRoleBindings(ctx context.Context, scenarioID uuid.UUID) ([]RoleBindingRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT srb.role_id, r.code, srb.can_see, srb.can_run, srb.can_manage, srb.can_review_publish
		FROM scenario_role_bindings srb
		JOIN roles r ON r.id = srb.role_id
		WHERE srb.scenario_id = $1
		ORDER BY r.code`, scenarioID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []RoleBindingRow
	for rows.Next() {
		var b RoleBindingRow
		if err := rows.Scan(&b.RoleID, &b.RoleCode, &b.CanSee, &b.CanRun, &b.CanManage, &b.CanReviewPublish); err != nil {
			return nil, err
		}
		list = append(list, b)
	}
	return list, rows.Err()
}

func (r *DefinitionRepository) listSourceBindings(ctx context.Context, scenarioID uuid.UUID) ([]SourceBindingRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT source_feed_id, binding_role FROM scenario_source_bindings WHERE scenario_id = $1`, scenarioID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []SourceBindingRow
	for rows.Next() {
		var b SourceBindingRow
		if err := rows.Scan(&b.SourceFeedID, &b.BindingRole); err != nil {
			return nil, err
		}
		list = append(list, b)
	}
	return list, rows.Err()
}

func (r *DefinitionRepository) listJobBindings(ctx context.Context, scenarioID uuid.UUID) ([]JobBindingRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT sjb.knowledge_job_id, COALESCE(kj.name, ''), sjb.relationship
		FROM scenario_job_bindings sjb
		LEFT JOIN knowledge_jobs kj ON kj.id = sjb.knowledge_job_id
		WHERE sjb.scenario_id = $1`, scenarioID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []JobBindingRow
	for rows.Next() {
		var b JobBindingRow
		if err := rows.Scan(&b.KnowledgeJobID, &b.JobName, &b.Relationship); err != nil {
			return nil, err
		}
		list = append(list, b)
	}
	return list, rows.Err()
}

func (r *DefinitionRepository) listUIBindings(ctx context.Context, scenarioID uuid.UUID) ([]UIBindingRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, surface_key, nav_group, sort_order, config_json
		FROM scenario_ui_bindings WHERE scenario_id = $1 ORDER BY sort_order, surface_key`, scenarioID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []UIBindingRow
	for rows.Next() {
		var b UIBindingRow
		var nav *string
		var cfg json.RawMessage
		if err := rows.Scan(&b.ID, &b.SurfaceKey, &nav, &b.SortOrder, &cfg); err != nil {
			return nil, err
		}
		b.NavGroup = nav
		b.ConfigJSON = normJSON(cfg)
		list = append(list, b)
	}
	return list, rows.Err()
}

// Create inserts definition + optional policy in one transaction.
func (r *DefinitionRepository) Create(ctx context.Context, in ScenarioWriteInput) (uuid.UUID, error) {
	if err := ValidateWriteInput(in, true); err != nil {
		return uuid.Nil, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	id := uuid.New()
	now := time.Now().UTC()
	active := true
	if in.Active != nil {
		active = *in.Active
	}
	isPreset := in.IsPreset
	var pk *string
	if in.PresetKey != nil && strings.TrimSpace(*in.PresetKey) != "" {
		pk = in.PresetKey
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO scenario_definitions (
			id, code, name, description, scenario_type, active,
			target_role_scope_json, input_scope_json, trigger_type, trigger_config_json,
			processing_mode, output_mode, ui_surface, config_json, preview_config, notes,
			owner_user_id, owner_team_id, is_preset, preset_key, cloned_from_scenario_id, source_preset_code,
			created_at, updated_at, output_policy_id
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,NULL
		)`,
		id, strings.TrimSpace(in.Code), strings.TrimSpace(in.Name), in.Description, strings.TrimSpace(in.ScenarioType), active,
		normJSON(in.TargetRoleScopeJSON), normJSON(in.InputScopeJSON), strings.TrimSpace(in.TriggerType), normJSON(in.TriggerConfigJSON),
		strings.TrimSpace(in.ProcessingMode), strings.TrimSpace(in.OutputMode), strings.TrimSpace(in.UISurface),
		normJSON(in.ConfigJSON), normJSON(in.PreviewConfig), in.Notes,
		in.OwnerUserID, in.OwnerTeamID, isPreset, pk, in.ClonedFromScenarioID, in.SourcePresetCode,
		now, now)
	if err != nil {
		return uuid.Nil, err
	}

	if in.OutputPolicy != nil {
		polID := uuid.New()
		pub := in.OutputPolicy.PublicationMode
		if strings.TrimSpace(pub) == "" {
			pub = "draft"
		}
		ex := normJSON(in.OutputPolicy.ExtraJSON)
		_, err = tx.Exec(ctx, `
			INSERT INTO scenario_output_policies (
				id, scenario_id, output_domain_id, output_sensitivity_level, review_required, publication_mode,
				citations_required, provenance_required, extra_json, created_at, updated_at
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
			polID, id, in.OutputPolicy.OutputDomainID, in.OutputPolicy.OutputSensitivity, in.OutputPolicy.ReviewRequired, pub,
			in.OutputPolicy.CitationsRequired, in.OutputPolicy.ProvenanceRequired, ex, now, now)
		if err != nil {
			return uuid.Nil, err
		}
		_, err = tx.Exec(ctx, `UPDATE scenario_definitions SET output_policy_id = $1, updated_at = $2 WHERE id = $3`, polID, now, id)
		if err != nil {
			return uuid.Nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

// Patch updates scalar fields; optional OutputPolicy replaces policy row.
func (r *DefinitionRepository) Patch(ctx context.Context, id uuid.UUID,
	name, code *string,
	description *string,
	scenarioType *string,
	active *bool,
	targetScope, inputScope, triggerCfg *json.RawMessage,
	triggerType *string,
	procMode, outMode, uiSurface *string,
	configJSON, previewConfig *json.RawMessage,
	notes *string,
	ownerUser, ownerTeam *uuid.UUID,
	policy *OutputPolicyWrite,
) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var cur ScenarioWriteInput
	var curDesc, curNotes *string
	var curActive bool
	var tgt, inSc, trCfg, cfg, pCfg json.RawMessage
	var trT, proc, outM, uiS string
	err = tx.QueryRow(ctx, `
		SELECT code, name, description, scenario_type, active, target_role_scope_json, input_scope_json,
		       trigger_type, trigger_config_json, processing_mode, output_mode, ui_surface, config_json, preview_config, notes
		FROM scenario_definitions WHERE id = $1`, id,
	).Scan(&cur.Code, &cur.Name, &curDesc, &cur.ScenarioType, &curActive, &tgt, &inSc, &trT, &trCfg, &proc, &outM, &uiS, &cfg, &pCfg, &curNotes)
	if err != nil {
		return err
	}
	cur.Description = curDesc
	cur.Name = strings.TrimSpace(cur.Name)
	cur.TargetRoleScopeJSON = tgt
	cur.InputScopeJSON = inSc
	cur.TriggerType = trT
	cur.TriggerConfigJSON = trCfg
	cur.ProcessingMode = proc
	cur.OutputMode = outM
	cur.UISurface = uiS
	cur.ConfigJSON = cfg
	cur.PreviewConfig = pCfg
	cur.Notes = curNotes

	if name != nil {
		cur.Name = strings.TrimSpace(*name)
	}
	if code != nil {
		cur.Code = strings.TrimSpace(*code)
	}
	if description != nil {
		cur.Description = description
	}
	if scenarioType != nil {
		cur.ScenarioType = strings.TrimSpace(*scenarioType)
	}
	if active != nil {
		curActive = *active
	}
	if targetScope != nil {
		cur.TargetRoleScopeJSON = normJSON(*targetScope)
	}
	if inputScope != nil {
		cur.InputScopeJSON = normJSON(*inputScope)
	}
	if triggerType != nil {
		cur.TriggerType = strings.TrimSpace(*triggerType)
	}
	if triggerCfg != nil {
		cur.TriggerConfigJSON = normJSON(*triggerCfg)
	}
	if procMode != nil {
		cur.ProcessingMode = strings.TrimSpace(*procMode)
	}
	if outMode != nil {
		cur.OutputMode = strings.TrimSpace(*outMode)
	}
	if uiSurface != nil {
		cur.UISurface = strings.TrimSpace(*uiSurface)
	}
	if configJSON != nil {
		cur.ConfigJSON = normJSON(*configJSON)
	}
	if previewConfig != nil {
		cur.PreviewConfig = normJSON(*previewConfig)
	}
	if notes != nil {
		cur.Notes = notes
	}

	// Load existing policy for validation merge
	var existingPol *OutputPolicyWrite
	var polID *uuid.UUID
	_ = tx.QueryRow(ctx, `SELECT output_policy_id FROM scenario_definitions WHERE id = $1`, id).Scan(&polID)
	if polID != nil {
		var p OutputPolicyWrite
		var ex json.RawMessage
		var dom *uuid.UUID
		err = tx.QueryRow(ctx, `
			SELECT output_domain_id, output_sensitivity_level, review_required, publication_mode,
			       citations_required, provenance_required, extra_json
			FROM scenario_output_policies WHERE id = $1`, *polID,
		).Scan(&dom, &p.OutputSensitivity, &p.ReviewRequired, &p.PublicationMode, &p.CitationsRequired, &p.ProvenanceRequired, &ex)
		if err == nil {
			p.OutputDomainID = dom
			p.ExtraJSON = normJSON(ex)
			existingPol = &p
		}
	}
	mergedPol := existingPol
	if policy != nil {
		mergedPol = policy
	}
	if mergedPol == nil && requiresOutputPolicy(cur.ScenarioType) {
		return ErrMissingOutputPolicy
	}
	if err := ValidateWriteInput(ScenarioWriteInput{
		Code: cur.Code, Name: cur.Name, ScenarioType: cur.ScenarioType,
		OutputMode: cur.OutputMode, InputScopeJSON: cur.InputScopeJSON, ConfigJSON: cur.ConfigJSON,
		TriggerType: cur.TriggerType, TriggerConfigJSON: cur.TriggerConfigJSON,
		OutputPolicy: mergedPol,
	}, false); err != nil {
		return err
	}

	now := time.Now().UTC()
	_, err = tx.Exec(ctx, `
		UPDATE scenario_definitions SET
			code = $2, name = $3, description = $4, scenario_type = $5, active = $6,
			target_role_scope_json = $7, input_scope_json = $8, trigger_type = $9, trigger_config_json = $10,
			processing_mode = $11, output_mode = $12, ui_surface = $13, config_json = $14, preview_config = $15,
			notes = $16, owner_user_id = COALESCE($17, owner_user_id), owner_team_id = COALESCE($18, owner_team_id),
			updated_at = $19
		WHERE id = $1`,
		id, cur.Code, cur.Name, cur.Description, cur.ScenarioType, curActive,
		normJSON(cur.TargetRoleScopeJSON), normJSON(cur.InputScopeJSON), cur.TriggerType, normJSON(cur.TriggerConfigJSON),
		cur.ProcessingMode, cur.OutputMode, cur.UISurface, normJSON(cur.ConfigJSON), normJSON(cur.PreviewConfig),
		cur.Notes, ownerUser, ownerTeam, now)
	if err != nil {
		return err
	}

	if policy != nil {
		pub := policy.PublicationMode
		if strings.TrimSpace(pub) == "" {
			pub = "draft"
		}
		ex := normJSON(policy.ExtraJSON)
		if polID == nil {
			newID := uuid.New()
			_, err = tx.Exec(ctx, `
				INSERT INTO scenario_output_policies (
					id, scenario_id, output_domain_id, output_sensitivity_level, review_required, publication_mode,
					citations_required, provenance_required, extra_json, created_at, updated_at
				) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
				newID, id, policy.OutputDomainID, policy.OutputSensitivity, policy.ReviewRequired, pub,
				policy.CitationsRequired, policy.ProvenanceRequired, ex, now, now)
			if err != nil {
				return err
			}
			_, err = tx.Exec(ctx, `UPDATE scenario_definitions SET output_policy_id = $1, updated_at = $2 WHERE id = $3`, newID, now, id)
		} else {
			_, err = tx.Exec(ctx, `
				UPDATE scenario_output_policies SET
					output_domain_id = $2, output_sensitivity_level = $3, review_required = $4, publication_mode = $5,
					citations_required = $6, provenance_required = $7, extra_json = $8, updated_at = $9
				WHERE id = $1`,
				*polID, policy.OutputDomainID, policy.OutputSensitivity, policy.ReviewRequired, pub,
				policy.CitationsRequired, policy.ProvenanceRequired, ex, now)
		}
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// Delete removes scenario (cascades to policies/bindings).
func (r *DefinitionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM scenario_definitions WHERE id = $1 AND is_preset = false`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("scenario not found or is system preset")
	}
	return nil
}

// GetCode returns scenario code by id.
func (r *DefinitionRepository) GetCode(ctx context.Context, id uuid.UUID) (string, error) {
	var code string
	err := r.pool.QueryRow(ctx, `SELECT code FROM scenario_definitions WHERE id = $1`, id).Scan(&code)
	return code, err
}

// IsPreset reports if scenario is seeded preset.
func (r *DefinitionRepository) IsPreset(ctx context.Context, id uuid.UUID) (bool, error) {
	var p bool
	err := r.pool.QueryRow(ctx, `SELECT is_preset FROM scenario_definitions WHERE id = $1`, id).Scan(&p)
	return p, err
}

// ListPresetsCatalog reads scenario_presets table.
func (r *DefinitionRepository) ListPresetsCatalog(ctx context.Context) ([]ScenarioPresetCatalogRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT preset_key, name, description, scenario_type, template_json
		FROM scenario_presets ORDER BY preset_key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []ScenarioPresetCatalogRow
	for rows.Next() {
		var row ScenarioPresetCatalogRow
		var desc *string
		var tpl json.RawMessage
		if err := rows.Scan(&row.PresetKey, &row.Name, &desc, &row.ScenarioType, &tpl); err != nil {
			return nil, err
		}
		row.Description = desc
		row.TemplateJSON = normJSON(tpl)
		list = append(list, row)
	}
	return list, rows.Err()
}

// GetPresetCatalogRow loads one preset template.
func (r *DefinitionRepository) GetPresetCatalogRow(ctx context.Context, presetKey string) (*ScenarioPresetCatalogRow, error) {
	var row ScenarioPresetCatalogRow
	var desc *string
	var tpl json.RawMessage
	err := r.pool.QueryRow(ctx, `
		SELECT preset_key, name, description, scenario_type, template_json
		FROM scenario_presets WHERE preset_key = $1`, presetKey,
	).Scan(&row.PresetKey, &row.Name, &desc, &row.ScenarioType, &tpl)
	if err != nil {
		return nil, err
	}
	row.Description = desc
	row.TemplateJSON = normJSON(tpl)
	return &row, nil
}

// GetSystemScenarioIDByPresetKey returns the seeded system scenario row for a catalog preset_key.
func (r *DefinitionRepository) GetSystemScenarioIDByPresetKey(ctx context.Context, presetKey string) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `
		SELECT id FROM scenario_definitions
		WHERE preset_key = $1 AND is_preset = true
		ORDER BY id LIMIT 1`, strings.TrimSpace(presetKey),
	).Scan(&id)
	return id, err
}
