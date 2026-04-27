package role_builder

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/knowledgelayer/api/internal/db"
)

func TestDefinitionService_CreateValidAndInvalid(t *testing.T) {
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
	svc := NewDefinitionService(repo)

	_, err = svc.Create(ctx, RoleWriteInput{Code: "", Name: "X", ActionCodes: []string{"view"}})
	if err == nil {
		t.Fatal("empty code should fail")
	}

	id, err := svc.Create(ctx, RoleWriteInput{
		Code:        fmt.Sprintf("rb_svc_%s", uuid.NewString()[:8]),
		Name:        "RB Svc Test",
		Category:    "domain",
		ScopeModel:  "global",
		ActionCodes: []string{"view", "search"},
		Governance:  &GovernanceRow{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if id.String() == "" {
		t.Fatal("expected id")
	}
}
