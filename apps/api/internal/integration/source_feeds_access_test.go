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
	"github.com/knowledgelayer/api/internal/ingestion_connectors"
)

func TestSourceFeeds_AccessScopeAndConfigOmission(t *testing.T) {
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
	telegramConnector := uuid.MustParse("20000000-0000-0000-0000-000000000001")

	// Create an isolated domain where viewer has no grant.
	isolatedDomain := uuid.MustParse("33000000-0000-0000-0000-000000000199")
	policy := uuid.MustParse("31000000-0000-0000-0000-000000000001")
	_, err = pool.Exec(ctx, `
		INSERT INTO domains (id, name, owner_id, default_access_policy_id, status)
		VALUES ($1, 'isolated_feeds', $2, $3, 'active')
		ON CONFLICT (id) DO NOTHING`, isolatedDomain, admin, policy)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO domain_grants (user_id, domain_id, access_level, sensitivity_cap)
		VALUES ($1, $2, 'admin', 3)
		ON CONFLICT (user_id, domain_id) DO NOTHING`, admin, isolatedDomain)
	if err != nil {
		t.Fatal(err)
	}

	cfg, _ := json.Marshal(map[string]string{"bot_token": "fake"})

	feedDefault, err := deps.Ingestion.CreateSourceFeed(ctx, ingestion_connectors.CreateSourceFeedInput{
		ConnectorID:         telegramConnector,
		DisplayName:         "access-default-domain",
		OwnerID:             admin,
		DomainID:            defaultDomain,
		SensitivityLevel:    0,
		AllowedJobTypesJSON: json.RawMessage(`["weekly_digest"]`),
		IngestionMode:       "ingestion_only",
		ExternalRef:         "-100123",
		ConnectorConfigJSON: cfg,
	})
	if err != nil {
		t.Fatal(err)
	}
	feedIsolated, err := deps.Ingestion.CreateSourceFeed(ctx, ingestion_connectors.CreateSourceFeedInput{
		ConnectorID:         telegramConnector,
		DisplayName:         "access-isolated-domain",
		OwnerID:             admin,
		DomainID:            isolatedDomain,
		SensitivityLevel:    0,
		AllowedJobTypesJSON: json.RawMessage(`["weekly_digest"]`),
		IngestionMode:       "ingestion_only",
		ExternalRef:         "-100123",
		ConnectorConfigJSON: cfg,
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("list_without_principal_401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/source-feeds", nil)
		resp, err := f.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.StatusCode)
		}
	})

	t.Run("viewer_list_scoped_to_granted_domains_and_no_config", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/source-feeds", nil)
		req.Header.Set(httpcontext.HeaderPrincipalUserID, viewer.String())
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
		for _, r := range rows {
			if r["domain_id"] == feedIsolated.DomainID.String() || r["id"] == feedIsolated.ID.String() {
				t.Fatalf("leaked isolated feed into viewer list")
			}
			if _, ok := r["connector_config_json"]; ok {
				t.Fatalf("connector_config_json should be omitted from list response")
			}
		}
	})

	t.Run("viewer_feed_detail_requires_manage_source_feed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/source-feeds/"+feedDefault.ID.String(), nil)
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

	t.Run("viewer_forbidden_feed_detail_outside_grants", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/source-feeds/"+feedIsolated.ID.String(), nil)
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

	t.Run("admin_can_view_feed_detail_with_config", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/source-feeds/"+feedDefault.ID.String(), nil)
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
		var out map[string]any
		if err := json.Unmarshal(b, &out); err != nil {
			t.Fatalf("json: %v body=%s", err, string(b))
		}
		if _, ok := out["connector_config_json"]; !ok {
			t.Fatalf("expected connector_config_json present for admin detail view")
		}
	})

	t.Run("viewer_cannot_trigger_source_sync", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/source-feeds/"+feedDefault.ID.String()+"/sync", nil)
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
}
