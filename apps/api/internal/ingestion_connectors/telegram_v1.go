package ingestion_connectors

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/knowledgelayer/api/internal/ingestion_connectors/families/chat"
)

// TelegramSyncState persists incremental getUpdates cursor (stored inside connector_config_json).
type TelegramSyncState struct {
	LastUpdateID int64 `json:"last_update_id"`
}

// TelegramFeedConfig is governed Telegram feed configuration (secrets + allowlist + optional sync cursor).
type TelegramFeedConfig struct {
	BotToken       string             `json:"bot_token"`
	AllowedChatIDs []int64            `json:"allowed_chat_ids"`
	FeedKind       string             `json:"feed_kind,omitempty"` // chat family; defaults to group_chat
	SyncState      *TelegramSyncState `json:"sync_state,omitempty"`
}

// ParseTelegramFeedConfig parses connector_config_json for Telegram feeds.
func ParseTelegramFeedConfig(raw json.RawMessage) (*TelegramFeedConfig, error) {
	var c TelegramFeedConfig
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// ParseTelegramExternalChatID parses source_feeds.external_ref as a Telegram chat id (int64).
func ParseTelegramExternalChatID(ref string) (int64, error) {
	s := strings.TrimSpace(ref)
	if s == "" {
		return 0, fmt.Errorf("telegram v1: external_ref required (primary chat id)")
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("telegram v1: external_ref must be a numeric chat id: %w", err)
	}
	return n, nil
}

// ValidateTelegramPrimaryChatInAllowlist ensures the governed primary chat is explicitly allowlisted.
func ValidateTelegramPrimaryChatInAllowlist(primaryChatID int64, allowed []int64) error {
	for _, id := range allowed {
		if id == primaryChatID {
			return nil
		}
	}
	return fmt.Errorf("telegram v1: external_ref chat %d must appear in allowed_chat_ids", primaryChatID)
}

// TelegramNextGetUpdatesOffset returns the Bot API offset parameter (last stored update_id + 1, or 0).
func TelegramNextGetUpdatesOffset(cfg *TelegramFeedConfig) int64 {
	if cfg == nil || cfg.SyncState == nil || cfg.SyncState.LastUpdateID <= 0 {
		return 0
	}
	return cfg.SyncState.LastUpdateID + 1
}

// MergeTelegramLastUpdateID writes sync_state.last_update_id into connector_config_json (preserves other keys).
func MergeTelegramLastUpdateID(raw json.RawMessage, lastUpdateID int64) (json.RawMessage, error) {
	var m map[string]any
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &m); err != nil {
			return nil, err
		}
	}
	if m == nil {
		m = make(map[string]any)
	}
	syncObj := map[string]any{"last_update_id": lastUpdateID}
	if existing, ok := m["sync_state"].(map[string]any); ok {
		for k, v := range existing {
			if k != "last_update_id" {
				syncObj[k] = v
			}
		}
	}
	m["sync_state"] = syncObj
	return json.Marshal(m)
}

// ValidateTelegramV1ForActivation enforces ingestion-only, explicit chat registration, and primary external_ref.
func ValidateTelegramV1ForActivation(feed *SourceFeed, cfg *TelegramFeedConfig) error {
	if cfg == nil {
		return fmt.Errorf("telegram v1: missing config")
	}
	if cfg.BotToken == "" {
		return fmt.Errorf("telegram v1: bot_token required in connector_config_json")
	}
	if len(cfg.AllowedChatIDs) == 0 {
		return fmt.Errorf("telegram v1: allowed_chat_ids required (no unrestricted chat reads)")
	}
	if feed == nil {
		return fmt.Errorf("telegram v1: missing feed")
	}
	primary, err := ParseTelegramExternalChatID(feed.ExternalRef)
	if err != nil {
		return err
	}
	if _, err := chat.DefaultFeedKindForTelegram(feed.ConnectorConfigJSON); err != nil {
		return fmt.Errorf("telegram v1: %w", err)
	}
	return ValidateTelegramPrimaryChatInAllowlist(primary, cfg.AllowedChatIDs)
}

func filterTelegramUpdatesByAllowlist(updates []tgUpdate, allowed map[int64]struct{}) []tgUpdate {
	if len(allowed) == 0 {
		return nil
	}
	var out []tgUpdate
	for _, u := range updates {
		if u.Message == nil {
			continue
		}
		cid := u.Message.Chat.ID
		if _, ok := allowed[cid]; ok {
			out = append(out, u)
		}
	}
	return out
}

// filterTelegramUpdatesForFeed keeps updates for the primary chat (external_ref) that are also allowlisted.
func filterTelegramUpdatesForFeed(updates []tgUpdate, primaryChatID int64, allowed map[int64]struct{}) []tgUpdate {
	if _, ok := allowed[primaryChatID]; !ok {
		return nil
	}
	var out []tgUpdate
	for _, u := range updates {
		if u.Message == nil {
			continue
		}
		if u.Message.Chat.ID != primaryChatID {
			continue
		}
		out = append(out, u)
	}
	return out
}

func maxTelegramUpdateID(updates []tgUpdate) int64 {
	var max int64
	for _, u := range updates {
		if u.UpdateID > max {
			max = u.UpdateID
		}
	}
	return max
}

func allowedChatSet(ids []int64) map[int64]struct{} {
	m := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		m[id] = struct{}{}
	}
	return m
}

// tgUpdate is a minimal Bot API update shape for getUpdates ingestion.
type tgUpdate struct {
	UpdateID int64 `json:"update_id"`
	Message  *struct {
		MessageID int    `json:"message_id"`
		Date      int64  `json:"date"`
		Text      string `json:"text"`
		Chat      struct {
			ID int64 `json:"id"`
		} `json:"chat"`
	} `json:"message"`
}

type tgGetUpdatesResp struct {
	OK     bool       `json:"ok"`
	Result []tgUpdate `json:"result"`
}
