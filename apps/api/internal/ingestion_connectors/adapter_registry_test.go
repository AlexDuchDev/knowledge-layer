package ingestion_connectors

import (
	"context"
	"testing"
)

type registryStubAdapter struct{ typ string }

func (s registryStubAdapter) ConnectorType() string { return s.typ }

func (registryStubAdapter) ValidateConnectorConfig(context.Context, Connector) error { return nil }

func (registryStubAdapter) ValidateSourceFeedConfig(context.Context, Connector, *SourceFeed) error {
	return nil
}

func (registryStubAdapter) ListAvailableFeeds(context.Context, Connector) ([]AvailableFeedRef, error) {
	return nil, ErrListAvailableFeedsNotSupported
}

func (registryStubAdapter) SyncFeed(context.Context, SourceFeed, Connector) error { return nil }

func (registryStubAdapter) MapArtifactMetadata(context.Context, string, []byte) (map[string]any, error) {
	return map[string]any{}, nil
}

func TestRegistry_AdapterForConnectorType(t *testing.T) {
	r := NewRegistry(registryStubAdapter{typ: "telegram"})
	a, err := r.AdapterForConnectorType("telegram")
	if err != nil || a.ConnectorType() != "telegram" {
		t.Fatalf("adapter: %+v %v", a, err)
	}
	_, err = r.AdapterForConnectorType("slack")
	if err == nil {
		t.Fatal("expected error for missing slack adapter")
	}
}

func TestRegistry_WebhookHandlerForType_optional(t *testing.T) {
	r := NewRegistry(registryStubAdapter{typ: "telegram"})
	_, ok := r.WebhookHandlerForType("telegram")
	if ok {
		t.Fatal("stub adapter must not implement WebhookHandler")
	}
}
