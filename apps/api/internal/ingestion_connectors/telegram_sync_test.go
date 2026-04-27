package ingestion_connectors

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestFetchTelegramUpdates_includesOffset(t *testing.T) {
	var got string
	s := &Service{
		HTTP: &http.Client{
			Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				got = req.URL.String()
				body := `{"ok":true,"result":[]}`
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader(body)),
					Header:     make(http.Header),
				}, nil
			}),
		},
	}
	_, _, _ = s.fetchTelegramUpdates("test-token", 99)
	if !strings.Contains(got, "offset=99") {
		t.Fatalf("expected offset in URL, got %q", got)
	}
}

func TestTelegramNextGetUpdatesOffset(t *testing.T) {
	if TelegramNextGetUpdatesOffset(&TelegramFeedConfig{}) != 0 {
		t.Fatal("expected 0 without sync state")
	}
	cfg := &TelegramFeedConfig{SyncState: &TelegramSyncState{LastUpdateID: 5}}
	if TelegramNextGetUpdatesOffset(cfg) != 6 {
		t.Fatalf("got %d", TelegramNextGetUpdatesOffset(cfg))
	}
}

func TestMergeTelegramLastUpdateID(t *testing.T) {
	raw := json.RawMessage(`{"bot_token":"x","allowed_chat_ids":[1]}`)
	out, err := MergeTelegramLastUpdateID(raw, 7)
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
	if int(ss["last_update_id"].(float64)) != 7 {
		t.Fatalf("%#v", ss)
	}
}

func TestFilterTelegramUpdatesForFeed_primaryOnly(t *testing.T) {
	raw := `[{"update_id":1,"message":{"message_id":1,"chat":{"id":10}}},{"update_id":2,"message":{"message_id":2,"chat":{"id":20}}}]`
	var updates []tgUpdate
	if err := json.Unmarshal([]byte(raw), &updates); err != nil {
		t.Fatal(err)
	}
	out := filterTelegramUpdatesForFeed(updates, 20, allowedChatSet([]int64{10, 20}))
	if len(out) != 1 || out[0].UpdateID != 2 {
		t.Fatalf("%+v", out)
	}
}

func TestMaxTelegramUpdateID(t *testing.T) {
	u := []tgUpdate{{UpdateID: 3}, {UpdateID: 9}, {UpdateID: 1}}
	if maxTelegramUpdateID(u) != 9 {
		t.Fatal(maxTelegramUpdateID(u))
	}
}

func TestValidateTelegramV1ForActivation_primaryNotAllowlisted(t *testing.T) {
	feed := &SourceFeed{ExternalRef: "99"}
	cfg := &TelegramFeedConfig{BotToken: "t", AllowedChatIDs: []int64{1, 2}}
	if err := ValidateTelegramV1ForActivation(feed, cfg); err == nil {
		t.Fatal("expected error when external_ref chat not in allowlist")
	}
}

func TestBuildRawArtifactMetadataJSON_includesGovernance(t *testing.T) {
	s := &Service{Registry: NewRegistry(metadataStubAdapter{})}
	did := uuid.MustParse("32000000-0000-0000-0000-000000000001")
	oid := uuid.MustParse("30000000-0000-0000-0000-000000000001")
	cid := uuid.MustParse("20000000-0000-0000-0000-000000000001")
	fid := uuid.MustParse("40000000-0000-0000-0000-000000000001")
	feed := SourceFeed{
		ID:                  fid,
		ConnectorID:         cid,
		OwnerID:             oid,
		DomainID:            did,
		SensitivityLevel:    1,
		AllowedJobTypesJSON: json.RawMessage(`["digest"]`),
		IngestionMode:       "ingestion_only",
		SyncMode:            SyncModeManual,
		ExternalRef:         "-100",
		KnowledgeScope:      "domain_linked",
	}
	conn := Connector{Type: "telegram"}
	payload := []byte(`{"update_id":1,"message":{"chat":{"id":-100}}}`)
	extra := map[string]any{"telegram_update": map[string]any{"update_id": float64(1)}}
	b, err := s.buildRawArtifactMetadataJSON(context.Background(), conn, feed, "telegram_update", payload, extra)
	if err != nil {
		t.Fatal(err)
	}
	var md map[string]any
	if err := json.Unmarshal(b, &md); err != nil {
		t.Fatal(err)
	}
	gov, ok := md["governance"].(map[string]any)
	if !ok {
		t.Fatalf("missing governance: %s", string(b))
	}
	if gov["domain_id"] != did.String() {
		t.Fatalf("domain: %#v", gov)
	}
}

// metadataStubAdapter satisfies ConnectorAdapter for metadata merge tests without importing adapters/telegram.
type metadataStubAdapter struct{}

func (metadataStubAdapter) ConnectorType() string { return "telegram" }

func (metadataStubAdapter) ValidateConnectorConfig(context.Context, Connector) error { return nil }

func (metadataStubAdapter) ValidateSourceFeedConfig(context.Context, Connector, *SourceFeed) error {
	return nil
}

func (metadataStubAdapter) ListAvailableFeeds(context.Context, Connector) ([]AvailableFeedRef, error) {
	return nil, ErrListAvailableFeedsNotSupported
}

func (metadataStubAdapter) SyncFeed(context.Context, SourceFeed, Connector) error { return nil }

func (metadataStubAdapter) MapArtifactMetadata(_ context.Context, _ string, _ []byte) (map[string]any, error) {
	return map[string]any{"from_adapter": true}, nil
}
