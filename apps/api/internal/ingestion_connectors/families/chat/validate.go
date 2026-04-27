package chat

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// ConfigJSON is the subset of connector_config_json used by chat family validation.
type ConfigJSON struct {
	FeedKind string `json:"feed_kind,omitempty"`
}

// ParseConfig extracts chat-related keys from connector_config_json.
func ParseConfig(raw json.RawMessage) (ConfigJSON, error) {
	if len(raw) == 0 {
		return ConfigJSON{}, nil
	}
	var c ConfigJSON
	if err := json.Unmarshal(raw, &c); err != nil {
		return ConfigJSON{}, fmt.Errorf("chat config: %w", err)
	}
	return c, nil
}

// ValidateFeedKind returns an error if feed_kind is set and not in the allowlist.
func ValidateFeedKind(kind string) error {
	if strings.TrimSpace(kind) == "" {
		return nil
	}
	if _, ok := ValidFeedKinds[FeedKind(kind)]; !ok {
		return fmt.Errorf("invalid chat feed_kind %q", kind)
	}
	return nil
}

// RequireFeedKindForSlack returns an error if feed_kind is missing (Slack v1 governance).
func RequireFeedKindForSlack(raw json.RawMessage) error {
	c, err := ParseConfig(raw)
	if err != nil {
		return err
	}
	if strings.TrimSpace(c.FeedKind) == "" {
		return errors.New("slack source feed requires connector_config_json.feed_kind")
	}
	return ValidateFeedKind(c.FeedKind)
}

// DefaultFeedKindForTelegram returns group_chat when unset (backward compatible).
func DefaultFeedKindForTelegram(raw json.RawMessage) (FeedKind, error) {
	c, err := ParseConfig(raw)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(c.FeedKind) == "" {
		return FeedKindGroupChat, nil
	}
	if err := ValidateFeedKind(c.FeedKind); err != nil {
		return "", err
	}
	return FeedKind(c.FeedKind), nil
}
