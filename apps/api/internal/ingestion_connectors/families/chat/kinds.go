// Package chat defines the chat connector family: feed kinds, artifact kinds, and record types.
package chat

// FeedKind describes how the source feed maps to chat semantics (channel, DM, etc.).
type FeedKind string

const (
	FeedKindChannel          FeedKind = "channel"
	FeedKindPrivateChannel   FeedKind = "private_channel"
	FeedKindGroupChat        FeedKind = "group_chat"
	FeedKindDirectChat       FeedKind = "direct_chat"
	FeedKindThreadCollection FeedKind = "thread_collection"
)

// ValidFeedKinds is the allowlist for connector_config_json.feed_kind (chat family).
var ValidFeedKinds = map[FeedKind]struct{}{
	FeedKindChannel:          {},
	FeedKindPrivateChannel:   {},
	FeedKindGroupChat:        {},
	FeedKindDirectChat:       {},
	FeedKindThreadCollection: {},
}

// ArtifactKind is used in raw_artifacts.artifact_type for chat-shaped payloads.
type ArtifactKind string

const (
	ArtifactKindMessage       ArtifactKind = "chat_message"
	ArtifactKindMessageBatch  ArtifactKind = "chat_message_batch"
	ArtifactKindThread        ArtifactKind = "chat_thread"
	ArtifactKindReplySet      ArtifactKind = "chat_reply_set"
	ArtifactKindFileReference ArtifactKind = "chat_file_reference"
)

// RecordTypeChatMessage is normalized_records.record_type for a single chat message.
const RecordTypeChatMessage = "chat_message"

// RecordTypeChatThread is normalized_records.record_type for thread metadata.
const RecordTypeChatThread = "chat_thread"
