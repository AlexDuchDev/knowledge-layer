package mattermost

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"testing"

	"github.com/knowledgelayer/api/internal/ingestion_connectors"
)

// Mirrors the Slack adapter test fixture style (Phase 2.2.3): no live
// Mattermost workspace required. Mattermost outgoing-webhook auth is a
// shared token in the form body (not HMAC), so the helper just builds an
// `application/x-www-form-urlencoded` payload with the configured token.

const testOutgoingWebhookToken = "mm-token-deadbeef-0000-0000-0000-deadbeefcafe"

func newTestFeed(t *testing.T) ingestion_connectors.SourceFeed {
	t.Helper()
	cfg := ingestion_connectors.MattermostFeedConfig{
		BaseURL:              "https://mm.example.com",
		Token:                "mm-pat",
		OutgoingWebhookToken: testOutgoingWebhookToken,
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal feed config: %v", err)
	}
	return ingestion_connectors.SourceFeed{ConnectorConfigJSON: raw}
}

// formBody returns Mattermost's standard outgoing-webhook form body.
// `extraToken` lets tests inject a wrong token without rebuilding the dict.
func formBody(extraToken string) []byte {
	tok := testOutgoingWebhookToken
	if extraToken != "" {
		tok = extraToken
	}
	v := url.Values{
		"token":         {tok},
		"team_id":       {"team-1"},
		"team_domain":   {"acme"},
		"channel_id":    {"chan-1"},
		"channel_name":  {"engineering"},
		"timestamp":     {"1700000000000"},
		"user_id":       {"user-1"},
		"user_name":     {"alice"},
		"post_id":       {"post-deadbeef"},
		"text":          {"hello world"},
		"trigger_word":  {""},
	}
	return []byte(v.Encode())
}

func TestHandleWebhook_ValidToken_ProducesArtifact(t *testing.T) {
	res, err := Adapter{}.HandleWebhook(context.Background(), ingestion_connectors.WebhookRequest{
		Feed:    newTestFeed(t),
		Headers: map[string][]string{"Content-Type": {"application/x-www-form-urlencoded"}},
		Body:    formBody(""),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil || len(res.RawArtifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %+v", res)
	}
	a := res.RawArtifacts[0]
	if a.ArtifactType != "mattermost_post" {
		t.Fatalf("artifact_type = %q", a.ArtifactType)
	}
	if a.ExternalRef != "post-deadbeef" {
		t.Fatalf("dedup key should be post_id, got %q", a.ExternalRef)
	}
	if a.SourceAuthorRef != "user-1" {
		t.Fatalf("author_ref should track Mattermost user_id, got %q", a.SourceAuthorRef)
	}
	if a.SourceCreatedAt == nil {
		t.Fatal("source_created_at should be derived from timestamp")
	}
	if a.SourceCreatedAt.Unix() != 1700000000 {
		t.Fatalf("source_created_at unix = %d, expected 1700000000", a.SourceCreatedAt.Unix())
	}
	// Verbatim payload preserved for replay tools.
	if string(a.Payload) != string(formBody("")) {
		t.Fatalf("payload should be the verbatim form body")
	}
	if v, _ := a.ExtraMetadata["mattermost_team_domain"].(string); v != "acme" {
		t.Fatalf("metadata.mattermost_team_domain missing or wrong: %v", a.ExtraMetadata)
	}
	if v, _ := a.ExtraMetadata["mattermost_channel_name"].(string); v != "engineering" {
		t.Fatalf("metadata.mattermost_channel_name missing or wrong: %v", a.ExtraMetadata)
	}
}

func TestHandleWebhook_BadToken_Rejected(t *testing.T) {
	_, err := Adapter{}.HandleWebhook(context.Background(), ingestion_connectors.WebhookRequest{
		Feed: newTestFeed(t),
		Body: formBody("wrong-token-still-same-length-as-real0"),
	})
	if !errors.Is(err, ErrWebhookBadToken) {
		t.Fatalf("expected ErrWebhookBadToken, got %v", err)
	}
}

func TestHandleWebhook_EmptyToken_Rejected(t *testing.T) {
	body := []byte("token=&team_id=team-1&post_id=p1")
	_, err := Adapter{}.HandleWebhook(context.Background(), ingestion_connectors.WebhookRequest{
		Feed: newTestFeed(t),
		Body: body,
	})
	if !errors.Is(err, ErrWebhookBadToken) {
		t.Fatalf("expected ErrWebhookBadToken for empty token, got %v", err)
	}
}

func TestHandleWebhook_MissingTokenConfig(t *testing.T) {
	cfg := ingestion_connectors.MattermostFeedConfig{
		BaseURL: "https://mm.example.com",
		Token:   "mm-pat",
		// OutgoingWebhookToken intentionally empty
	}
	raw, _ := json.Marshal(cfg)
	feed := ingestion_connectors.SourceFeed{ConnectorConfigJSON: raw}
	_, err := Adapter{}.HandleWebhook(context.Background(), ingestion_connectors.WebhookRequest{
		Feed: feed,
		Body: formBody(""),
	})
	if !errors.Is(err, ErrWebhookTokenMissing) {
		t.Fatalf("expected ErrWebhookTokenMissing, got %v", err)
	}
}

func TestHandleWebhook_MalformedBody_Rejected(t *testing.T) {
	// `%ZZ` is invalid percent-encoding — url.ParseQuery returns an error.
	_, err := Adapter{}.HandleWebhook(context.Background(), ingestion_connectors.WebhookRequest{
		Feed: newTestFeed(t),
		Body: []byte("token=%ZZ"),
	})
	if !errors.Is(err, ErrWebhookMalformedBody) {
		t.Fatalf("expected ErrWebhookMalformedBody, got %v", err)
	}
}

func TestHandleWebhook_ExternalRefFallback_WhenPostIDMissing(t *testing.T) {
	v := url.Values{
		"token":      {testOutgoingWebhookToken},
		"channel_id": {"chan-x"},
		"timestamp":  {"1700000010000"},
		"user_id":    {"u"},
	}
	res, err := Adapter{}.HandleWebhook(context.Background(), ingestion_connectors.WebhookRequest{
		Feed: newTestFeed(t),
		Body: []byte(v.Encode()),
	})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if res == nil || len(res.RawArtifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %+v", res)
	}
	want := "chan-x:1700000010000"
	if res.RawArtifacts[0].ExternalRef != want {
		t.Fatalf("expected fallback external_ref %q, got %q", want, res.RawArtifacts[0].ExternalRef)
	}
}

func TestHandleWebhook_ConstantTimeTokenCompare(t *testing.T) {
	// Sanity: same-length wrong token is still rejected. (We do NOT measure
	// timing here — that would be flaky in CI; this just ensures the path
	// reaches the constant-time branch instead of a length-based shortcut.)
	wrong := strings.Repeat("z", len(testOutgoingWebhookToken))
	_, err := Adapter{}.HandleWebhook(context.Background(), ingestion_connectors.WebhookRequest{
		Feed: newTestFeed(t),
		Body: formBody(wrong),
	})
	if !errors.Is(err, ErrWebhookBadToken) {
		t.Fatalf("expected ErrWebhookBadToken, got %v", err)
	}
}
