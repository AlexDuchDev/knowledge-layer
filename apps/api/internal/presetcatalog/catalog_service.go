package presetcatalog

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/knowledgelayer/api/internal/knowledge_jobs"
	"github.com/knowledgelayer/api/internal/role_builder"
)

// CatalogService lists presets and builds previews.
type CatalogService struct {
	repo *CatalogRepository
	rb   *role_builder.Services
}

func NewCatalogService(repo *CatalogRepository, rb *role_builder.Services) *CatalogService {
	return &CatalogService{repo: repo, rb: rb}
}

// List delegates to repository.
func (s *CatalogService) List(ctx context.Context, presetType, categoryAxis, categoryCode string) ([]ListRow, error) {
	return s.repo.List(ctx, presetType, categoryAxis, categoryCode)
}

// Detail bundles entry, categories, and structured preview JSON.
type Detail struct {
	Entry      CatalogEntry  `json:"entry"`
	Categories []CategoryRef `json:"categories"`
	Preview    any           `json:"preview"`
}

// GetDetail returns catalog row plus preview for builders.
func (s *CatalogService) GetDetail(ctx context.Context, id uuid.UUID) (*Detail, error) {
	entry, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	cats, err := s.repo.categoriesForEntry(ctx, id)
	if err != nil {
		return nil, err
	}
	preview, err := s.buildPreview(ctx, entry)
	if err != nil {
		return nil, err
	}
	return &Detail{Entry: *entry, Categories: cats, Preview: preview}, nil
}

func (s *CatalogService) buildPreview(ctx context.Context, entry *CatalogEntry) (any, error) {
	switch entry.PresetType {
	case "role":
		rid, err := s.repo.RoleTemplateID(ctx, entry.Code)
		if err != nil {
			return map[string]string{"error": "template role not found"}, nil
		}
		full, err := s.rb.Definitions.Get(ctx, rid)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"kind":            "role",
			"scope_model":     full.ScopeModel,
			"action_codes":    full.ActionCodes,
			"allowed_domains": full.DomainIDs,
			"entity_types":    full.EntityTypes,
			"governance":      full.Governance,
			"scenario_keys":   full.ScenarioKeys,
			"job_permissions": full.JobPermissions,
		}, nil
	case "scenario":
		name, desc, st, tpl, err := s.repo.ScenarioPresetTemplate(ctx, entry.Code)
		if err != nil {
			return map[string]string{"error": "scenario preset template not found"}, nil
		}
		return map[string]any{
			"kind":          "scenario",
			"name":          name,
			"description":   desc,
			"scenario_type": st,
			"template_json": json.RawMessage(tpl),
		}, nil
	case "job":
		name, desc, tk, defs, err := s.repo.JobBuilderPresetRow(ctx, entry.Code)
		if err != nil {
			return map[string]string{"error": "job builder preset not found"}, nil
		}
		var tpl *knowledge_jobs.JobTemplatePublic
		for _, t := range knowledge_jobs.ListJobTemplatesPublic() {
			if t.ID == tk {
				c := t
				tpl = &c
				break
			}
		}
		return map[string]any{
			"kind":            "job",
			"name":            name,
			"description":     desc,
			"template_key":    tk,
			"defaults_json":   json.RawMessage(defs),
			"template_public": tpl,
		}, nil
	default:
		return nil, fmt.Errorf("unknown preset_type %q", entry.PresetType)
	}
}
