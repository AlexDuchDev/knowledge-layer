package microsoft365

import (
	"time"

	"github.com/google/uuid"
)

// NormalizedMailMessage is an Outlook / Exchange mail message (read-only v1).
type NormalizedMailMessage struct {
	SourceFeedID       uuid.UUID  `json:"source_feed_id"`
	ConnectorFamily    string     `json:"connector_family"` // microsoft365
	ConnectorType      string     `json:"connector_type"`   // outlook
	ExternalRef        string     `json:"external_ref"`     // Graph message id
	ExternalMessageID  string     `json:"external_message_id,omitempty"`
	ExternalThreadID   string     `json:"external_thread_id,omitempty"` // conversationId
	InternetMessageID  string     `json:"internet_message_id,omitempty"`
	Subject            string     `json:"subject"`
	BodyPreview        string     `json:"body_preview,omitempty"`
	BodyText           string     `json:"body,omitempty"` // capped in sync
	ReceivedAt         *time.Time `json:"received_at,omitempty"`
	FromRef            string     `json:"from_ref,omitempty"`
	ToRefs             []string   `json:"to_refs,omitempty"`
	CCRefs             []string   `json:"cc_refs,omitempty"`
	FolderOrMailboxRef string     `json:"folder_or_mailbox_ref,omitempty"`
	AttachmentRefs     []string   `json:"attachment_refs,omitempty"` // names or ids
}

// NormalizedTeamsMessage is a Teams channel chat message.
type NormalizedTeamsMessage struct {
	SourceFeedID    uuid.UUID `json:"source_feed_id"`
	ConnectorFamily string    `json:"connector_family"`
	ConnectorType   string    `json:"connector_type"` // teams
	ExternalRef     string    `json:"external_ref"`
	TeamID          string    `json:"team_id"`
	ChannelID       string    `json:"channel_id"`
	BodyPreview     string    `json:"body_preview,omitempty"`
	CreatedAt       string    `json:"created_at,omitempty"`
}
