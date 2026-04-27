package chat

import (
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// FromTelegramUpdate maps Telegram getUpdates message fields into NormalizedChatMessage.
// updateID is stored in raw_provider_payload for dedup traceability.
func FromTelegramUpdate(sourceFeedID uuid.UUID, updateID int64, messageID int, dateUnix int64, text string, chatID int64) NormalizedChatMessage {
	var posted *time.Time
	if dateUnix > 0 {
		t := time.Unix(dateUnix, 0).UTC()
		posted = &t
	}
	return NormalizedChatMessage{
		SourceFeedID:      sourceFeedID,
		ConnectorFamily:   "chat",
		ConnectorType:     "telegram",
		ChannelOrChatRef:  strconv.FormatInt(chatID, 10),
		ExternalMessageID: strconv.Itoa(messageID),
		PostedAt:          posted,
		TextBody:          text,
		RawProviderPayload: map[string]any{
			"update_id":        updateID,
			"message_id":       messageID,
			"telegram_chat_id": chatID,
		},
	}
}

// TelegramExternalMessageID builds a stable external id string for Slack/Telegram-style ids.
func TelegramExternalMessageID(chatID int64, messageID int) string {
	return fmt.Sprintf("%d:%d", chatID, messageID)
}
