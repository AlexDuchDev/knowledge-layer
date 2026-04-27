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

func TestEntityAsk_PermissionAndNoLeak(t *testing.T) {
	if os.Getenv("E2E_DB") == "" {
		t.Skip("set E2E_DB=1 and DATABASE_URL")
	}
	// For deterministic E2E, use the built-in mock mode in internal/llm.
	_ = os.Setenv("OPENAI_MOCK", "1")
	_ = os.Setenv("AI_PRIVACY_DEV_PLAINTEXT_STORE", "1")

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
	policy := uuid.MustParse("31000000-0000-0000-0000-000000000001")

	otherDomain := uuid.MustParse("33000000-0000-0000-0000-000000000211")
	_, err = pool.Exec(ctx, `
		INSERT INTO domains (id, name, owner_id, default_access_policy_id, status)
		VALUES ($1, 'ask_other', $2, $3, 'active')
		ON CONFLICT (id) DO NOTHING`, otherDomain, admin, policy)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO domain_grants (user_id, domain_id, access_level, sensitivity_cap)
		VALUES ($1, $2, 'admin', 3)
		ON CONFLICT (user_id, domain_id) DO NOTHING`, admin, otherDomain)
	if err != nil {
		t.Fatal(err)
	}

	repo := knowledge_core.NewEntityRepo(pool)
	root, err := repo.Create(ctx, knowledge_core.CreateEntityInput{
		Type:             "Insight",
		Title:            "ask_root",
		OwnerID:          &viewer,
		DomainID:         defaultDomain,
		SensitivityLevel: 0,
		TruthMode:        "derived",
		LifecycleState:   "draft",
	})
	if err != nil {
		t.Fatal(err)
	}
	foreign, err := repo.Create(ctx, knowledge_core.CreateEntityInput{
		Type:             "Insight",
		Title:            "ask_foreign",
		OwnerID:          &admin,
		DomainID:         otherDomain,
		SensitivityLevel: 0,
		TruthMode:        "derived",
		LifecycleState:   "draft",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = repo.AddLink(ctx, root.ID, foreign.ID, "related", &admin)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("viewer_forbidden_when_scenario_code_not_permitted", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/entities/"+root.ID.String()+"/ask", strings.NewReader(`{"question":"hi","scenario_code":"__no_such_scenario_for_viewer__"}`))
		req.Header.Set(httpcontext.HeaderPrincipalUserID, viewer.String())
		req.Header.Set("Content-Type", "application/json")
		resp, err := f.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			b, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 403 for unknown scenario, got %d body=%s", resp.StatusCode, string(b))
		}
	})

	t.Run("viewer_denied_outside_grants", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/entities/"+foreign.ID.String()+"/ask", strings.NewReader(`{"question":"hi"}`))
		req.Header.Set(httpcontext.HeaderPrincipalUserID, viewer.String())
		req.Header.Set("Content-Type", "application/json")
		resp, err := f.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			b, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 403, got %d body=%s", resp.StatusCode, string(b))
		}
	})

	t.Run("viewer_include_related_does_not_leak_foreign_entity", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/entities/"+root.ID.String()+"/ask", strings.NewReader(`{"question":"summarize","include_related":true}`))
		req.Header.Set(httpcontext.HeaderPrincipalUserID, viewer.String())
		req.Header.Set("Content-Type", "application/json")
		resp, err := f.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		// If OpenAI is mocked correctly, this should succeed; otherwise test is skipped above.
		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 200, got %d body=%s", resp.StatusCode, string(b))
		}
		b, _ := io.ReadAll(resp.Body)
		var out map[string]any
		if err := json.Unmarshal(b, &out); err != nil {
			t.Fatalf("json: %v body=%s", err, string(b))
		}
		supporting, _ := out["supporting_entities"].([]any)
		for _, x := range supporting {
			m, _ := x.(map[string]any)
			if m["entity_id"] == foreign.ID.String() {
				t.Fatalf("leaked foreign supporting entity")
			}
		}
	})
}
