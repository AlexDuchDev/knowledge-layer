package ingestion_connectors

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestCanActivate_Validation(t *testing.T) {
	s := &Service{}
	ctx := context.Background()
	f := &SourceFeed{
		OwnerID:             uuid.New(),
		DomainID:            uuid.New(),
		AllowedJobTypesJSON: json.RawMessage(`["weekly_digest"]`),
		IngestionMode:       "ingestion_only",
		SyncMode:            SyncModeManual,
		KnowledgeScope:      "domain_linked",
	}
	if err := s.CanActivate(ctx, f); err != nil {
		t.Fatal(err)
	}

	bad := *f
	bad.AllowedJobTypesJSON = json.RawMessage(`[]`)
	if err := s.CanActivate(ctx, &bad); err == nil {
		t.Fatal("expected error for empty allowed jobs")
	}
}
