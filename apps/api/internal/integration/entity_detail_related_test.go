package integration

import (
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
	"github.com/knowledgelayer/api/internal/knowledge_core"
)

func TestEntityDetailAndRelated_AccessSafe(t *testing.T) {
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

	webView := "https://drive.google.com/file/d/abc/view"
	payloadJSON, _ := json.Marshal(map[string]any{
		"source":        "google_drive_ingestion",
		"web_view_link": webView,
	})
	body := "hello world"
	ext := "gdrive:abc"
	doc, err := repo.Create(ctx, knowledge_core.CreateEntityInput{
		Type:             "ReferenceDocument",
		Title:            "Doc",
		Body:             &body,
		OwnerID:          &admin,
		DomainID:         defaultDomain,
		SensitivityLevel: 0,
		TruthMode:        "mirrored_authority",
		LifecycleState:   "draft",
		ExternalRef:      &ext,
		PayloadJSON:      payloadJSON,
	})
	if err != nil {
		t.Fatal(err)
	}

	foreignDomain := uuid.MustParse("33000000-0000-0000-0000-000000000311")
	policy := uuid.MustParse("31000000-0000-0000-0000-000000000001")
	_, err = pool.Exec(ctx, `
		INSERT INTO domains (id, name, owner_id, default_access_policy_id, status)
		VALUES ($1, 'related_other', $2, $3, 'active')
		ON CONFLICT (id) DO NOTHING`, foreignDomain, admin, policy)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO domain_grants (user_id, domain_id, access_level, sensitivity_cap)
		VALUES ($1, $2, 'admin', 3)
		ON CONFLICT (user_id, domain_id) DO NOTHING`, admin, foreignDomain)
	if err != nil {
		t.Fatal(err)
	}

	foreign, err := repo.Create(ctx, knowledge_core.CreateEntityInput{
		Type:             "Insight",
		Title:            "Foreign",
		OwnerID:          &admin,
		DomainID:         foreignDomain,
		SensitivityLevel: 0,
		TruthMode:        "derived",
		LifecycleState:   "draft",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = repo.AddLink(ctx, doc.ID, foreign.ID, "related", &admin)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("detail_includes_open_in_source_url", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/entities/"+doc.ID.String()+"/detail", nil)
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
		var out map[string]any
		if err := json.Unmarshal(b, &out); err != nil {
			t.Fatalf("json: %v body=%s", err, string(b))
		}
		if out["open_in_source_url"] != webView {
			t.Fatalf("expected open_in_source_url=%q, got %v", webView, out["open_in_source_url"])
		}
		if out["truth_mode"] != "mirrored_authority" {
			t.Fatalf("expected truth_mode mirrored_authority, got %v", out["truth_mode"])
		}
	})

	t.Run("viewer_related_does_not_leak_foreign", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/entities/"+doc.ID.String()+"/related", nil)
		req.Header.Set(httpcontext.HeaderPrincipalUserID, viewer.String())
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
		var list []map[string]any
		if err := json.Unmarshal(b, &list); err != nil {
			t.Fatalf("json: %v body=%s", err, string(b))
		}
		for _, it := range list {
			ent, _ := it["entity"].(map[string]any)
			if ent["id"] == foreign.ID.String() {
				t.Fatalf("leaked foreign entity in related rail")
			}
		}
	})

	t.Run("viewer_recommendations_does_not_leak_foreign", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/entities/"+doc.ID.String()+"/recommendations", nil)
		req.Header.Set(httpcontext.HeaderPrincipalUserID, viewer.String())
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
		var list []map[string]any
		if err := json.Unmarshal(b, &list); err != nil {
			t.Fatalf("json: %v body=%s", err, string(b))
		}
		for _, it := range list {
			ent, _ := it["entity"].(map[string]any)
			if ent["id"] == foreign.ID.String() {
				t.Fatalf("leaked foreign entity in recommendations rail")
			}
		}
	})

	t.Run("evidence_omits_ids_without_permissions", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/entities/"+doc.ID.String()+"/evidence", nil)
		req.Header.Set(httpcontext.HeaderPrincipalUserID, viewer.String())
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
		var out map[string]any
		if err := json.Unmarshal(b, &out); err != nil {
			t.Fatalf("json: %v body=%s", err, string(b))
		}

		items, _ := out["evidence"].([]any)
		for _, x := range items {
			m, _ := x.(map[string]any)
			if _, ok := m["raw_artifact_ids"]; ok {
				t.Fatalf("raw_artifact_ids should be omitted for viewer")
			}
			if _, ok := m["normalized_record_ids"]; ok {
				t.Fatalf("normalized_record_ids should be omitted for viewer")
			}
		}
	})
}
