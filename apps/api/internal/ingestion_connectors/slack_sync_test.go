package ingestion_connectors

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestMergeSlackLastMessageTS(t *testing.T) {
	raw := json.RawMessage(`{"bot_token":"x","feed_kind":"channel"}`)
	out, err := MergeSlackLastMessageTS(raw, "1234.5678")
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	ss, ok := m["sync_state"].(map[string]any)
	if !ok {
		t.Fatalf("sync_state: %#v", m["sync_state"])
	}
	if ss["last_message_ts"] != "1234.5678" {
		t.Fatalf("%#v", ss)
	}
}

func TestSlackOldestParam(t *testing.T) {
	if slackOldestParam(nil) != "" {
		t.Fatal()
	}
	cfg := &SlackFeedConfig{SyncState: &SlackSyncState{LastMessageTS: "99.0"}}
	if slackOldestParam(cfg) != "99.0" {
		t.Fatal()
	}
}

func TestSlackConversationsHistory_mock(t *testing.T) {
	body := `{"ok":true,"messages":[{"type":"message","user":"U1","text":"hi","ts":"10.0","thread_ts":"10.0","reply_count":0}],"has_more":false}`
	s := &Service{
		HTTP: &http.Client{
			Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				if !strings.Contains(req.URL.String(), "conversations.history") {
					t.Fatalf("url %s", req.URL)
				}
				if req.Header.Get("Authorization") != "Bearer tok" {
					t.Fatalf("auth %q", req.Header.Get("Authorization"))
				}
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader(body)),
					Header:     make(http.Header),
				}, nil
			}),
		},
	}
	r, err := s.slackConversationsHistory(context.Background(), "tok", "C123", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !r.OK || len(r.Messages) != 1 || r.Messages[0].Text != "hi" {
		t.Fatalf("%+v", r)
	}
}

func TestSlackConversationsReplies_mock(t *testing.T) {
	body := `{"ok":true,"messages":[{"type":"message","user":"U1","text":"root","ts":"1.0","thread_ts":"1.0"},{"type":"message","user":"U2","text":"reply","ts":"2.0","thread_ts":"1.0"}],"has_more":false}`
	s := &Service{
		HTTP: &http.Client{
			Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				if !strings.Contains(req.URL.String(), "conversations.replies") {
					t.Fatalf("url %s", req.URL)
				}
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader(body)),
					Header:     make(http.Header),
				}, nil
			}),
		},
	}
	r, err := s.slackConversationsReplies(context.Background(), "tok", "C1", "1.0", "")
	if err != nil {
		t.Fatal(err)
	}
	if !r.OK || len(r.Messages) != 2 {
		t.Fatalf("%+v", r)
	}
}
