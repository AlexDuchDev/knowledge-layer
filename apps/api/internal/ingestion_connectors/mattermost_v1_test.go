package ingestion_connectors

import (
	"encoding/json"
	"testing"
)

func TestParseMattermostFeedConfig(t *testing.T) {
	t.Parallel()
	cfg, err := ParseMattermostFeedConfig([]byte(`{"mattermost_base_url":"https://mm.example.com/","mattermost_token":"tok"}`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BaseURL != "https://mm.example.com" || cfg.Token != "tok" {
		t.Fatalf("%+v", cfg)
	}
}

func TestValidateMattermostV1ForActivation_requiresChannel(t *testing.T) {
	t.Parallel()
	cfg, err := ParseMattermostFeedConfig([]byte(`{"mattermost_base_url":"https://x","mattermost_token":"y"}`))
	if err != nil {
		t.Fatal(err)
	}
	feed := &SourceFeed{ExternalRef: " ", Status: "active"}
	if err := ValidateMattermostV1ForActivation(feed, cfg); err == nil {
		t.Fatal("expected error for missing channel external_ref")
	}
}

func TestMergeMattermostLastPostID(t *testing.T) {
	t.Parallel()
	raw := json.RawMessage(`{"mattermost_base_url":"https://x","mattermost_token":"t","extra":1}`)
	out, err := MergeMattermostLastPostID(raw, "post-9")
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	ss, ok := m["mattermost_sync_state"].(map[string]any)
	if !ok {
		t.Fatalf("mattermost_sync_state: %#v", m["mattermost_sync_state"])
	}
	if ss["last_post_id"] != "post-9" {
		t.Fatalf("%#v", ss)
	}
	if m["extra"].(float64) != 1 {
		t.Fatalf("lost extra: %#v", m)
	}
}
