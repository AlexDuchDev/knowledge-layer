package scenario_builder

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/knowledgelayer/api/internal/db"
)

func TestPreviewService_Structure(t *testing.T) {
	if os.Getenv("E2E_DB") == "" {
		t.Skip("set E2E_DB=1 and DATABASE_URL")
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Fatal("DATABASE_URL required")
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

	svc := NewServices(pool)
	id := uuid.MustParse("f1000001-0000-4000-8000-000000000002")
	pv, err := svc.Preview.Build(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if pv.Code != "weekly_team_digest" {
		t.Fatalf("code %s", pv.Code)
	}
	if !pv.GovernanceSummary.HasOutputPolicy {
		t.Fatal("expected policy")
	}
	if len(pv.UIBindings) < 1 {
		t.Fatal("expected ui bindings from seed")
	}
}
