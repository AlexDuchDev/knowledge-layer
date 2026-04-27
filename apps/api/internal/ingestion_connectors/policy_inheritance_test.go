package ingestion_connectors

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestMergeFeedPolicyMetadata_IncludesDomainAndSensitivity(t *testing.T) {
	did := uuid.MustParse("32000000-0000-0000-0000-000000000001")
	oid := uuid.MustParse("30000000-0000-0000-0000-000000000001")
	cid := uuid.MustParse("20000000-0000-0000-0000-000000000001")
	fid := uuid.MustParse("40000000-0000-0000-0000-000000000001")
	feed := SourceFeed{
		ID:                  fid,
		ConnectorID:         cid,
		OwnerID:             oid,
		DomainID:            did,
		SensitivityLevel:    2,
		AllowedJobTypesJSON: json.RawMessage(`["weekly_digest"]`),
		IngestionMode:       "ingestion_only",
		SyncMode:            SyncModeIncremental,
		Status:              "active",
		SyncStatus:          "idle",
		ExternalRef:         "chat:-100",
		KnowledgeScope:      "domain_linked",
	}
	md := MergeFeedPolicyMetadata(feed, map[string]any{"k": "v"})
	gov, ok := md["governance"].(map[string]any)
	if !ok {
		t.Fatalf("missing governance: %#v", md)
	}
	if gov["domain_id"] != did.String() {
		t.Fatalf("unexpected domain: %#v", gov)
	}
	switch v := gov["sensitivity_level"].(type) {
	case int:
		if v != 2 {
			t.Fatalf("sensitivity %v", v)
		}
	case float64:
		if v != 2 {
			t.Fatalf("sensitivity %v", v)
		}
	default:
		t.Fatalf("sensitivity type %T", v)
	}
	if md["k"] != "v" {
		t.Fatal("base key dropped")
	}
}
