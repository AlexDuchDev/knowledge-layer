package chat

import (
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// FromSlackMessage maps a Slack message (conversations.history / replies) into NormalizedChatMessage.
// channelID is the feed's external_ref; ts is message ts; threadTs is parent ts when this is a reply.
func FromSlackMessage(sourceFeedID uuid.UUID, channelID, ts, userID, text, threadTs string) NormalizedChatMessage {
	var posted *time.Time
	if sec, ok := slackTsUnixSeconds(ts); ok {
		t := time.Unix(sec, 0).UTC()
		posted = &t
	}
	extThread := ""
	if threadTs != "" && threadTs != ts {
		extThread = threadTs
	}
	return NormalizedChatMessage{
		SourceFeedID:      sourceFeedID,
		ConnectorFamily:   "chat",
		ConnectorType:     "slack",
		ChannelOrChatRef:  channelID,
		ExternalThreadID:  extThread,
		ExternalMessageID: ts,
		PostedAt:          posted,
		AuthorRef:         userID,
		TextBody:          text,
		RawProviderPayload: map[string]any{
			"slack_ts":        ts,
			"slack_thread_ts": threadTs,
		},
	}
}

func slackTsUnixSeconds(ts string) (int64, bool) {
	ts = strings.TrimSpace(ts)
	if ts == "" {
		return 0, false
	}
	parts := strings.SplitN(ts, ".", 2)
	sec, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, false
	}
	return sec, true
}
