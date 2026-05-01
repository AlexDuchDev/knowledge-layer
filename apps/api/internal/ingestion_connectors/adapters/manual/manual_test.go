package manual

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/knowledgelayer/api/internal/ingestion_connectors"
)

func TestAdapter_ValidateSourceFeedConfig(t *testing.T) {
	a := Adapter{}
	conn := ingestion_connectors.Connector{ID: uuid.New(), Type: ingestion_connectors.ConnectorTypeManual}

	t.Run("nil feed rejected", func(t *testing.T) {
		if err := a.ValidateSourceFeedConfig(context.Background(), conn, nil); err == nil {
			t.Fatal("expected error for nil feed")
		}
	})

	t.Run("empty config rejected", func(t *testing.T) {
		feed := &ingestion_connectors.SourceFeed{ID: uuid.New()}
		if err := a.ValidateSourceFeedConfig(context.Background(), conn, feed); err == nil {
			t.Fatal("expected error for missing collection.label")
		}
	})

	t.Run("blank label rejected", func(t *testing.T) {
		raw, err := ingestion_connectors.MarshalManualCollectionConfig(ingestion_connectors.ManualCollectionConfig{Label: "   "})
		if err != nil {
			t.Fatal(err)
		}
		feed := &ingestion_connectors.SourceFeed{ID: uuid.New(), ConnectorConfigJSON: json.RawMessage(raw)}
		if err := a.ValidateSourceFeedConfig(context.Background(), conn, feed); err == nil {
			t.Fatal("expected error for blank label")
		}
	})

	t.Run("valid label accepted", func(t *testing.T) {
		raw, err := ingestion_connectors.MarshalManualCollectionConfig(ingestion_connectors.ManualCollectionConfig{Label: "Q2 research"})
		if err != nil {
			t.Fatal(err)
		}
		feed := &ingestion_connectors.SourceFeed{ID: uuid.New(), ConnectorConfigJSON: json.RawMessage(raw)}
		if err := a.ValidateSourceFeedConfig(context.Background(), conn, feed); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestAdapter_ListAvailableFeedsUnsupported(t *testing.T) {
	a := Adapter{}
	conn := ingestion_connectors.Connector{Type: ingestion_connectors.ConnectorTypeManual}
	_, err := a.ListAvailableFeeds(context.Background(), conn)
	if err != ingestion_connectors.ErrListAvailableFeedsNotSupported {
		t.Fatalf("expected ErrListAvailableFeedsNotSupported, got %v", err)
	}
}

func TestAdapter_MapArtifactMetadata_includesConnectorTag(t *testing.T) {
	a := Adapter{}
	got, err := a.MapArtifactMetadata(context.Background(), "manual_text", []byte("hi"))
	if err != nil {
		t.Fatal(err)
	}
	if got["connector"] != "manual" {
		t.Errorf("connector field = %v, want manual", got["connector"])
	}
	if got["artifact_type"] != "manual_text" {
		t.Errorf("artifact_type field = %v, want manual_text", got["artifact_type"])
	}
	if got["raw_bytes"] != 2 {
		t.Errorf("raw_bytes = %v, want 2", got["raw_bytes"])
	}
}

func TestAdapter_SyncFeedIsNoOp(t *testing.T) {
	// SyncFeed is a no-op for manual; exercising it must not return an error.
	// Manual artifacts arrive via the upload routes, not the sync framework.
	a := Adapter{}
	conn := ingestion_connectors.Connector{Type: ingestion_connectors.ConnectorTypeManual}
	feed := ingestion_connectors.SourceFeed{}
	if err := a.SyncFeed(context.Background(), feed, conn); err != nil {
		t.Fatalf("SyncFeed returned error: %v", err)
	}
}

func TestAdapter_ConnectorTypeMatchesFrameworkConstant(t *testing.T) {
	// Locks in the cross-package contract: the framework constant and the
	// adapter's reported type must agree, otherwise the registry lookup at
	// runtime would silently miss this adapter.
	if got := (Adapter{}).ConnectorType(); got != ingestion_connectors.ConnectorTypeManual {
		t.Fatalf("ConnectorType()=%q, want %q", got, ingestion_connectors.ConnectorTypeManual)
	}
}

func TestParseManualCollectionConfig_strict(t *testing.T) {
	// Garbage JSON must surface as an error so a misconfigured feed cannot
	// silently activate with a zero-value Label that ValidateSourceFeedConfig
	// would then accept.
	if _, err := ingestion_connectors.ParseManualCollectionConfig(json.RawMessage("not json")); err == nil {
		t.Fatal("expected error on malformed config json")
	}
	if !strings.Contains(strings.ToLower(mustErr(t)), "parse") {
		t.Errorf("error wording: %s", mustErr(t))
	}
}

func mustErr(t *testing.T) string {
	t.Helper()
	_, err := ingestion_connectors.ParseManualCollectionConfig(json.RawMessage("not json"))
	if err == nil {
		t.Fatal("expected error")
	}
	return err.Error()
}
