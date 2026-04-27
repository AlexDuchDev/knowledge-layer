package slack

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/knowledgelayer/api/internal/ingestion_connectors"
)

// Tests verify the OSS-quality Slack webhook handler without requiring a live
// Slack workspace: we craft signed payloads with a known signing_secret, freeze
// the clock via nowFn, and assert the adapter accepts/rejects as Slack would.
//
// The signature scheme is the documented one (HMAC-SHA256 over
// "v0:{ts}:{body}"). Operators can re-use these tests as a reference for
// generating curl one-liners against a local API.

const testSigningSecret = "8f742231b10e8888abcd99yyyzzz85a5"

func newTestFeed(t *testing.T) ingestion_connectors.SourceFeed {
	t.Helper()
	cfg := ingestion_connectors.SlackFeedConfig{
		BotToken:      "xoxb-test-token",
		SigningSecret: testSigningSecret,
		FeedKind:      "channel",
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal feed config: %v", err)
	}
	return ingestion_connectors.SourceFeed{
		ConnectorConfigJSON: raw,
	}
}

func sign(t *testing.T, ts string, body []byte) string {
	t.Helper()
	mac := hmac.New(sha256.New, []byte(testSigningSecret))
	mac.Write([]byte("v0:" + ts + ":"))
	mac.Write(body)
	return "v0=" + hex.EncodeToString(mac.Sum(nil))
}

func freezeClock(t *testing.T, at time.Time) {
	t.Helper()
	prev := nowFn
	nowFn = func() time.Time { return at }
	t.Cleanup(func() { nowFn = prev })
}

func TestHandleWebhook_URLVerificationChallenge(t *testing.T) {
	body := []byte(`{"type":"url_verification","challenge":"test_challenge_value","token":"deprecated"}`)
	now := time.Unix(1700000000, 0)
	freezeClock(t, now)
	ts := strconv.FormatInt(now.Unix(), 10)
	headers := map[string][]string{
		"X-Slack-Request-Timestamp": {ts},
		"X-Slack-Signature":         {sign(t, ts, body)},
	}

	res, err := Adapter{}.HandleWebhook(context.Background(), ingestion_connectors.WebhookRequest{
		Feed:    newTestFeed(t),
		Headers: headers,
		Body:    body,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil || res.Challenge != "test_challenge_value" {
		t.Fatalf("expected challenge round-trip, got %+v", res)
	}
	if len(res.RawArtifacts) != 0 {
		t.Fatalf("url_verification must not produce artifacts, got %d", len(res.RawArtifacts))
	}
}

func TestHandleWebhook_EventCallback_ProducesArtifact(t *testing.T) {
	body := []byte(`{
        "type":"event_callback",
        "team_id":"T123",
        "api_app_id":"A123",
        "event_id":"Ev0PV52K21",
        "event_time":1700000010,
        "event":{
            "type":"message",
            "channel":"C0XYZ",
            "user":"U0ABC",
            "ts":"1700000010.000200",
            "text":"hello world"
        }
    }`)
	now := time.Unix(1700000020, 0) // 10s after event_time → within skew
	freezeClock(t, now)
	ts := strconv.FormatInt(now.Unix(), 10)
	headers := map[string][]string{
		"X-Slack-Request-Timestamp": {ts},
		"X-Slack-Signature":         {sign(t, ts, body)},
	}

	res, err := Adapter{}.HandleWebhook(context.Background(), ingestion_connectors.WebhookRequest{
		Feed:    newTestFeed(t),
		Headers: headers,
		Body:    body,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil || len(res.RawArtifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %+v", res)
	}
	a := res.RawArtifacts[0]
	if a.ArtifactType != "slack_message" {
		t.Fatalf("artifact_type = %q", a.ArtifactType)
	}
	if a.ExternalRef != "Ev0PV52K21" {
		t.Fatalf("dedup key should be event_id, got %q", a.ExternalRef)
	}
	if a.SourceAuthorRef != "U0ABC" {
		t.Fatalf("author_ref should track Slack user id, got %q", a.SourceAuthorRef)
	}
	if a.SourceCreatedAt == nil {
		t.Fatal("source_created_at should be derived from event_time")
	}
	if a.SourceCreatedAt.Unix() != 1700000010 {
		t.Fatalf("source_created_at = %s, expected 2023-11-14 22:13:30 UTC", a.SourceCreatedAt)
	}
	if string(a.Payload) != string(body) {
		t.Fatalf("payload should be the verbatim envelope (preserves team_id/event_id for replay)")
	}
	if v, _ := a.ExtraMetadata["slack_team_id"].(string); v != "T123" {
		t.Fatalf("metadata.slack_team_id missing or wrong: %v", a.ExtraMetadata)
	}
	if v, _ := a.ExtraMetadata["slack_channel"].(string); v != "C0XYZ" {
		t.Fatalf("metadata.slack_channel missing or wrong: %v", a.ExtraMetadata)
	}
}

func TestHandleWebhook_BadSignature(t *testing.T) {
	body := []byte(`{"type":"event_callback"}`)
	now := time.Unix(1700000000, 0)
	freezeClock(t, now)
	ts := strconv.FormatInt(now.Unix(), 10)
	headers := map[string][]string{
		"X-Slack-Request-Timestamp": {ts},
		"X-Slack-Signature":         {"v0=" + hex.EncodeToString(make([]byte, sha256.Size))}, // garbage but right length
	}

	_, err := Adapter{}.HandleWebhook(context.Background(), ingestion_connectors.WebhookRequest{
		Feed:    newTestFeed(t),
		Headers: headers,
		Body:    body,
	})
	if !errors.Is(err, ErrWebhookBadSignature) {
		t.Fatalf("expected ErrWebhookBadSignature, got %v", err)
	}
}

func TestHandleWebhook_StaleTimestamp_RejectsReplay(t *testing.T) {
	body := []byte(`{"type":"event_callback"}`)
	now := time.Unix(1700000000, 0)
	freezeClock(t, now)
	staleTS := strconv.FormatInt(now.Unix()-int64(MaxWebhookSkew/time.Second)-10, 10) // 10s past the cutoff
	headers := map[string][]string{
		"X-Slack-Request-Timestamp": {staleTS},
		"X-Slack-Signature":         {sign(t, staleTS, body)},
	}

	_, err := Adapter{}.HandleWebhook(context.Background(), ingestion_connectors.WebhookRequest{
		Feed:    newTestFeed(t),
		Headers: headers,
		Body:    body,
	})
	if !errors.Is(err, ErrWebhookStaleTimestamp) {
		t.Fatalf("expected ErrWebhookStaleTimestamp, got %v", err)
	}
}

func TestHandleWebhook_MissingHeaders(t *testing.T) {
	_, err := Adapter{}.HandleWebhook(context.Background(), ingestion_connectors.WebhookRequest{
		Feed:    newTestFeed(t),
		Headers: map[string][]string{},
		Body:    []byte(`{}`),
	})
	if !errors.Is(err, ErrWebhookMissingHeaders) {
		t.Fatalf("expected ErrWebhookMissingHeaders, got %v", err)
	}
}

func TestHandleWebhook_MissingSigningSecret(t *testing.T) {
	cfg := ingestion_connectors.SlackFeedConfig{BotToken: "xoxb-test", FeedKind: "channel"}
	raw, _ := json.Marshal(cfg)
	feed := ingestion_connectors.SourceFeed{ConnectorConfigJSON: raw}
	_, err := Adapter{}.HandleWebhook(context.Background(), ingestion_connectors.WebhookRequest{
		Feed: feed,
		Body: []byte(`{}`),
		Headers: map[string][]string{
			"X-Slack-Request-Timestamp": {"1700000000"},
			"X-Slack-Signature":         {"v0=00"},
		},
	})
	if !errors.Is(err, ErrWebhookSigningSecretMissing) {
		t.Fatalf("expected ErrWebhookSigningSecretMissing, got %v", err)
	}
}

func TestHandleWebhook_HeaderLookupIsCaseInsensitive(t *testing.T) {
	body := []byte(`{"type":"event_callback","event":{"type":"message","channel":"C","ts":"1.0"}}`)
	now := time.Unix(1700000000, 0)
	freezeClock(t, now)
	ts := strconv.FormatInt(now.Unix(), 10)
	headers := map[string][]string{
		"x-slack-request-timestamp": {ts}, // lowercase
		"x-slack-signature":         {sign(t, ts, body)},
	}
	res, err := Adapter{}.HandleWebhook(context.Background(), ingestion_connectors.WebhookRequest{
		Feed:    newTestFeed(t),
		Headers: headers,
		Body:    body,
	})
	if err != nil {
		t.Fatalf("expected case-insensitive header lookup to succeed, got %v", err)
	}
	if res == nil || len(res.RawArtifacts) != 1 {
		t.Fatalf("expected 1 artifact from event_callback, got %+v", res)
	}
}

func TestHandleWebhook_UnknownEventType_NoArtifact(t *testing.T) {
	body := []byte(`{"type":"future_envelope_we_dont_know"}`)
	now := time.Unix(1700000000, 0)
	freezeClock(t, now)
	ts := strconv.FormatInt(now.Unix(), 10)
	headers := map[string][]string{
		"X-Slack-Request-Timestamp": {ts},
		"X-Slack-Signature":         {sign(t, ts, body)},
	}

	res, err := Adapter{}.HandleWebhook(context.Background(), ingestion_connectors.WebhookRequest{
		Feed:    newTestFeed(t),
		Headers: headers,
		Body:    body,
	})
	if err != nil {
		t.Fatalf("expected unknown-but-signed type to be silently accepted, got %v", err)
	}
	if res == nil {
		t.Fatal("expected empty result, got nil")
	}
	if res.Challenge != "" || len(res.RawArtifacts) != 0 {
		t.Fatalf("expected empty WebhookResult, got %+v", res)
	}
}
