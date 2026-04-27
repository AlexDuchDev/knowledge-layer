package integration

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/knowledgelayer/api/internal/app"
	"github.com/knowledgelayer/api/internal/config"
	"github.com/knowledgelayer/api/internal/db"
	"github.com/knowledgelayer/api/internal/httpcontext"
	"github.com/knowledgelayer/api/internal/httpserver"
	"github.com/knowledgelayer/api/internal/knowledge_core"
)

func TestOwnerRemediation_ListAndAssign(t *testing.T) {
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

	deps, err := app.NewDeps(pool, config.Config{})
	if err != nil {
		t.Fatal(err)
	}
	f := fiber.New()
	httpserver.Mount(f, deps)

	admin := uuid.MustParse("30000000-0000-0000-0000-000000000001")
	viewer := uuid.MustParse("30000000-0000-0000-0000-000000000002")
	defaultDomain := uuid.MustParse("32000000-0000-0000-0000-000000000001")

	repo := knowledge_core.NewEntityRepo(pool)
	ent, err := repo.Create(ctx, knowledge_core.CreateEntityInput{
		Type:             "Insight",
		Title:            "owner_missing_entity",
		OwnerID:          nil,
		DomainID:         defaultDomain,
		SensitivityLevel: 0,
		TruthMode:        "derived",
		LifecycleState:   "draft",
	})
	if err != nil {
		t.Fatal(err)
	}

	// List should include the entity for admin
	req := httptest.NewRequest(http.MethodGet, "/governance/missing-owners", nil)
	req.Header.Set(httpcontext.HeaderPrincipalUserID, admin.String())
	resp, err := f.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d body=%s", resp.StatusCode, string(b))
	}
	b, _ := io.ReadAll(resp.Body)
	var rows []map[string]any
	if err := json.Unmarshal(b, &rows); err != nil {
		t.Fatalf("json: %v body=%s", err, string(b))
	}
	var found bool
	for _, r := range rows {
		if r["resource_type"] == "entity" && r["resource_id"] == ent.ID.String() {
			found = true
		}
	}
	if !found {
		t.Fatal("expected missing-owner entity row present")
	}

	// Assign owner via API
	assignBody := `{"resource_type":"entity","resource_id":"` + ent.ID.String() + `","owner_id":"` + viewer.String() + `"}`
	req2 := httptest.NewRequest(http.MethodPost, "/governance/missing-owners/assign", strings.NewReader(assignBody))
	req2.Header.Set(httpcontext.HeaderPrincipalUserID, admin.String())
	req2.Header.Set("Content-Type", "application/json")
	resp2, err := f.Test(req2)
	if err != nil {
		t.Fatal(err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp2.StatusCode)
	}

	updated, err := repo.Get(ctx, ent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.OwnerID == nil || *updated.OwnerID != viewer {
		t.Fatalf("expected owner updated to viewer; got %+v", updated.OwnerID)
	}
}
