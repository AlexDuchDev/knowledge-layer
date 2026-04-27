package presetcatalog

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// CatalogEntry is a row in preset_catalog_entries.
type CatalogEntry struct {
	ID           uuid.UUID       `json:"id"`
	PresetType   string          `json:"preset_type"`
	Code         string          `json:"code"`
	Name         string          `json:"name"`
	Description  *string         `json:"description,omitempty"`
	Active       bool            `json:"active"`
	MetadataJSON json.RawMessage `json:"metadata_json"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// CategoryRef is a lightweight category for list rows.
type CategoryRef struct {
	Axis  string `json:"axis"`
	Code  string `json:"code"`
	Label string `json:"label"`
}

// ListRow is catalog entry plus categories for API list.
type ListRow struct {
	CatalogEntry
	Categories []CategoryRef `json:"categories"`
}

// RelatedEntry is a neighbor preset with relationship type.
type RelatedEntry struct {
	RelationshipType string       `json:"relationship_type"`
	Entry            CatalogEntry `json:"entry"`
}

// InstantiateRequest is POST /api/presets/:id/instantiate body.
type InstantiateRequest struct {
	Name        *string         `json:"name,omitempty"`
	Code        *string         `json:"code,omitempty"`
	Description *string         `json:"description,omitempty"`
	Purpose     *string         `json:"purpose,omitempty"`
	Overrides   json.RawMessage `json:"overrides,omitempty"`
}

// InstantiateResult is returned after successful instantiation.
type InstantiateResult struct {
	PresetCatalogEntryID uuid.UUID `json:"preset_catalog_entry_id"`
	PresetType           string    `json:"preset_type"`
	Code                 string    `json:"code"`
	TargetKind           string    `json:"target_kind"`
	TargetID             uuid.UUID `json:"target_id"`
	EditPathHint         string    `json:"edit_path_hint"`
}
