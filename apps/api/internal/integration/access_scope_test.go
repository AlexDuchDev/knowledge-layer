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

// Requires PostgreSQL with migrations + seed (DATABASE_URL). Enable with E2E_DB=1.
func TestPhase0_EntityAccess_Scope(t *testing.T) {
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
	policy := uuid.MustParse("31000000-0000-0000-0000-000000000001")

	// Isolated domain: only admin has grant (not viewer).
	otherDomain := uuid.MustParse("33000000-0000-0000-0000-000000000099")
	_, err = pool.Exec(ctx, `
		INSERT INTO domains (id, name, owner_id, default_access_policy_id, status)
		VALUES ($1, 'isolated', $2, $3, 'active')
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
	title := "secret entity"
	ent, err := repo.Create(ctx, knowledge_core.CreateEntityInput{
		Type:             "Insight",
		Title:            title,
		OwnerID:          &admin,
		DomainID:         otherDomain,
		SensitivityLevel: 0,
		TruthMode:        "derived",
		LifecycleState:   "draft",
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("entities_without_principal_401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/entities", nil)
		resp, err := f.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.StatusCode)
		}
	})

	t.Run("viewer_cannot_read_entity_outside_grants", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/entities/"+ent.ID.String(), nil)
		req.Header.Set(httpcontext.HeaderPrincipalUserID, viewer.String())
		resp, err := f.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 403, got %d: %s", resp.StatusCode, body)
		}
	})

	t.Run("admin_can_read_entity", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/entities/"+ent.ID.String(), nil)
		req.Header.Set(httpcontext.HeaderPrincipalUserID, admin.String())
		resp, err := f.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
		}
	})

	t.Run("viewer_raw_artifact_forbidden_without_view_raw", func(t *testing.T) {
		// Viewer has view/search on default domain but seed does not grant view_raw to viewer role after migration 000009.
		defaultDomain := uuid.MustParse("32000000-0000-0000-0000-000000000001")
		tgConn := uuid.MustParse("20000000-0000-0000-0000-000000000001")
		cfg, _ := json.Marshal(map[string]string{"bot_token": "fake"})
		_, err = pool.Exec(ctx, `
			INSERT INTO source_feeds (id, connector_id, source_uri, display_name, owner_id, domain_id,
				sensitivity_level, allowed_job_types_json, ingestion_mode, connector_config_json, status)
			VALUES ($1, $2, '', 'scope-feed', $3, $4, 0, '["weekly_digest"]'::jsonb, 'ingestion_only', $5::jsonb, 'active')
			ON CONFLICT (id) DO NOTHING`,
			uuid.MustParse("33000000-0000-0000-0000-0000000000aa"), tgConn, admin, defaultDomain, cfg)
		if err != nil {
			t.Fatal(err)
		}
		feedID := uuid.MustParse("33000000-0000-0000-0000-0000000000aa")
		runID := uuid.New()
		_, _ = pool.Exec(ctx, `
			INSERT INTO ingestion_runs (id, source_feed_id, trigger_type, status, started_at)
			VALUES ($1, $2, 'manual', 'completed', now())`, runID, feedID)
		rawID := uuid.New()
		hash := "hash-scope-" + rawID.String()
		_, err = pool.Exec(ctx, `
			INSERT INTO raw_artifacts (id, source_feed_id, ingestion_run_id, artifact_type, storage_uri, content_hash, metadata_json)
			VALUES ($1, $2, $3, 'test', '', $4, '{}'::jsonb)`, rawID, feedID, runID, hash)
		if err != nil {
			t.Fatal(err)
		}

		req := httptest.NewRequest(http.MethodGet, "/raw-artifacts/"+rawID.String(), nil)
		req.Header.Set(httpcontext.HeaderPrincipalUserID, viewer.String())
		resp, err := f.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 403 for viewer raw, got %d: %s", resp.StatusCode, body)
		}
	})
}
