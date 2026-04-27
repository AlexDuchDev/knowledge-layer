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

func TestKnowledgeJobTemplates_listAndCreate(t *testing.T) {
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
	t.Cleanup(pool.Close)

	deps, err := app.NewDeps(pool, config.Config{})
	if err != nil {
		t.Fatal(err)
	}
	f := fiber.New()
	httpserver.Mount(f, deps)

	admin := uuid.MustParse("30000000-0000-0000-0000-000000000001")

	t.Run("list_templates", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/knowledge-job-templates", nil)
		resp, err := f.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			t.Fatalf("status %d: %s", resp.StatusCode, b)
		}
		var list []map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
			t.Fatal(err)
		}
		if len(list) < 5 {
			t.Fatalf("expected template catalog, got %d", len(list))
		}
	})

	t.Run("create_from_template_bad_template_400", func(t *testing.T) {
		body := []byte(`{"template_id":"not_a_real_template","owner_id":"` + admin.String() + `"}`)
		req := httptest.NewRequest(http.MethodPost, "/knowledge-jobs", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(httpcontext.HeaderPrincipalUserID, admin.String())
		resp, err := f.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			b, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 400, got %d: %s", resp.StatusCode, b)
		}
	})

	t.Run("create_from_template_ok", func(t *testing.T) {
		domain := uuid.MustParse("32000000-0000-0000-0000-000000000001")
		payload := map[string]any{
			"template_id":       "stale_scan",
			"name":              "",
			"owner_id":          admin.String(),
			"source_scope_json": map[string]string{"domain_id": domain.String()},
			"output_domain_id":  domain.String(),
		}
		body, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodPost, "/knowledge-jobs", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(httpcontext.HeaderPrincipalUserID, admin.String())
		resp, err := f.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			b, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 201, got %d: %s", resp.StatusCode, b)
		}
		var j map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&j); err != nil {
			t.Fatal(err)
		}
		if j["job_type"] != "stale_scan" {
			t.Fatalf("job_type: %v", j["job_type"])
		}
		if j["name"] != "Stale knowledge scan" {
			t.Fatalf("name: %v", j["name"])
		}
		purpose, _ := j["purpose"].(string)
		if purpose == "" {
			t.Fatal("expected purpose populated")
		}
	})
}
