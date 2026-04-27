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
)

func TestPolicyExceptions_Lifecycle(t *testing.T) {
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
	targetID := uuid.New()

	// Create
	req := httptest.NewRequest(
		http.MethodPost,
		"/governance/policy-exceptions",
		strings.NewReader(`{"target_type":"entity","target_id":"`+targetID.String()+`","override_type":"allow","policy_payload":{},"reason":"pilot exception"}`),
	)
	req.Header.Set(httpcontext.HeaderPrincipalUserID, admin.String())
	req.Header.Set("Content-Type", "application/json")
	resp, err := f.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 201, got %d body=%s", resp.StatusCode, string(b))
	}
	var created map[string]any
	b, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(b, &created); err != nil {
		t.Fatalf("json: %v body=%s", err, string(b))
	}
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatal("expected id")
	}
	if created["status"] != "pending_review" {
		t.Fatalf("expected pending_review, got %v", created["status"])
	}

	// Review activates
	req2 := httptest.NewRequest(http.MethodPost, "/governance/policy-exceptions/"+id+"/review", strings.NewReader(`{}`))
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

	// Revoke disables
	req3 := httptest.NewRequest(http.MethodPost, "/governance/policy-exceptions/"+id+"/revoke", strings.NewReader(`{}`))
	req3.Header.Set(httpcontext.HeaderPrincipalUserID, admin.String())
	req3.Header.Set("Content-Type", "application/json")
	resp3, err := f.Test(req3)
	if err != nil {
		t.Fatal(err)
	}
	resp3.Body.Close()
	if resp3.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp3.StatusCode)
	}

	// List shows revoked effective status
	req4 := httptest.NewRequest(http.MethodGet, "/governance/policy-exceptions", nil)
	req4.Header.Set(httpcontext.HeaderPrincipalUserID, admin.String())
	resp4, err := f.Test(req4)
	if err != nil {
		t.Fatal(err)
	}
	defer resp4.Body.Close()
	if resp4.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp4.StatusCode)
	}
	b4, _ := io.ReadAll(resp4.Body)
	var list []map[string]any
	if err := json.Unmarshal(b4, &list); err != nil {
		t.Fatalf("json: %v body=%s", err, string(b4))
	}
	var found bool
	for _, row := range list {
		if row["id"] == id {
			found = true
			if row["effective_status"] != "revoked" {
				t.Fatalf("expected effective_status revoked, got %v", row["effective_status"])
			}
		}
	}
	if !found {
		t.Fatal("expected created exception in list")
	}
}
