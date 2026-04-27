package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/knowledgelayer/api/internal/app"
	"github.com/knowledgelayer/api/internal/config"
	"github.com/knowledgelayer/api/internal/db"
	"github.com/knowledgelayer/api/internal/httpcontext"
	"github.com/knowledgelayer/api/internal/httpserver"
)

// Requires PostgreSQL with migrations + dev seed (DATABASE_URL). Enable with E2E_DB=1.
func TestScenarioGate_SearchAndAsk_Parity(t *testing.T) {
	if os.Getenv("E2E_DB") == "" {
		t.Skip("set E2E_DB=1 and DATABASE_URL")
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Fatal("DATABASE_URL required")
	}
	ctx := context.Background()
	t.Setenv("OPENAI_MOCK", "1")
	if err := db.MigrateUp(dsn); err != nil {
		t.Fatal(err)
	}
	pool, err := db.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	deps, err := app.NewDeps(pool, config.Load())
	if err != nil {
		t.Fatal(err)
	}
	f := fiber.New()
	httpserver.Mount(f, deps)

	const scenarioKey = "e2e_scenario_gate_search_ask"
	viewer := uuid.MustParse("30000000-0000-0000-0000-000000000002")
	viewerRole := uuid.MustParse("10000000-0000-0000-0000-000000000003")

	_, err = pool.Exec(ctx, `DELETE FROM role_scenario_bindings WHERE role_id = $1 AND scenario_key = $2`, viewerRole, scenarioKey)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("search_forbidden_without_binding", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/search?scenario_code="+scenarioKey, nil)
		req.Header.Set(httpcontext.HeaderPrincipalUserID, viewer.String())
		resp, err := f.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			b, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 403 got %d: %s", resp.StatusCode, b)
		}
	})

	t.Run("ask_forbidden_without_binding", func(t *testing.T) {
		body := map[string]string{"question": "x", "scenario_code": scenarioKey}
		raw, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/ask", bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(httpcontext.HeaderPrincipalUserID, viewer.String())
		resp, err := f.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			b, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 403 got %d: %s", resp.StatusCode, b)
		}
	})

	_, err = pool.Exec(ctx, `
		INSERT INTO role_scenario_bindings (role_id, scenario_key) VALUES ($1, $2)
		ON CONFLICT (role_id, scenario_key) DO NOTHING`,
		viewerRole, scenarioKey)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM role_scenario_bindings WHERE role_id = $1 AND scenario_key = $2`, viewerRole, scenarioKey)
	})

	t.Run("search_ok_with_binding", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/search?scenario_code="+scenarioKey, nil)
		req.Header.Set(httpcontext.HeaderPrincipalUserID, viewer.String())
		resp, err := f.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 200 got %d: %s", resp.StatusCode, b)
		}
	})

	t.Run("ask_ok_with_binding", func(t *testing.T) {
		body := map[string]string{"question": "What is documented?", "scenario_code": scenarioKey}
		raw, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/ask", bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(httpcontext.HeaderPrincipalUserID, viewer.String())
		resp, err := f.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusBadRequest {
			b, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 200 or 400 (empty corpus) got %d: %s", resp.StatusCode, b)
		}
	})
}
