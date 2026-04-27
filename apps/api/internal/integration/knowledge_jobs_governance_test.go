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
	"github.com/knowledgelayer/api/internal/ingestion_connectors"
	"github.com/knowledgelayer/api/internal/knowledge_jobs"
)

// Requires PostgreSQL with migrations + seed (DATABASE_URL). Enable with E2E_DB=1.
func TestKnowledgeJobsHTTP_listRequiresPrincipal(t *testing.T) {
	if os.Getenv("E2E_DB") == "" {
		t.Skip("set E2E_DB=1 and DATABASE_URL")
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Fatal("DATABASE_URL required")
	}
	if err := db.MigrateUp(dsn); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
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

	req := httptest.NewRequest(http.MethodGet, "/knowledge-jobs", nil)
	resp, err := f.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 401 got %d: %s", resp.StatusCode, b)
	}
}

func TestKnowledgeJobsHTTP_createRequiresManageJobs(t *testing.T) {
	if os.Getenv("E2E_DB") == "" {
		t.Skip("set E2E_DB=1 and DATABASE_URL")
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Fatal("DATABASE_URL required")
	}
	if err := db.MigrateUp(dsn); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
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
	domain := uuid.MustParse("32000000-0000-0000-0000-000000000001")
	viewer := uuid.MustParse("30000000-0000-0000-0000-000000000002")
	telegramConnector := uuid.MustParse("20000000-0000-0000-0000-000000000001")

	cfg, _ := json.Marshal(map[string]any{"bot_token": "fake", "allowed_chat_ids": []int64{-100123}})
	feed, err := deps.Ingestion.CreateSourceFeed(ctx, ingestion_connectors.CreateSourceFeedInput{
		ConnectorID:         telegramConnector,
		DisplayName:         "kj-gov-feed",
		OwnerID:             admin,
		DomainID:            domain,
		SensitivityLevel:    0,
		AllowedJobTypesJSON: json.RawMessage(`["weekly_digest"]`),
		IngestionMode:       "ingestion_only",
		ExternalRef:         "-100123",
		ConnectorConfigJSON: cfg,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := deps.Ingestion.Activate(ctx, feed.ID); err != nil {
		t.Fatal(err)
	}
	scope, _ := json.Marshal(map[string]uuid.UUID{"source_feed_id": feed.ID, "domain_id": domain})
	body := map[string]any{
		"name":                     "gov test job",
		"job_type":                 "weekly_digest",
		"owner_id":                 viewer.String(),
		"source_scope_json":        json.RawMessage(scope),
		"output_domain_id":         domain.String(),
		"output_sensitivity_level": 0,
		"review_required":          true,
		"publication_mode":         knowledge_jobs.PublicationModeReviewedPublish,
		"config_json":              map[string]any{},
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/knowledge-jobs", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(httpcontext.HeaderPrincipalUserID, viewer.String())
	resp, err := f.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 403 got %d: %s", resp.StatusCode, raw)
	}
}

func TestKnowledgeJobsHTTP_createRejectsInvalidScope(t *testing.T) {
	if os.Getenv("E2E_DB") == "" {
		t.Skip("set E2E_DB=1 and DATABASE_URL")
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Fatal("DATABASE_URL required")
	}
	if err := db.MigrateUp(dsn); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
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
	domain := uuid.MustParse("32000000-0000-0000-0000-000000000001")
	scope, _ := json.Marshal(map[string]uuid.UUID{"domain_id": domain})
	body := map[string]any{
		"name":                     "bad scope job",
		"job_type":                 "weekly_digest",
		"owner_id":                 admin.String(),
		"source_scope_json":        json.RawMessage(scope),
		"output_domain_id":         domain.String(),
		"output_sensitivity_level": 0,
		"review_required":          true,
		"publication_mode":         knowledge_jobs.PublicationModeReviewedPublish,
		"config_json":              map[string]any{},
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/knowledge-jobs", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(httpcontext.HeaderPrincipalUserID, admin.String())
	resp, err := f.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400 got %d: %s", resp.StatusCode, raw)
	}
}
