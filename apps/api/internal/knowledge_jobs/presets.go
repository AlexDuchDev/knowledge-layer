package knowledge_jobs

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// JobBuilderPresetRow is a persisted builder preset (seeded + future org presets).
type JobBuilderPresetRow struct {
	PresetKey    string          `json:"preset_key"`
	Name         string          `json:"name"`
	Description  *string         `json:"description,omitempty"`
	TemplateKey  string          `json:"template_key"`
	DefaultsJSON json.RawMessage `json:"defaults_json"`
	IsSystem     bool            `json:"is_system"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// JobBuilderPresetMerged combines code template metadata with DB preset defaults.
type JobBuilderPresetMerged struct {
	PresetKey            string             `json:"preset_key"`
	Name                 string             `json:"name"`
	Description          *string            `json:"description,omitempty"`
	TemplateKey          string             `json:"template_key"`
	Template             *JobTemplatePublic `json:"template,omitempty"`
	ProcessorImplemented bool               `json:"processor_implemented"`
	DefaultsJSON         json.RawMessage    `json:"defaults_json"`
	IsSystem             bool               `json:"is_system"`
}

// ListJobBuilderPresetsFromDB loads preset rows (fails soft if table missing — caller should migrate).
func ListJobBuilderPresetsFromDB(ctx context.Context, pool *pgxpool.Pool) ([]JobBuilderPresetRow, error) {
	rows, err := pool.Query(ctx, `
		SELECT preset_key, name, description, template_key, defaults_json, is_system, created_at, updated_at
		FROM job_builder_presets ORDER BY preset_key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []JobBuilderPresetRow
	for rows.Next() {
		var r JobBuilderPresetRow
		if err := rows.Scan(&r.PresetKey, &r.Name, &r.Description, &r.TemplateKey, &r.DefaultsJSON, &r.IsSystem, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ListBuilderPresetsMerged returns DB-seeded presets merged with the code template catalog.
func (s *JobService) ListBuilderPresetsMerged(ctx context.Context) ([]JobBuilderPresetMerged, error) {
	rows, err := ListJobBuilderPresetsFromDB(ctx, s.pool)
	if err != nil {
		return nil, err
	}
	return MergeJobBuilderPresetsForAPI(rows), nil
}

// MergeJobBuilderPresetsForAPI joins DB presets with in-code template catalog.
func MergeJobBuilderPresetsForAPI(rows []JobBuilderPresetRow) []JobBuilderPresetMerged {
	code := ListJobTemplatesPublic()
	idx := make(map[string]JobTemplatePublic, len(code))
	for _, t := range code {
		idx[t.ID] = t
	}
	out := make([]JobBuilderPresetMerged, 0, len(rows))
	for _, r := range rows {
		m := JobBuilderPresetMerged{
			PresetKey:    r.PresetKey,
			Name:         r.Name,
			Description:  r.Description,
			TemplateKey:  r.TemplateKey,
			DefaultsJSON: r.DefaultsJSON,
			IsSystem:     r.IsSystem,
		}
		if t, ok := idx[r.TemplateKey]; ok {
			tCopy := t
			m.Template = &tCopy
			m.ProcessorImplemented = IsKnowledgeJobProcessorImplemented(t.JobType)
		}
		out = append(out, m)
	}
	return out
}
