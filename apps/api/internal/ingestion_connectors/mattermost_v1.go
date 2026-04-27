package ingestion_connectors

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// MattermostFeedConfig is v1 Mattermost ingestion: personal access token + channel id (external_ref).
//
// OutgoingWebhookToken is the per-channel "Token" from a Mattermost Outgoing
// Webhook integration (System Console → Integrations → Outgoing Webhooks).
// Required ONLY when the feed accepts push deliveries via
// POST /connectors/webhook/mattermost/:source_feed_id; pure polling feeds may
// leave it empty.
type MattermostFeedConfig struct {
	BaseURL              string               `json:"mattermost_base_url"`
	Token                string               `json:"mattermost_token"`
	OutgoingWebhookToken string               `json:"outgoing_webhook_token,omitempty"`
	SyncState            *MattermostSyncState `json:"mattermost_sync_state,omitempty"`
}

// MattermostSyncState stores incremental sync cursor (newest ingested post id).
type MattermostSyncState struct {
	LastPostID string `json:"last_post_id,omitempty"`
}

// ParseMattermostFeedConfig parses connector_config_json for mattermost feeds.
func ParseMattermostFeedConfig(raw json.RawMessage) (*MattermostFeedConfig, error) {
	if len(raw) == 0 {
		return nil, errors.New("mattermost: empty connector_config_json")
	}
	var c MattermostFeedConfig
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, err
	}
	c.BaseURL = strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	c.Token = strings.TrimSpace(c.Token)
	return &c, nil
}

// ValidateMattermostV1ForActivation enforces required fields for an active feed.
func ValidateMattermostV1ForActivation(feed *SourceFeed, cfg *MattermostFeedConfig) error {
	if feed == nil {
		return errors.New("mattermost: nil feed")
	}
	if cfg == nil {
		return errors.New("mattermost: nil config")
	}
	if cfg.BaseURL == "" {
		return errors.New("mattermost: mattermost_base_url required in connector_config_json")
	}
	if cfg.Token == "" {
		return errors.New("mattermost: mattermost_token required in connector_config_json")
	}
	if strings.TrimSpace(feed.ExternalRef) == "" {
		return fmt.Errorf("mattermost: external_ref required (channel id)")
	}
	return nil
}

// MergeMattermostLastPostID updates mattermost_sync_state.last_post_id in connector_config_json (preserves other keys).
func MergeMattermostLastPostID(raw json.RawMessage, postID string) (json.RawMessage, error) {
	var m map[string]any
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &m); err != nil {
			return nil, err
		}
	}
	if m == nil {
		m = make(map[string]any)
	}
	syncObj := map[string]any{"last_post_id": postID}
	if existing, ok := m["mattermost_sync_state"].(map[string]any); ok {
		for k, v := range existing {
			if k != "last_post_id" {
				syncObj[k] = v
			}
		}
	}
	m["mattermost_sync_state"] = syncObj
	return json.Marshal(m)
}
