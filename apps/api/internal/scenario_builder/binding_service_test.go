package scenario_builder

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/knowledgelayer/api/internal/db"
)

// Admin role from seed 000001
var seedAdminRoleID = uuid.MustParse("10000000-0000-0000-0000-000000000001")

func TestBindingService_RoleMirrorSync(t *testing.T) {
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
	scenarioID := uuid.MustParse("f1000001-0000-4000-8000-000000000001")
	err = svc.Bindings.ReplaceRoleBindings(ctx, scenarioID, []RoleBindingWrite{
		{RoleID: seedAdminRoleID, CanSee: true, CanRun: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	var n int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(1) FROM role_scenario_bindings
		WHERE role_id = $1 AND scenario_key = 'ask_allowed_knowledge'`, seedAdminRoleID).Scan(&n)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("expected mirror row, got %d", n)
	}
}

func TestBindingService_InvalidJob(t *testing.T) {
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
	scenarioID := uuid.MustParse("f1000001-0000-4000-8000-000000000001")
	err = svc.Bindings.ReplaceJobBindings(ctx, scenarioID, []JobBindingRow{
		{KnowledgeJobID: uuid.New(), Relationship: "supports"},
	})
	if err == nil {
		t.Fatal("expected error for missing job")
	}
}
