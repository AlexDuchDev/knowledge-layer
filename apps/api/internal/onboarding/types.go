package onboarding

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// SessionSummary is a row for session lists (setup home).
type SessionSummary struct {
	ID           uuid.UUID `json:"id"`
	Status       string    `json:"status"`
	TemplateCode *string   `json:"template_code,omitempty"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Template is a seeded setup mode.
type Template struct {
	ID           uuid.UUID       `json:"id"`
	Code         string          `json:"code"`
	Title        string          `json:"title"`
	Description  string          `json:"description"`
	MetadataJSON json.RawMessage `json:"metadata_json"`
}

// SessionView is a resumable onboarding session with related rows.
type SessionView struct {
	ID              uuid.UUID                  `json:"id"`
	Status          string                     `json:"status"`
	TemplateCode    *string                    `json:"template_code,omitempty"`
	OrgProfileJSON  json.RawMessage            `json:"org_profile_json"`
	Steps           map[string]json.RawMessage `json:"steps"`
	SelectedPresets []SelectedPresetRow        `json:"selected_presets"`
	Connectors      []ConnectorRow             `json:"connector_selections"`
	FeedDrafts      []json.RawMessage          `json:"source_feed_drafts"`
	Assignment      *AssignmentRow             `json:"assignment_drafts,omitempty"`
	CreatedAt       time.Time                  `json:"created_at"`
	UpdatedAt       time.Time                  `json:"updated_at"`
}

// SelectedPresetRow links a catalog entry to a session slot.
type SelectedPresetRow struct {
	ID                   uuid.UUID       `json:"id"`
	PresetCatalogEntryID uuid.UUID       `json:"preset_catalog_entry_id"`
	PresetType           string          `json:"preset_type"`
	PresetCode           string          `json:"preset_code"`
	Slot                 string          `json:"slot"`
	CustomizationsJSON   json.RawMessage `json:"customizations_json"`
}

// ConnectorRow is an enabled connector family toggle.
type ConnectorRow struct {
	FamilyCode string `json:"connector_family_code"`
	Enabled    bool   `json:"enabled"`
}

// AssignmentRow holds user picks for launch.
type AssignmentRow struct {
	InitialAdminUserID *uuid.UUID      `json:"initial_admin_user_id,omitempty"`
	DomainOwnerUserID  *uuid.UUID      `json:"domain_owner_user_id,omitempty"`
	AssignmentsJSON    json.RawMessage `json:"assignments_json"`
}

// SessionPatch is PATCH body for a session.
type SessionPatch struct {
	OrgProfileJSON  json.RawMessage            `json:"org_profile_json,omitempty"`
	Steps           map[string]json.RawMessage `json:"steps,omitempty"`
	SelectedPresets []SelectedPresetPatch      `json:"selected_presets,omitempty"`
	Connectors      []ConnectorPatch           `json:"connector_selections,omitempty"`
	Assignment      *AssignmentPatch           `json:"assignment,omitempty"`
}

// SelectedPresetPatch upserts selection (full replace of list when non-nil in service).
type SelectedPresetPatch struct {
	PresetCatalogEntryID uuid.UUID       `json:"preset_catalog_entry_id"`
	Slot                 string          `json:"slot"`
	CustomizationsJSON   json.RawMessage `json:"customizations_json,omitempty"`
}

// ConnectorPatch toggles a family.
type ConnectorPatch struct {
	FamilyCode string `json:"connector_family_code"`
	Enabled    bool   `json:"enabled"`
}

// AssignmentPatch updates assignment drafts.
type AssignmentPatch struct {
	InitialAdminUserID *uuid.UUID      `json:"initial_admin_user_id"`
	DomainOwnerUserID  *uuid.UUID      `json:"domain_owner_user_id"`
	AssignmentsJSON    json.RawMessage `json:"assignments_json,omitempty"`
}

// LaunchPreview is POST preview response.
type LaunchPreview struct {
	TemplateCode      *string              `json:"template_code,omitempty"`
	ValidationIssues  []string             `json:"validation_issues"`
	PlannedRoles      []PlannedInstantiate `json:"planned_roles"`
	PlannedScenarios  []PlannedInstantiate `json:"planned_scenarios"`
	PlannedJobs       []PlannedInstantiate `json:"planned_jobs"`
	ConnectorsEnabled []string             `json:"connectors_enabled"`
	Assignments       *AssignmentRow       `json:"assignments,omitempty"`
}

// PlannedInstantiate describes one preset instantiation.
type PlannedInstantiate struct {
	PresetCatalogEntryID uuid.UUID `json:"preset_catalog_entry_id"`
	PresetType           string    `json:"preset_type"`
	Code                 string    `json:"code"`
	Name                 string    `json:"name"`
	Slot                 string    `json:"slot"`
}

// LaunchResult is POST launch response.
type LaunchResult struct {
	SessionID   uuid.UUID     `json:"session_id"`
	Status      string        `json:"status"`
	Created     LaunchCreated `json:"created"`
	LaunchLogID uuid.UUID     `json:"launch_log_id"`
}

// LaunchCreated lists new object ids by kind.
type LaunchCreated struct {
	RoleIDs     []uuid.UUID `json:"role_ids"`
	ScenarioIDs []uuid.UUID `json:"scenario_ids"`
	JobIDs      []uuid.UUID `json:"job_ids"`
}
