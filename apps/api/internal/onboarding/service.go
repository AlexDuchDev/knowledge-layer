package onboarding

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/knowledgelayer/api/internal/presetcatalog"
)

// Service orchestrates onboarding sessions and launch.
type Service struct {
	repo   *Repository
	preset *presetcatalog.Bundle
}

// NewService constructs the onboarding service.
func NewService(pool *pgxpool.Pool, preset *presetcatalog.Bundle) *Service {
	return &Service{
		repo:   NewRepository(pool),
		preset: preset,
	}
}

// ListTemplates returns setup modes.
func (s *Service) ListTemplates(ctx context.Context) ([]Template, error) {
	return s.repo.ListTemplates(ctx)
}

// CreateSession starts a draft session.
func (s *Service) CreateSession(ctx context.Context, createdBy uuid.UUID) (*SessionView, error) {
	id, err := s.repo.CreateSession(ctx, createdBy)
	if err != nil {
		return nil, err
	}
	return s.repo.LoadSessionView(ctx, id)
}

// ListSessions returns recent sessions created by the principal.
func (s *Service) ListSessions(ctx context.Context, principal uuid.UUID, limit int) ([]SessionSummary, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.repo.ListSessionsByCreator(ctx, principal, limit)
}

func (s *Service) assertOwner(ctx context.Context, sessionID, principal uuid.UUID) error {
	owner, err := s.repo.SessionCreatedBy(ctx, sessionID)
	if err != nil {
		return err
	}
	if owner != principal {
		return ErrForbidden
	}
	return nil
}

// GetSession loads a session aggregate.
func (s *Service) GetSession(ctx context.Context, id, principal uuid.UUID) (*SessionView, error) {
	if err := s.assertOwner(ctx, id, principal); err != nil {
		return nil, err
	}
	return s.repo.LoadSessionView(ctx, id)
}

// PatchSession updates org profile, steps, selections, connectors, assignments.
func (s *Service) PatchSession(ctx context.Context, id, principal uuid.UUID, p SessionPatch) (*SessionView, error) {
	if err := s.assertOwner(ctx, id, principal); err != nil {
		return nil, err
	}
	if _, err := s.repo.LoadSessionView(ctx, id); err != nil {
		return nil, err
	}
	if len(p.OrgProfileJSON) > 0 && string(p.OrgProfileJSON) != "null" {
		if err := s.repo.UpdateSessionMeta(ctx, id, p.OrgProfileJSON, nil); err != nil {
			return nil, err
		}
	}
	for k, v := range p.Steps {
		if err := s.repo.UpsertStep(ctx, id, k, v); err != nil {
			return nil, err
		}
	}
	if p.SelectedPresets != nil {
		if err := s.repo.ReplaceSelectedPresets(ctx, id, p.SelectedPresets); err != nil {
			return nil, err
		}
	}
	for _, c := range p.Connectors {
		if err := s.repo.UpsertConnector(ctx, id, strings.TrimSpace(c.FamilyCode), c.Enabled); err != nil {
			return nil, err
		}
	}
	if p.Assignment != nil {
		cur, _ := s.repo.loadAssignment(ctx, id)
		patch := mergeAssignment(cur, p.Assignment)
		if err := s.repo.ReplaceAssignment(ctx, id, patch); err != nil {
			return nil, err
		}
	}
	return s.repo.LoadSessionView(ctx, id)
}

func mergeAssignment(cur *AssignmentRow, p *AssignmentPatch) *AssignmentPatch {
	out := &AssignmentPatch{}
	if cur != nil {
		out.InitialAdminUserID = cur.InitialAdminUserID
		out.DomainOwnerUserID = cur.DomainOwnerUserID
		out.AssignmentsJSON = cur.AssignmentsJSON
	}
	if p.InitialAdminUserID != nil {
		out.InitialAdminUserID = p.InitialAdminUserID
	}
	if p.DomainOwnerUserID != nil {
		out.DomainOwnerUserID = p.DomainOwnerUserID
	}
	if len(p.AssignmentsJSON) > 0 && string(p.AssignmentsJSON) != "null" {
		out.AssignmentsJSON = p.AssignmentsJSON
	}
	if out.AssignmentsJSON == nil {
		out.AssignmentsJSON = []byte("{}")
	}
	return out
}

// SelectTemplate applies template defaults to session selections.
func (s *Service) SelectTemplate(ctx context.Context, sessionID, principal uuid.UUID, templateCode string) (*SessionView, error) {
	if err := s.assertOwner(ctx, sessionID, principal); err != nil {
		return nil, err
	}
	templateCode = strings.TrimSpace(templateCode)
	if templateCode == "" {
		return nil, fmt.Errorf("template_code required")
	}
	tpl, err := s.repo.GetTemplateByCode(ctx, templateCode)
	if err != nil {
		return nil, fmt.Errorf("unknown template: %w", err)
	}
	var meta struct {
		RoleCodes     []string `json:"role_codes"`
		ScenarioCodes []string `json:"scenario_codes"`
		JobCodes      []string `json:"job_codes"`
		Families      []string `json:"connector_families"`
	}
	_ = json.Unmarshal(tpl.MetadataJSON, &meta)

	var patches []SelectedPresetPatch
	i := 0
	for _, code := range meta.RoleCodes {
		ids, err := s.repo.CatalogIDsByTypeAndCodes(ctx, "role", []string{code})
		if err != nil {
			return nil, err
		}
		if cid, ok := ids[code]; ok {
			patches = append(patches, SelectedPresetPatch{PresetCatalogEntryID: cid, Slot: fmt.Sprintf("role_%d", i)})
			i++
		}
	}
	i = 0
	for _, code := range meta.ScenarioCodes {
		ids, err := s.repo.CatalogIDsByTypeAndCodes(ctx, "scenario", []string{code})
		if err != nil {
			return nil, err
		}
		if cid, ok := ids[code]; ok {
			patches = append(patches, SelectedPresetPatch{PresetCatalogEntryID: cid, Slot: fmt.Sprintf("scenario_%d", i)})
			i++
		}
	}
	i = 0
	for _, code := range meta.JobCodes {
		ids, err := s.repo.CatalogIDsByTypeAndCodes(ctx, "job", []string{code})
		if err != nil {
			return nil, err
		}
		if cid, ok := ids[code]; ok {
			patches = append(patches, SelectedPresetPatch{PresetCatalogEntryID: cid, Slot: fmt.Sprintf("job_%d", i)})
			i++
		}
	}
	if err := s.repo.SetTemplateCode(ctx, sessionID, templateCode); err != nil {
		return nil, err
	}
	if len(patches) > 0 {
		if err := s.repo.ReplaceSelectedPresets(ctx, sessionID, patches); err != nil {
			return nil, err
		}
	}
	for _, fam := range meta.Families {
		fam = strings.TrimSpace(fam)
		if fam == "" {
			continue
		}
		if err := s.repo.UpsertConnector(ctx, sessionID, fam, true); err != nil {
			return nil, err
		}
	}
	return s.repo.LoadSessionView(ctx, sessionID)
}

// PreviewLaunch builds a plan and validation issues.
func (s *Service) PreviewLaunch(ctx context.Context, sessionID, principal uuid.UUID) (*LaunchPreview, error) {
	if err := s.assertOwner(ctx, sessionID, principal); err != nil {
		return nil, err
	}
	sess, err := s.repo.LoadSessionView(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	var issues []string
	if sess.TemplateCode == nil || strings.TrimSpace(*sess.TemplateCode) == "" {
		issues = append(issues, "setup mode (template) not selected")
	}
	if sess.Assignment == nil || sess.Assignment.InitialAdminUserID == nil || *sess.Assignment.InitialAdminUserID == uuid.Nil {
		issues = append(issues, "initial_admin_user_id is required before launch")
	}
	if len(sess.SelectedPresets) == 0 {
		issues = append(issues, "no presets selected")
	}

	prev := &LaunchPreview{
		TemplateCode:     sess.TemplateCode,
		ValidationIssues: issues,
		Assignments:      sess.Assignment,
	}
	for _, p := range sess.SelectedPresets {
		pl := PlannedInstantiate{
			PresetCatalogEntryID: p.PresetCatalogEntryID,
			PresetType:           p.PresetType,
			Code:                 p.PresetCode,
			Name:                 p.PresetCode,
			Slot:                 p.Slot,
		}
		switch p.PresetType {
		case "role":
			prev.PlannedRoles = append(prev.PlannedRoles, pl)
		case "scenario":
			prev.PlannedScenarios = append(prev.PlannedScenarios, pl)
		case "job":
			prev.PlannedJobs = append(prev.PlannedJobs, pl)
		}
	}
	for _, c := range sess.Connectors {
		if c.Enabled {
			prev.ConnectorsEnabled = append(prev.ConnectorsEnabled, c.FamilyCode)
		}
	}
	return prev, nil
}

// Launch instantiates selected presets and records results.
func (s *Service) Launch(ctx context.Context, sessionID, principal uuid.UUID) (*LaunchResult, error) {
	if err := s.assertOwner(ctx, sessionID, principal); err != nil {
		return nil, err
	}
	st, err := s.repo.SessionStatus(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if st == "launched" {
		return nil, ErrAlreadyLaunched
	}
	preview, err := s.PreviewLaunch(ctx, sessionID, principal)
	if err != nil {
		return nil, err
	}
	if len(preview.ValidationIssues) > 0 {
		return nil, fmt.Errorf("validation failed: %s", strings.Join(preview.ValidationIssues, "; "))
	}

	logID, err := s.repo.InsertLaunchLog(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	sess, _ := s.repo.LoadSessionView(ctx, sessionID)
	var created LaunchCreated
	for _, p := range sess.SelectedPresets {
		suffix := strings.ReplaceAll(uuid.New().String(), "-", "")
		if len(suffix) > 12 {
			suffix = suffix[:12]
		}
		name := p.PresetCode + " (setup " + suffix + ")"
		code := p.PresetCode + "_ob_" + suffix
		req := presetcatalog.InstantiateRequest{
			Name: &name,
			Code: &code,
		}
		res, err := s.preset.Instantiation.Instantiate(ctx, p.PresetCatalogEntryID, principal, req)
		if err != nil {
			_ = s.repo.FinishLaunchLog(ctx, logID, "failed", mustJSON(map[string]any{"created": created}), strPtr(err.Error()))
			return nil, fmt.Errorf("instantiate %s %s: %w", p.PresetType, p.PresetCode, err)
		}
		switch res.TargetKind {
		case "role":
			created.RoleIDs = append(created.RoleIDs, res.TargetID)
		case "scenario":
			created.ScenarioIDs = append(created.ScenarioIDs, res.TargetID)
		case "job":
			created.JobIDs = append(created.JobIDs, res.TargetID)
		}
	}

	_ = s.repo.FinishLaunchLog(ctx, logID, "succeeded", mustJSON(map[string]any{"created": created}), nil)
	_ = s.repo.MarkSessionLaunched(ctx, sessionID)

	return &LaunchResult{
		SessionID:   sessionID,
		Status:      "launched",
		Created:     created,
		LaunchLogID: logID,
	}, nil
}

func mustJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

func strPtr(s string) *string { return &s }
