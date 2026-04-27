package role_builder

import (
	"time"

	"github.com/google/uuid"
)

// RoleSummary is a list row for GET /roles.
type RoleSummary struct {
	ID          uuid.UUID `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	Category    string    `json:"category"`
	Active      bool      `json:"active"`
	ScopeModel  string    `json:"scope_model"`
	IsPreset    bool      `json:"is_preset"`
	PresetKey   *string   `json:"preset_key,omitempty"`
	IsSystem    bool      `json:"is_system"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// RoleFull is a complete role definition for API detail.
type RoleFull struct {
	RoleSummary
	ClonedFromRoleID *uuid.UUID         `json:"cloned_from_role_id,omitempty"`
	SourcePresetCode *string            `json:"source_preset_code,omitempty"`
	ActionCodes      []string           `json:"action_codes"`
	DomainIDs        []uuid.UUID        `json:"allowed_domains"`
	EntityTypes      []string           `json:"allowed_entity_types"`
	SourceScopes     []SourceScopeRef   `json:"allowed_source_scopes"`
	ScenarioKeys     []string           `json:"allowed_scenarios"`
	DashboardKeys    []string           `json:"allowed_dashboards"`
	JobPermissions   []JobPermissionRow `json:"job_permissions"`
	Governance       GovernanceRow      `json:"governance"`
}

// SourceScopeRef is one row in role_source_scope_bindings.
type SourceScopeRef struct {
	ScopeKind string `json:"scope_kind"`
	ScopeRef  string `json:"scope_ref"`
}

// JobPermissionRow is role_job_permissions.
type JobPermissionRow struct {
	JobID              uuid.UUID `json:"knowledge_job_id"`
	CanRun             bool      `json:"can_run"`
	CanConfigure       bool      `json:"can_configure"`
	CanReviewJobOutput bool      `json:"can_review_job_output"`
}

// GovernanceRow is role_governance_permissions (defaults false when row missing).
type GovernanceRow struct {
	CanReviewOutputs     bool `json:"can_review_outputs"`
	CanApproveOutputs    bool `json:"can_approve_outputs"`
	CanPublishOutputs    bool `json:"can_publish_outputs"`
	CanOverridePolicies  bool `json:"can_override_policies"`
	CanManageAssignments bool `json:"can_manage_assignments"`
}

// RoleWriteInput is create/patch payload (bindings replaced entirely on patch).
type RoleWriteInput struct {
	Code        string
	Name        string
	Description *string
	Category    string
	Active      *bool
	ScopeModel  string

	ActionCodes    []string
	DomainIDs      []uuid.UUID
	EntityTypes    []string
	SourceScopes   []SourceScopeRef
	ScenarioKeys   []string
	DashboardKeys  []string
	JobPermissions []JobPermissionWrite
	Governance     *GovernanceRow
}

// JobPermissionWrite binds job permissions by job id.
type JobPermissionWrite struct {
	JobID              uuid.UUID
	CanRun             bool
	CanConfigure       bool
	CanReviewJobOutput bool
}

// EffectiveAccessPreview is the stable JSON shape for GET /roles/:id/preview.
type EffectiveAccessPreview struct {
	RoleID       uuid.UUID          `json:"role_id"`
	Code         string             `json:"code"`
	Name         string             `json:"name"`
	Category     string             `json:"category"`
	ScopeModel   string             `json:"scope_model"`
	Active       bool               `json:"active"`
	Domains      []uuid.UUID        `json:"allowed_domains"`
	EntityTypes  []string           `json:"allowed_entity_types"`
	SourceScopes []SourceScopeRef   `json:"allowed_source_scopes"`
	Scenarios    []string           `json:"allowed_scenarios"`
	Dashboards   []string           `json:"allowed_dashboards"`
	Actions      []string           `json:"allowed_actions"`
	Governance   GovernanceRow      `json:"governance"`
	Jobs         []JobPermissionRow `json:"job_permissions"`
}

// UserRoleAssignment is a user_role_bindings row exposed by Role Builder.
type UserRoleAssignment struct {
	ID        uuid.UUID  `json:"id"`
	UserID    uuid.UUID  `json:"user_id"`
	RoleID    uuid.UUID  `json:"role_id"`
	ScopeType string     `json:"scope_type"`
	ScopeID   *uuid.UUID `json:"scope_id,omitempty"`
	GrantedBy *uuid.UUID `json:"granted_by,omitempty"`
	GrantedAt time.Time  `json:"granted_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}
