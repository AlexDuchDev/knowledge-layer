package ingestion_connectors

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/knowledgelayer/api/internal/ingestion_connectors/families/chat"
)

// SlackSyncState tracks incremental Slack conversations.history.
type SlackSyncState struct {
	// LastMessageTS is the newest message ts ingested in the last successful sync (Slack ts string).
	LastMessageTS string `json:"last_message_ts,omitempty"`
}

// SlackFeedConfig is governed Slack feed configuration (bot token + optional sync cursor).
//
// SigningSecret is the per-app secret Slack uses to sign Events API deliveries
// (see https://api.slack.com/authentication/verifying-requests-from-slack).
// It is REQUIRED only when a feed accepts webhook deliveries via
// POST /connectors/webhook/slack/:source_feed_id; pure polling feeds may leave
// it empty.
type SlackFeedConfig struct {
	BotToken      string          `json:"bot_token"`
	SigningSecret string          `json:"signing_secret,omitempty"`
	SyncState     *SlackSyncState `json:"sync_state,omitempty"`
	FeedKind      string          `json:"feed_kind,omitempty"`
}

// ParseSlackFeedConfig parses connector_config_json for Slack feeds.
func ParseSlackFeedConfig(raw json.RawMessage) (*SlackFeedConfig, error) {
	var c SlackFeedConfig
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// ValidateSlackV1ForActivation enforces chat family feed_kind and bot token.
func ValidateSlackV1ForActivation(feed *SourceFeed, cfg *SlackFeedConfig) error {
	if cfg == nil {
		return fmt.Errorf("slack v1: missing config")
	}
	if cfg.BotToken == "" {
		return fmt.Errorf("slack v1: bot_token required in connector_config_json")
	}
	if err := chat.RequireFeedKindForSlack(feed.ConnectorConfigJSON); err != nil {
		return fmt.Errorf("slack v1: %w", err)
	}
	if feed == nil {
		return fmt.Errorf("slack v1: missing feed")
	}
	ref := strings.TrimSpace(feed.ExternalRef)
	if ref == "" {
		return fmt.Errorf("slack v1: external_ref required (channel id)")
	}
	return nil
}

// MergeSlackLastMessageTS writes sync_state.last_message_ts into connector_config_json.
func MergeSlackLastMessageTS(raw json.RawMessage, lastTS string) (json.RawMessage, error) {
	var m map[string]any
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &m); err != nil {
			return nil, err
		}
	}
	if m == nil {
		m = make(map[string]any)
	}
	syncObj := map[string]any{"last_message_ts": lastTS}
	if existing, ok := m["sync_state"].(map[string]any); ok {
		for k, v := range existing {
			if k != "last_message_ts" {
				syncObj[k] = v
			}
		}
	}
	m["sync_state"] = syncObj
	return json.Marshal(m)
}

// slackAPIEnvelope is the common ok/error shape for Slack Web API.
type slackAPIEnvelope struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

type slackHistoryMessage struct {
	Type       string `json:"type"`
	User       string `json:"user"`
	Text       string `json:"text"`
	Ts         string `json:"ts"`
	ThreadTs   string `json:"thread_ts"`
	ReplyCount int    `json:"reply_count"`
}

type slackConversationsHistoryResp struct {
	slackAPIEnvelope
	Messages         []slackHistoryMessage `json:"messages"`
	HasMore          bool                  `json:"has_more"`
	ResponseMetadata struct {
		NextCursor string `json:"next_cursor"`
	} `json:"response_metadata"`
}

type slackConversationsRepliesResp struct {
	slackAPIEnvelope
	Messages         []slackHistoryMessage `json:"messages"`
	HasMore          bool                  `json:"has_more"`
	ResponseMetadata struct {
		NextCursor string `json:"next_cursor"`
	} `json:"response_metadata"`
}
