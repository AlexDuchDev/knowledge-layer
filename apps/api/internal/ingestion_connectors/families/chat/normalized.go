package chat

import (
	"time"

	"github.com/google/uuid"
)

// NormalizedAttachmentRef is a lightweight file or media reference on a message.
type NormalizedAttachmentRef struct {
	ExternalID   string `json:"external_id,omitempty"`
	Filename     string `json:"filename,omitempty"`
	MimeType     string `json:"mime_type,omitempty"`
	SizeBytes    int64  `json:"size_bytes,omitempty"`
	URLHint      string `json:"url_hint,omitempty"` // opaque; may be empty for privacy
	ProviderHint string `json:"provider_hint,omitempty"`
}

// NormalizedChatMessage is the canonical chat message shape for digest and retrieval.
type NormalizedChatMessage struct {
	SourceFeedID       uuid.UUID                 `json:"source_feed_id"`
	ConnectorFamily    string                    `json:"connector_family"` // "chat"
	ConnectorType      string                    `json:"connector_type"`   // telegram, slack, ...
	ChannelOrChatRef   string                    `json:"channel_or_chat_ref"`
	ExternalThreadID   string                    `json:"external_thread_id,omitempty"`
	ExternalMessageID  string                    `json:"external_message_id"`
	PostedAt           *time.Time                `json:"posted_at,omitempty"`
	AuthorRef          string                    `json:"author_ref,omitempty"`
	AuthorDisplay      string                    `json:"author_display,omitempty"`
	TextBody           string                    `json:"text_body,omitempty"`
	Attachments        []NormalizedAttachmentRef `json:"attachments,omitempty"`
	RawProviderPayload map[string]any            `json:"raw_provider_payload,omitempty"` // minimal slice for traceability
}

// NormalizedChatThread groups messages under a thread (Slack thread_ts, etc.).
type NormalizedChatThread struct {
	SourceFeedID     uuid.UUID `json:"source_feed_id"`
	ConnectorFamily  string    `json:"connector_family"`
	ConnectorType    string    `json:"connector_type"`
	ChannelOrChatRef string    `json:"channel_or_chat_ref"`
	ExternalThreadID string    `json:"external_thread_id"`
	RootMessageID    string    `json:"root_message_id,omitempty"`
	TitleOrPreview   string    `json:"title_or_preview,omitempty"`
}
