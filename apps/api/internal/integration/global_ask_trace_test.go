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

func TestGlobalAsk_PersistsTraceRetrievalMode(t *testing.T) {
	if os.Getenv("E2E_DB") == "" {
		t.Skip("set E2E_DB=1 and DATABASE_URL")
	}
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
	defaultDomain := uuid.MustParse("32000000-0000-0000-0000-000000000001")

	repo := knowledge_core.NewEntityRepo(pool)
	_, err = repo.Create(ctx, knowledge_core.CreateEntityInput{
		Type:             "Insight",
		Title:            "global_ask_trace_keyword",
		OwnerID:          &admin,
		DomainID:         defaultDomain,
		SensitivityLevel: 0,
		TruthMode:        "derived",
		LifecycleState:   "draft",
		Body:             strPtr("Some searchable body text for global ask trace."),
	})
	if err != nil {
		t.Fatal(err)
	}

	body := `{"question":"searchable body text","retrieval_mode":"keyword_only"}`
	req := httptest.NewRequest(http.MethodPost, "/ask", strings.NewReader(body))
	req.Header.Set(httpcontext.HeaderPrincipalUserID, admin.String())
	req.Header.Set("Content-Type", "application/json")
	resp, err := f.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}
	b, _ := io.ReadAll(resp.Body)
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("json: %v", err)
	}
	tid, _ := out["trace_id"].(string)
	traceID, err := uuid.Parse(tid)
	if err != nil {
		t.Fatalf("trace_id: %v", err)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/answer-traces/"+traceID.String(), nil)
	req2.Header.Set(httpcontext.HeaderPrincipalUserID, admin.String())
	resp2, err := f.Test(req2)
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		b2, _ := io.ReadAll(resp2.Body)
		t.Fatalf("get trace: %d %s", resp2.StatusCode, string(b2))
	}
	b2, _ := io.ReadAll(resp2.Body)
	var row map[string]any
	if err := json.Unmarshal(b2, &row); err != nil {
		t.Fatal(err)
	}
	if row["retrieval_mode"] != "keyword_only" {
		t.Fatalf("expected retrieval_mode keyword_only, got %v", row["retrieval_mode"])
	}
	cj := row["citations_json"]
	if cj == nil {
		t.Fatal("missing citations_json")
	}
}

func strPtr(s string) *string { return &s }
