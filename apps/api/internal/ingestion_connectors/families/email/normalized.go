package email

import (
	"time"

	"github.com/google/uuid"
)

// NormalizedEmailMessage is a normalized mail row shared across Gmail / M365 mail where applicable.
type NormalizedEmailMessage struct {
	SourceFeedID      uuid.UUID  `json:"source_feed_id"`
	ConnectorFamily   string     `json:"connector_family"` // email
	ConnectorType     string     `json:"connector_type"`   // gmail | outlook (when using email record_type)
	ExternalRef       string     `json:"external_ref"`
	ExternalMessageID string     `json:"external_message_id,omitempty"`
	ExternalThreadID  string     `json:"external_thread_id,omitempty"`
	Subject           string     `json:"subject"`
	Snippet           string     `json:"snippet,omitempty"`
	Body              string     `json:"body,omitempty"`
	InternalDate      *time.Time `json:"internal_date,omitempty"`
	FromRef           string     `json:"from_ref,omitempty"`
	ToRefs            []string   `json:"to_refs,omitempty"`
	CCRefs            []string   `json:"cc_refs,omitempty"`
	FolderMailboxRef  string     `json:"folder_or_mailbox_ref,omitempty"`
	AttachmentRefs    []string   `json:"attachment_refs,omitempty"`
}
