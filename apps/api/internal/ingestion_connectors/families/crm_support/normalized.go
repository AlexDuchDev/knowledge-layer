package crm_support

import (
	"time"

	"github.com/google/uuid"
)

// NormalizedSupportConversation maps a vendor conversation/ticket to a stable shape.
type NormalizedSupportConversation struct {
	SourceFeedID    uuid.UUID  `json:"source_feed_id"`
	ConnectorFamily string     `json:"connector_family"` // crm_support
	ConnectorType   string     `json:"connector_type"`   // intercom
	ExternalRef     string     `json:"external_ref"`
	Title           string     `json:"title,omitempty"`
	State           string     `json:"state,omitempty"`
	UpdatedAt       *time.Time `json:"updated_at,omitempty"`
	BodyPreview     string     `json:"body_preview,omitempty"`
}
