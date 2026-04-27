package scenario_builder

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ScenarioSummary is a list row for GET /scenarios.
type ScenarioSummary struct {
	ID               uuid.UUID `json:"id"`
	Code             string    `json:"code"`
	Name             string    `json:"name"`
	Description      *string   `json:"description,omitempty"`
	ScenarioType     string    `json:"scenario_type"`
	Active           bool      `json:"active"`
	IsPreset         bool      `json:"is_preset"`
	PresetKey        *string   `json:"preset_key,omitempty"`
	VisibleRoleCodes []string  `json:"visible_role_codes"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// OutputPolicyRow is scenario_output_policies.
type OutputPolicyRow struct {
	ID                 uuid.UUID       `json:"id"`
	ScenarioID         uuid.UUID       `json:"scenario_id"`
	OutputDomainID     *uuid.UUID      `json:"output_domain_id,omitempty"`
	OutputSensitivity  int             `json:"output_sensitivity_level"`
	ReviewRequired     bool            `json:"review_required"`
	PublicationMode    string          `json:"publication_mode"`
	CitationsRequired  bool            `json:"citations_required"`
	ProvenanceRequired bool            `json:"provenance_required"`
	ExtraJSON          json.RawMessage `json:"extra_json"`
}

// RoleBindingRow is scenario_role_bindings.
type RoleBindingRow struct {
	RoleID           uuid.UUID `json:"role_id"`
	RoleCode         string    `json:"role_code,omitempty"`
	CanSee           bool      `json:"can_see"`
	CanRun           bool      `json:"can_run"`
	CanManage        bool      `json:"can_manage"`
	CanReviewPublish bool      `json:"can_review_publish"`
}

// SourceBindingRow is scenario_source_bindings.
type SourceBindingRow struct {
	SourceFeedID uuid.UUID `json:"source_feed_id"`
	BindingRole  string    `json:"binding_role"`
}

// JobBindingRow is scenario_job_bindings.
type JobBindingRow struct {
	KnowledgeJobID uuid.UUID `json:"knowledge_job_id"`
	JobName        string    `json:"job_name,omitempty"`
	Relationship   string    `json:"relationship"`
}

// UIBindingRow is scenario_ui_bindings.
type UIBindingRow struct {
	ID         uuid.UUID       `json:"id"`
	SurfaceKey string          `json:"surface_key"`
	NavGroup   *string         `json:"nav_group,omitempty"`
	SortOrder  int             `json:"sort_order"`
	ConfigJSON json.RawMessage `json:"config_json"`
}

// ScenarioFull is complete definition for API detail.
type ScenarioFull struct {
	ScenarioSummary
	TargetRoleScopeJSON  json.RawMessage    `json:"target_role_scope_json"`
	InputScopeJSON       json.RawMessage    `json:"input_scope_json"`
	TriggerType          string             `json:"trigger_type"`
	TriggerConfigJSON    json.RawMessage    `json:"trigger_config_json"`
	ProcessingMode       string             `json:"processing_mode"`
	OutputMode           string             `json:"output_mode"`
	UISurface            string             `json:"ui_surface"`
	ConfigJSON           json.RawMessage    `json:"config_json"`
	PreviewConfig        json.RawMessage    `json:"preview_config"`
	Notes                *string            `json:"notes,omitempty"`
	OwnerUserID          *uuid.UUID         `json:"owner_user_id,omitempty"`
	OwnerTeamID          *uuid.UUID         `json:"owner_team_id,omitempty"`
	ClonedFromScenarioID *uuid.UUID         `json:"cloned_from_scenario_id,omitempty"`
	SourcePresetCode     *string            `json:"source_preset_code,omitempty"`
	OutputPolicyID       *uuid.UUID         `json:"output_policy_id,omitempty"`
	OutputPolicy         *OutputPolicyRow   `json:"output_policy,omitempty"`
	RoleBindings         []RoleBindingRow   `json:"role_bindings"`
	SourceBindings       []SourceBindingRow `json:"source_bindings"`
	JobBindings          []JobBindingRow    `json:"job_bindings"`
	UIBindings           []UIBindingRow     `json:"ui_bindings"`
}

// ScenarioWriteInput is create payload (and patch when replacing sections).
type ScenarioWriteInput struct {
	Code                 string
	Name                 string
	Description          *string
	ScenarioType         string
	Active               *bool
	TargetRoleScopeJSON  json.RawMessage
	InputScopeJSON       json.RawMessage
	TriggerType          string
	TriggerConfigJSON    json.RawMessage
	ProcessingMode       string
	OutputMode           string
	UISurface            string
	ConfigJSON           json.RawMessage
	PreviewConfig        json.RawMessage
	Notes                *string
	OwnerUserID          *uuid.UUID
	OwnerTeamID          *uuid.UUID
	OutputPolicy         *OutputPolicyWrite
	IsPreset             bool
	PresetKey            *string
	ClonedFromScenarioID *uuid.UUID
	SourcePresetCode     *string
}

// OutputPolicyWrite is upsert payload for output policy.
type OutputPolicyWrite struct {
	OutputDomainID     *uuid.UUID
	OutputSensitivity  int
	ReviewRequired     bool
	PublicationMode    string
	CitationsRequired  bool
	ProvenanceRequired bool
	ExtraJSON          json.RawMessage
}

// ScenarioPresetCatalogRow is GET /scenarios/presets.
type ScenarioPresetCatalogRow struct {
	PresetKey    string          `json:"preset_key"`
	Name         string          `json:"name"`
	Description  *string         `json:"description,omitempty"`
	ScenarioType string          `json:"scenario_type"`
	TemplateJSON json.RawMessage `json:"template_json"`
}

// ScenarioPreview is GET /scenarios/:id/preview.
type ScenarioPreview struct {
	ScenarioID        uuid.UUID          `json:"scenario_id"`
	Code              string             `json:"code"`
	Name              string             `json:"name"`
	ScenarioType      string             `json:"scenario_type"`
	Active            bool               `json:"active"`
	TargetRoleScope   json.RawMessage    `json:"target_role_scope_json"`
	InputScope        json.RawMessage    `json:"input_scope_json"`
	TriggerType       string             `json:"trigger_type"`
	TriggerConfig     json.RawMessage    `json:"trigger_config_json"`
	ProcessingMode    string             `json:"processing_mode"`
	OutputMode        string             `json:"output_mode"`
	UISurface         string             `json:"ui_surface"`
	OutputPolicy      *OutputPolicyRow   `json:"output_policy,omitempty"`
	GovernanceSummary GovernanceSummary  `json:"governance_summary"`
	VisibleRoles      []RoleBindingRow   `json:"visible_roles"`
	SourceBindings    []SourceBindingRow `json:"source_bindings"`
	JobBindings       []JobBindingRow    `json:"job_bindings"`
	UIBindings        []UIBindingRow     `json:"ui_bindings"`
	ConfigJSON        json.RawMessage    `json:"config_json"`
	PreviewConfig     json.RawMessage    `json:"preview_config"`
}

// GovernanceSummary flattens policy flags for builder readability.
type GovernanceSummary struct {
	ReviewRequired     bool   `json:"review_required"`
	PublicationMode    string `json:"publication_mode"`
	CitationsRequired  bool   `json:"citations_required"`
	ProvenanceRequired bool   `json:"provenance_required"`
	OutputSensitivity  int    `json:"output_sensitivity_level"`
	HasOutputPolicy    bool   `json:"has_output_policy"`
}

// RoleBindingWrite is POST role-bindings item.
type RoleBindingWrite struct {
	RoleID           uuid.UUID `json:"role_id"`
	CanSee           bool      `json:"can_see"`
	CanRun           bool      `json:"can_run"`
	CanManage        bool      `json:"can_manage"`
	CanReviewPublish bool      `json:"can_review_publish"`
}

// FromPresetInput is POST /scenarios/from-preset.
type FromPresetInput struct {
	PresetKey string                 `json:"preset_key"`
	Code      string                 `json:"code"`
	Name      string                 `json:"name"`
	Overrides map[string]interface{} `json:"overrides,omitempty"`
}
