package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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

func jobBuilderTestDeps(t *testing.T) (*fiber.App, *app.Deps, func()) {
	t.Helper()
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
	deps, err := app.NewDeps(pool, config.Config{})
	if err != nil {
		t.Fatal(err)
	}
	f := fiber.New()
	httpserver.Mount(f, deps)
	return f, deps, func() { pool.Close() }
}

func jobBuilderCreateFeedAndJob(t *testing.T, f *fiber.App, deps *app.Deps, owner uuid.UUID) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	admin := uuid.MustParse("30000000-0000-0000-0000-000000000001")
	domain := uuid.MustParse("32000000-0000-0000-0000-000000000001")
	telegramConnector := uuid.MustParse("20000000-0000-0000-0000-000000000001")
	ext := fmt.Sprintf("jb-%s", uuid.NewString()[:8])
	cfg, _ := json.Marshal(map[string]any{"bot_token": "fake", "allowed_chat_ids": []int64{-100123}})
	feed, err := deps.Ingestion.CreateSourceFeed(ctx, ingestion_connectors.CreateSourceFeedInput{
		ConnectorID:         telegramConnector,
		DisplayName:         "jb-feed-" + ext,
		OwnerID:             admin,
		DomainID:            domain,
		SensitivityLevel:    0,
		AllowedJobTypesJSON: json.RawMessage(`["weekly_digest"]`),
		IngestionMode:       "ingestion_only",
		ExternalRef:         ext,
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
		"name":                     "job builder integration " + ext,
		"job_type":                 "weekly_digest",
		"owner_id":                 owner.String(),
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
	res, err := f.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create job: %d %s", res.StatusCode, raw)
	}
	var created struct {
		ID uuid.UUID `json:"id"`
	}
	if err := json.Unmarshal(raw, &created); err != nil {
		t.Fatal(err)
	}
	return created.ID
}

func TestJobBuilderHTTP_clonePreviewDryRun(t *testing.T) {
	f, deps, cleanup := jobBuilderTestDeps(t)
	defer cleanup()
	admin := uuid.MustParse("30000000-0000-0000-0000-000000000001")
	jobID := jobBuilderCreateFeedAndJob(t, f, deps, admin)

	cloneReq := httptest.NewRequest(http.MethodPost, "/knowledge-jobs/"+jobID.String()+"/clone", nil)
	cloneReq.Header.Set(httpcontext.HeaderPrincipalUserID, admin.String())
	res, err := f.Test(cloneReq)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("clone: %d %s", res.StatusCode, raw)
	}
	var cloned struct {
		ID              uuid.UUID  `json:"id"`
		ClonedFromJobID *uuid.UUID `json:"cloned_from_job_id"`
	}
	if err := json.Unmarshal(raw, &cloned); err != nil {
		t.Fatal(err)
	}
	if cloned.ID == jobID {
		t.Fatal("expected new id")
	}
	if cloned.ClonedFromJobID == nil || *cloned.ClonedFromJobID != jobID {
		t.Fatalf("cloned_from_job_id: %+v", cloned.ClonedFromJobID)
	}

	pvReq := httptest.NewRequest(http.MethodGet, "/knowledge-jobs/"+jobID.String()+"/preview", nil)
	pvReq.Header.Set(httpcontext.HeaderPrincipalUserID, admin.String())
	res2, err := f.Test(pvReq)
	if err != nil {
		t.Fatal(err)
	}
	defer res2.Body.Close()
	if res2.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(res2.Body)
		t.Fatalf("preview: %d %s", res2.StatusCode, b)
	}

	dryBody := []byte(`{"dry_run":true}`)
	dryReq := httptest.NewRequest(http.MethodPost, "/knowledge-jobs/"+jobID.String()+"/test-run", bytes.NewReader(dryBody))
	dryReq.Header.Set("Content-Type", "application/json")
	dryReq.Header.Set(httpcontext.HeaderPrincipalUserID, admin.String())
	res3, err := f.Test(dryReq)
	if err != nil {
		t.Fatal(err)
	}
	defer res3.Body.Close()
	b3, _ := io.ReadAll(res3.Body)
	if res3.StatusCode != http.StatusOK {
		t.Fatalf("dry run: %d %s", res3.StatusCode, b3)
	}
	var dry struct {
		Valid   bool            `json:"valid"`
		Preview json.RawMessage `json:"preview"`
	}
	if err := json.Unmarshal(b3, &dry); err != nil {
		t.Fatal(err)
	}
	if !dry.Valid || len(dry.Preview) == 0 {
		t.Fatalf("unexpected dry payload: %+v", dry)
	}
}

func TestJobBuilderHTTP_ownerCanCreateTrigger(t *testing.T) {
	f, deps, cleanup := jobBuilderTestDeps(t)
	defer cleanup()
	viewer := uuid.MustParse("30000000-0000-0000-0000-000000000002")
	jobID := jobBuilderCreateFeedAndJob(t, f, deps, viewer)
	sched := "0 7 * * *"
	tBody, _ := json.Marshal(map[string]any{"trigger_type": "scheduled", "schedule_expr": sched})
	req := httptest.NewRequest(http.MethodPost, "/knowledge-jobs/"+jobID.String()+"/triggers", bytes.NewReader(tBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(httpcontext.HeaderPrincipalUserID, viewer.String())
	res, err := f.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusCreated && res.StatusCode != http.StatusOK {
		t.Fatalf("owner trigger: %d %s", res.StatusCode, raw)
	}
}

func TestJobBuilderHTTP_scenarioBindingsRoundTrip(t *testing.T) {
	f, deps, cleanup := jobBuilderTestDeps(t)
	defer cleanup()
	admin := uuid.MustParse("30000000-0000-0000-0000-000000000001")
	jobID := jobBuilderCreateFeedAndJob(t, f, deps, admin)

	scCode := "jb_scn_" + uuid.NewString()[:8]
	scBody := map[string]any{
		"code":                scCode,
		"name":                "JB scenario " + scCode,
		"scenario_type":       "ask",
		"output_mode":         "ui_response",
		"output_policy":       map[string]any{"publication_mode": "draft"},
		"input_scope_json":    map[string]any{"unrestricted": true, "inherit_user_retrieval_scope": true},
		"trigger_type":        "interactive",
		"trigger_config_json": map[string]any{},
		"processing_mode":     "ask",
		"config_json":         map[string]any{"allow_unrestricted_input_scope": true},
	}
	sb, _ := json.Marshal(scBody)
	sreq := httptest.NewRequest(http.MethodPost, "/scenarios", bytes.NewReader(sb))
	sreq.Header.Set("Content-Type", "application/json")
	sreq.Header.Set(httpcontext.HeaderPrincipalUserID, admin.String())
	sres, err := f.Test(sreq)
	if err != nil {
		t.Fatal(err)
	}
	defer sres.Body.Close()
	sraw, _ := io.ReadAll(sres.Body)
	if sres.StatusCode != http.StatusCreated {
		t.Fatalf("create scenario: %d %s", sres.StatusCode, sraw)
	}
	var swrap struct {
		ID uuid.UUID `json:"id"`
	}
	if err := json.Unmarshal(sraw, &swrap); err != nil {
		t.Fatal(err)
	}

	bind := []map[string]string{{"scenario_id": swrap.ID.String(), "relationship": "supports"}}
	bb, _ := json.Marshal(bind)
	breq := httptest.NewRequest(http.MethodPost, "/knowledge-jobs/"+jobID.String()+"/scenario-bindings", bytes.NewReader(bb))
	breq.Header.Set("Content-Type", "application/json")
	breq.Header.Set(httpcontext.HeaderPrincipalUserID, admin.String())
	bres, err := f.Test(breq)
	if err != nil {
		t.Fatal(err)
	}
	defer bres.Body.Close()
	if bres.StatusCode != http.StatusNoContent {
		br, _ := io.ReadAll(bres.Body)
		t.Fatalf("scenario-bindings: %d %s", bres.StatusCode, br)
	}

	pvReq := httptest.NewRequest(http.MethodGet, "/knowledge-jobs/"+jobID.String()+"/preview", nil)
	pvReq.Header.Set(httpcontext.HeaderPrincipalUserID, admin.String())
	pres, err := f.Test(pvReq)
	if err != nil {
		t.Fatal(err)
	}
	defer pres.Body.Close()
	pr, _ := io.ReadAll(pres.Body)
	if pres.StatusCode != http.StatusOK {
		t.Fatalf("preview after bind: %d %s", pres.StatusCode, pr)
	}
	var pv map[string]any
	if err := json.Unmarshal(pr, &pv); err != nil {
		t.Fatal(err)
	}
	sbinds, _ := pv["scenario_bindings"].([]any)
	if len(sbinds) != 1 {
		t.Fatalf("expected 1 scenario binding, got %v", pv["scenario_bindings"])
	}
}
