package chat

import (
	"encoding/json"
	"testing"
)

func TestValidateFeedKind(t *testing.T) {
	if err := ValidateFeedKind(""); err != nil {
		t.Fatal(err)
	}
	if err := ValidateFeedKind("channel"); err != nil {
		t.Fatal(err)
	}
	if ValidateFeedKind("invalid") == nil {
		t.Fatal("expected error")
	}
}

func TestRequireFeedKindForSlack(t *testing.T) {
	if err := RequireFeedKindForSlack(nil); err == nil {
		t.Fatal("expected error for empty")
	}
	raw, _ := json.Marshal(map[string]string{"feed_kind": "channel"})
	if err := RequireFeedKindForSlack(raw); err != nil {
		t.Fatal(err)
	}
}

func TestDefaultFeedKindForTelegram(t *testing.T) {
	k, err := DefaultFeedKindForTelegram(nil)
	if err != nil || k != FeedKindGroupChat {
		t.Fatalf("got %v %v", k, err)
	}
	raw, _ := json.Marshal(map[string]string{"feed_kind": "direct_chat"})
	k, err = DefaultFeedKindForTelegram(raw)
	if err != nil || k != FeedKindDirectChat {
		t.Fatalf("got %v %v", k, err)
	}
}
