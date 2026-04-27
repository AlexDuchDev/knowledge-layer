package ingestion_connectors

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/knowledgelayer/api/internal/db"
)

// TestCreateSourceFeed_insertsRow asserts the INSERT in CreateSourceFeed has
// matching column and values counts. The original draft used $1..$16 + 2
// literals (18 expressions) for 17 columns, which PG rejected with SQLSTATE
// 42601 ("INSERT has more expressions than target columns"). Surfaced during
// R3 RC validation when adapter-level validation finally let the request reach
// CreateSourceFeed (the earlier ExternalRef-drop bug had been masking it).
//
// Skipped when DATABASE_URL is unset (CI runs the verify job with DB).
func TestCreateSourceFeed_insertsRow(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}

	ctx := context.Background()
	if err := db.MigrateUp(dsn); err != nil {
		t.Fatal(err)
	}
	pool, err := db.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	svc := NewService(pool, nil)

	in := CreateSourceFeedInput{
		ConnectorID:         uuid.MustParse("20000000-0000-0000-0000-000000000012"), // filesystem
		DomainID:            uuid.MustParse("32000000-0000-0000-0000-000000000001"), // seed default
		OwnerID:             uuid.MustParse("30000000-0000-0000-0000-000000000001"), // seed admin
		DisplayName:         "test-create-source-feed-regression",
		ExternalRef:         "opensearch", // any safe relative path under /data
		SensitivityLevel:    0,
		IngestionMode:       "ingestion_only",
		AllowedJobTypesJSON: json.RawMessage(`["weekly_digest"]`),
		ConnectorConfigJSON: json.RawMessage(`{}`),
	}

	feed, err := svc.CreateSourceFeed(ctx, in)
	if err != nil {
		t.Fatalf("CreateSourceFeed failed: %v", err)
	}
	if feed == nil || feed.ID == uuid.Nil {
		t.Fatal("expected created feed with non-nil ID")
	}

	// Confirm the row really landed in the column shape we declare.
	defer pool.Exec(ctx, `DELETE FROM source_feeds WHERE id = $1`, feed.ID)

	if feed.ConnectorID != in.ConnectorID {
		t.Errorf("ConnectorID round-trip mismatch: want %s, got %s", in.ConnectorID, feed.ConnectorID)
	}
	if feed.ExternalRef != in.ExternalRef {
		t.Errorf("ExternalRef round-trip mismatch: want %q, got %q", in.ExternalRef, feed.ExternalRef)
	}
	if feed.DisplayName != in.DisplayName {
		t.Errorf("DisplayName round-trip mismatch: want %q, got %q", in.DisplayName, feed.DisplayName)
	}
	if feed.Status != "draft" {
		t.Errorf("expected default status='draft' literal, got %q", feed.Status)
	}
	if feed.SyncStatus != "idle" {
		t.Errorf("expected default sync_status='idle' literal, got %q", feed.SyncStatus)
	}
}
