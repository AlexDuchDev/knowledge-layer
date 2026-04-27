package ingestion_connectors

import (
	"encoding/json"
	"testing"
)

func TestValidateTelegramV1ForActivation_RequiresAllowlist(t *testing.T) {
	feed := &SourceFeed{ExternalRef: "1"}
	if err := ValidateTelegramV1ForActivation(feed, &TelegramFeedConfig{BotToken: "x", AllowedChatIDs: []int64{1}}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateTelegramV1ForActivation(feed, &TelegramFeedConfig{BotToken: "x"}); err == nil {
		t.Fatal("expected error without allowed_chat_ids")
	}
	if err := ValidateTelegramV1ForActivation(feed, &TelegramFeedConfig{AllowedChatIDs: []int64{1}}); err == nil {
		t.Fatal("expected error without bot_token")
	}
}

func TestFilterTelegramUpdatesByAllowlist(t *testing.T) {
	raw := `[{"update_id":1,"message":{"message_id":1,"chat":{"id":10}}},{"update_id":2,"message":{"message_id":2,"chat":{"id":99}}}]`
	var updates []tgUpdate
	if err := json.Unmarshal([]byte(raw), &updates); err != nil {
		t.Fatal(err)
	}
	out := filterTelegramUpdatesByAllowlist(updates, allowedChatSet([]int64{10}))
	if len(out) != 1 || out[0].UpdateID != 1 {
		t.Fatalf("got %+v", out)
	}
}

func TestParseTelegramFeedConfig(t *testing.T) {
	raw, _ := json.Marshal(map[string]any{"bot_token": "t", "allowed_chat_ids": []int64{1, 2}})
	c, err := ParseTelegramFeedConfig(raw)
	if err != nil || c.BotToken != "t" || len(c.AllowedChatIDs) != 2 {
		t.Fatalf("%+v %v", c, err)
	}
}
