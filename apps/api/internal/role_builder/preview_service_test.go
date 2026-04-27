package role_builder

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/knowledgelayer/api/internal/db"
)

func TestPreviewService_PreviewRoleShape(t *testing.T) {
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

	repo := NewDefinitionRepository(pool)
	svc := NewPreviewService(repo)
	// Platform admin preset from migration
	id := uuid.MustParse("c1000001-0000-4000-8000-000000000001")
	pv, err := svc.PreviewRole(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if pv.Code == "" || len(pv.Actions) == 0 {
		t.Fatalf("preview missing code or actions: %+v", pv)
	}
	if !pv.Governance.CanOverridePolicies {
		t.Fatal("platform admin preset should allow override in preview")
	}
}
