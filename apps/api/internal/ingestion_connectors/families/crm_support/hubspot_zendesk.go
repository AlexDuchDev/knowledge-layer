package crm_support

import (
	"time"

	"github.com/google/uuid"
)

// NormalizedCRMRecord is a HubSpot (or similar) CRM object row.
type NormalizedCRMRecord struct {
	SourceFeedID         uuid.UUID  `json:"source_feed_id"`
	ConnectorFamily      string     `json:"connector_family"` // crm_support
	ConnectorType        string     `json:"connector_type"`   // hubspot
	ExternalObjectID     string     `json:"external_object_id"`
	ObjectType           string     `json:"object_type"` // contacts, companies, deals
	TitleOrDisplayName   string     `json:"title_or_display_name,omitempty"`
	Owner                string     `json:"owner,omitempty"`
	CustomerOrCompanyRef string     `json:"customer_or_company_ref,omitempty"`
	StageOrStatus        string     `json:"stage_or_status,omitempty"`
	NotesOrEventsPreview string     `json:"notes_or_events_preview,omitempty"`
	CreatedAt            *time.Time `json:"created_at,omitempty"`
	UpdatedAt            *time.Time `json:"updated_at,omitempty"`
}

// NormalizedSupportTicket is a Zendesk (or similar) ticket with thread context.
type NormalizedSupportTicket struct {
	SourceFeedID     uuid.UUID  `json:"source_feed_id"`
	ConnectorFamily  string     `json:"connector_family"`
	ConnectorType    string     `json:"connector_type"` // zendesk
	ExternalObjectID string     `json:"external_object_id"`
	ObjectType       string     `json:"object_type"` // ticket
	Title            string     `json:"title,omitempty"`
	Owner            string     `json:"owner,omitempty"`
	RequesterRef     string     `json:"requester_ref,omitempty"`
	OrganizationRef  string     `json:"organization_ref,omitempty"`
	StageOrStatus    string     `json:"stage_or_status,omitempty"`
	CommentsPreview  string     `json:"comments_preview,omitempty"`
	CreatedAt        *time.Time `json:"created_at,omitempty"`
	UpdatedAt        *time.Time `json:"updated_at,omitempty"`
}
