package integration

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/knowledgelayer/api/internal/app"
	"github.com/knowledgelayer/api/internal/config"
	"github.com/knowledgelayer/api/internal/db"
	"github.com/knowledgelayer/api/internal/ingestion_connectors"
	"github.com/knowledgelayer/api/internal/knowledge_jobs"
)

// Requires PostgreSQL with migrations + seed (DATABASE_URL). Enable with E2E_DB=1.
func TestWeeklyDigestEndToEnd(t *testing.T) {
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
	admin := uuid.MustParse("30000000-0000-0000-0000-000000000001")
	domain := uuid.MustParse("32000000-0000-0000-0000-000000000001")
	telegramConnector := uuid.MustParse("20000000-0000-0000-0000-000000000001")

	cfg, _ := json.Marshal(map[string]any{"bot_token": "fake", "allowed_chat_ids": []int64{-100123}})
	feed, err := deps.Ingestion.CreateSourceFeed(ctx, ingestion_connectors.CreateSourceFeedInput{
		ConnectorID:         telegramConnector,
		DisplayName:         "e2e-telegram",
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
	job, err := deps.Jobs.Create(ctx, knowledge_jobs.CreateJobInput{
		Name:              "e2e weekly digest",
		JobType:           "weekly_digest",
		OwnerID:           admin,
		SourceScopeJSON:   scope,
		OutputDomainID:    &domain,
		OutputSensitivity: 0,
		ReviewRequired:    true,
		ConfigJSON:        json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	run, err := deps.Jobs.Run(ctx, job.ID, admin)
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != "completed" {
		t.Fatalf("expected completed run, got %s", run.Status)
	}
	tasks, err := deps.Review.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) == 0 {
		t.Fatal("expected at least one review task")
	}
}
