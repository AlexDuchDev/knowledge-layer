package integration

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/knowledgelayer/api/internal/db"
	"github.com/knowledgelayer/api/internal/identity_access"
	"github.com/knowledgelayer/api/internal/role_builder"
	"github.com/knowledgelayer/api/internal/scenario_builder"
)

func TestScenarioBuilder_RoleFullIncludesMirroredScenarioKey(t *testing.T) {
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

	access := identity_access.NewAccessEvaluator(pool)
	rb := role_builder.NewServices(pool, access)
	sb := scenario_builder.NewServices(pool)

	adminRole := uuid.MustParse("10000000-0000-0000-0000-000000000001")
	scenarioID := uuid.MustParse("f1000001-0000-4000-8000-000000000006")

	if err := sb.Bindings.ReplaceRoleBindings(ctx, scenarioID, []scenario_builder.RoleBindingWrite{
		{RoleID: adminRole, CanSee: true},
	}); err != nil {
		t.Fatal(err)
	}

	full, err := rb.Definitions.Get(ctx, adminRole)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, k := range full.ScenarioKeys {
		if k == "decision_explorer" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected decision_explorer in allowed_scenarios, got %v", full.ScenarioKeys)
	}
}
