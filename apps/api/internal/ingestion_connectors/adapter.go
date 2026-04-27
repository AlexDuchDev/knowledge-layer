package ingestion_connectors

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// ErrListAvailableFeedsNotSupported means the connector does not implement discovery (e.g. Telegram v1).
var ErrListAvailableFeedsNotSupported = errors.New("ingestion: list available feeds not supported for this connector")

// AvailableFeedRef is a discoverable source candidate before it becomes a governed source_feed row.
type AvailableFeedRef struct {
	ExternalRef string         `json:"external_ref"`
	DisplayName string         `json:"display_name"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// ConnectorAdapter implements connector-specific behavior behind a single registry.
// Framework code resolves adapters by connectors.type; connectors must not bypass this contract.
type ConnectorAdapter interface {
	ConnectorType() string

	// ValidateConnectorConfig checks non-secret connector row configuration (capabilities, config_json).
	ValidateConnectorConfig(ctx context.Context, connector Connector) error

	// ValidateSourceFeedConfig checks feed-level governance config (including connector_config_json) before activation/sync.
	ValidateSourceFeedConfig(ctx context.Context, connector Connector, feed *SourceFeed) error

	// ListAvailableFeeds returns candidate sources when the connector supports discovery (optional).
	ListAvailableFeeds(ctx context.Context, connector Connector) ([]AvailableFeedRef, error)

	// SyncFeed performs one sync pass; persistence of raw artifacts is owned by the ingestion Service / orchestrator, not the adapter alone.
	// v1 Telegram uses Service.SyncTelegram for persistence; the adapter remains a policy gate and future home for Bot API calls.
	SyncFeed(ctx context.Context, feed SourceFeed, connector Connector) error

	// MapArtifactMetadata normalizes connector-specific raw bytes into framework metadata fragments (merged with policy inheritance by the service).
	MapArtifactMetadata(ctx context.Context, artifactType string, rawPayload []byte) (map[string]any, error)
}

// Registry returns adapters keyed by connectors.type (e.g. "telegram").
type Registry struct {
	byType map[string]ConnectorAdapter
}

func NewRegistry(adapters ...ConnectorAdapter) *Registry {
	m := make(map[string]ConnectorAdapter)
	for _, a := range adapters {
		if a == nil {
			continue
		}
		m[a.ConnectorType()] = a
	}
	return &Registry{byType: m}
}

func (r *Registry) AdapterForConnectorType(t string) (ConnectorAdapter, error) {
	if r == nil {
		return nil, fmt.Errorf("ingestion: adapter registry not configured")
	}
	a, ok := r.byType[t]
	if !ok {
		return nil, fmt.Errorf("ingestion: no adapter for connector type %q", t)
	}
	return a, nil
}

// WebhookRequest carries one inbound HTTP delivery to an event-driven connector.
// Headers carries canonicalised header keys (http.Header lookup semantics).
// Body is the verbatim request body — adapters that verify HMAC signatures must
// hash exactly these bytes. Adapters never see the *http.Request directly so
// the route can defend the body buffer (size limits, encoding) before invoking.
type WebhookRequest struct {
	Connector Connector
	Feed      SourceFeed
	Headers   map[string][]string
	Body      []byte
}

// WebhookResult is what an adapter returns from HandleWebhook. The route layer
// chooses how to act:
//   - If Challenge != "" — return it verbatim as the response body (Slack URL
//     verification handshake; comparable patterns for other providers). No
//     artifacts are persisted in this case.
//   - Otherwise persist each RawArtifacts entry through the ingestion service
//     and enqueue normalisation if configured.
type WebhookResult struct {
	Challenge    string
	RawArtifacts []RawArtifactInput
}

// RawArtifactInput is one artifact destined for raw_artifacts. The route owns
// hashing, governance metadata merging, and dedup; adapters return semantic
// fields only.
type RawArtifactInput struct {
	ArtifactType    string
	ExternalRef     string
	Payload         []byte
	SourceCreatedAt *time.Time
	SourceAuthorRef string
	ExtraMetadata   map[string]any
}

// WebhookHandler is an optional capability for event-driven connectors.
// Implement on the same concrete adapter type when a source supports push
// delivery; routes type-assert via Registry.WebhookHandlerForType.
//
// Adapters MUST verify any provider-specific signature on Headers+Body before
// returning a non-error result. The route's job is transport (load feed,
// resolve adapter, persist results) — the adapter is the single owner of
// "is this delivery authentic?".
type WebhookHandler interface {
	HandleWebhook(ctx context.Context, in WebhookRequest) (*WebhookResult, error)
}

// WebhookHandlerForType returns a webhook-capable handler when the registered adapter implements it.
func (r *Registry) WebhookHandlerForType(t string) (WebhookHandler, bool) {
	a, err := r.AdapterForConnectorType(t)
	if err != nil {
		return nil, false
	}
	h, ok := a.(WebhookHandler)
	return h, ok
}
