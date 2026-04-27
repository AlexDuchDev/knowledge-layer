package chat

import (
	"time"

	"github.com/google/uuid"
)

// FromMattermostPost maps a Mattermost v4 post into NormalizedChatMessage.
// createAtMs is Mattermost's CreateAt (Unix milliseconds).
func FromMattermostPost(sourceFeedID uuid.UUID, channelID, postID, rootID, userID, message string, createAtMs int64) NormalizedChatMessage {
	var posted *time.Time
	if createAtMs > 0 {
		t := time.UnixMilli(createAtMs).UTC()
		posted = &t
	}
	extThread := ""
	if rootID != "" && rootID != postID {
		extThread = rootID
	}
	return NormalizedChatMessage{
		SourceFeedID:      sourceFeedID,
		ConnectorFamily:   "chat",
		ConnectorType:     "mattermost",
		ChannelOrChatRef:  channelID,
		ExternalThreadID:  extThread,
		ExternalMessageID: postID,
		PostedAt:          posted,
		AuthorRef:         userID,
		TextBody:          message,
		RawProviderPayload: map[string]any{
			"mattermost_post_id":      postID,
			"mattermost_root_id":      rootID,
			"mattermost_create_at_ms": createAtMs,
			"mattermost_channel_id":   channelID,
		},
	}
}

// MattermostExternalMessageID builds a stable external id for raw artifacts.
func MattermostExternalMessageID(channelID, postID string) string {
	return channelID + ":" + postID
}
