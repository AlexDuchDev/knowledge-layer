package integration

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/knowledgelayer/api/internal/app"
	"github.com/knowledgelayer/api/internal/config"
	"github.com/knowledgelayer/api/internal/db"
	"github.com/knowledgelayer/api/internal/httpcontext"
	"github.com/knowledgelayer/api/internal/httpserver"
	"github.com/knowledgelayer/api/internal/knowledge_core"
	"github.com/knowledgelayer/api/internal/review"
)

func TestGovernance_PublishGate(t *testing.T) {
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
	policy := uuid.MustParse("31000000-0000-0000-0000-000000000001")

	t.Run("viewer_forbidden_policy_exceptions_list", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/governance/policy-exceptions", nil)
		req.Header.Set(httpcontext.HeaderPrincipalUserID, viewer.String())
		resp, err := f.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", resp.StatusCode)
		}
	})

	t.Run("admin_ok_policy_exceptions_list", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/governance/policy-exceptions", nil)
		req.Header.Set(httpcontext.HeaderPrincipalUserID, admin.String())
		resp, err := f.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
	})

	// Domain scoping checks: admin has publish in defaultDomain, but only a read grant in otherDomain.
	// Governance queues must not include tasks from otherDomain.
	otherDomain := uuid.MustParse("33000000-0000-0000-0000-000000000155")
	_, err = pool.Exec(ctx, `
		INSERT INTO domains (id, name, owner_id, default_access_policy_id, status)
		VALUES ($1, 'other', $2, $3, 'active')
		ON CONFLICT (id) DO NOTHING`, otherDomain, admin, policy)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO domain_grants (user_id, domain_id, access_level, sensitivity_cap)
		VALUES ($1, $2, 'read', 3)
		ON CONFLICT (user_id, domain_id) DO NOTHING`, admin, otherDomain)
	if err != nil {
		t.Fatal(err)
	}

	repo := knowledge_core.NewEntityRepo(pool)
	entDefault, err := repo.Create(ctx, knowledge_core.CreateEntityInput{
		Type:             "Insight",
		Title:            "gov_default_entity",
		OwnerID:          &admin,
		DomainID:         defaultDomain,
		SensitivityLevel: 0,
		TruthMode:        "derived",
		LifecycleState:   "draft",
	})
	if err != nil {
		t.Fatal(err)
	}
	entOther, err := repo.Create(ctx, knowledge_core.CreateEntityInput{
		Type:             "Insight",
		Title:            "gov_other_entity",
		OwnerID:          &admin,
		DomainID:         otherDomain,
		SensitivityLevel: 0,
		TruthMode:        "derived",
		LifecycleState:   "draft",
	})
	if err != nil {
		t.Fatal(err)
	}

	rev := review.NewService(pool)
	past := time.Now().UTC().Add(-2 * time.Hour)
	_, err = rev.Create(ctx, "entity", entDefault.ID, admin, nil, &past)
	if err != nil {
		t.Fatal(err)
	}
	_, err = rev.Create(ctx, "entity", entOther.ID, admin, nil, &past)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("viewer_forbidden_overdue_and_approval", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/governance/reviews/overdue", nil)
		req.Header.Set(httpcontext.HeaderPrincipalUserID, viewer.String())
		resp, err := f.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", resp.StatusCode)
		}

		req2 := httptest.NewRequest(http.MethodGet, "/governance/approval-queue", nil)
		req2.Header.Set(httpcontext.HeaderPrincipalUserID, viewer.String())
		resp2, err := f.Test(req2)
		if err != nil {
			t.Fatal(err)
		}
		defer resp2.Body.Close()
		if resp2.StatusCode != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", resp2.StatusCode)
		}
	})

	t.Run("admin_overdue_scoped_to_publish_domains", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/governance/reviews/overdue", nil)
		req.Header.Set(httpcontext.HeaderPrincipalUserID, admin.String())
		resp, err := f.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		b, _ := io.ReadAll(resp.Body)
		var rows []map[string]any
		if err := json.Unmarshal(b, &rows); err != nil {
			t.Fatalf("json: %v body=%s", err, string(b))
		}
		var sawDefault bool
		for _, r := range rows {
			if r["target_id"] == entOther.ID.String() {
				t.Fatalf("leaked other-domain entity into overdue queue")
			}
			if r["target_id"] == entDefault.ID.String() {
				sawDefault = true
			}
		}
		if !sawDefault {
			t.Fatal("expected default-domain overdue task present")
		}
	})

	t.Run("admin_approval_queue_scoped_to_publish_domains", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/governance/approval-queue", nil)
		req.Header.Set(httpcontext.HeaderPrincipalUserID, admin.String())
		resp, err := f.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		b, _ := io.ReadAll(resp.Body)
		var rows []map[string]any
		if err := json.Unmarshal(b, &rows); err != nil {
			t.Fatalf("json: %v body=%s", err, string(b))
		}
		var sawDefault bool
		for _, r := range rows {
			if r["target_id"] == entOther.ID.String() {
				t.Fatalf("leaked other-domain entity into approval queue")
			}
			if r["target_id"] == entDefault.ID.String() {
				sawDefault = true
			}
		}
		if !sawDefault {
			t.Fatal("expected default-domain review task present")
		}
	})
}
