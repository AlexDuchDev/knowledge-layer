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
	"github.com/knowledgelayer/api/internal/identity_access"
	"github.com/knowledgelayer/api/internal/ingestion_connectors"
	"github.com/knowledgelayer/api/internal/knowledge_core"
	"github.com/knowledgelayer/api/internal/knowledge_jobs"
)

// Requires PostgreSQL with migrations + seed (DATABASE_URL). Enable with E2E_DB=1.
func TestPermissionFoundation_DomainEntityACLAndSearch(t *testing.T) {
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
	financeUser := uuid.MustParse("f0000001-0001-0001-0001-000000000001")
	marketingUser := uuid.MustParse("f0000002-0002-0002-0002-000000000002")
	financeDomain := uuid.MustParse("34000000-0000-0000-0000-0000000000f1")
	marketingDomain := uuid.MustParse("34000000-0000-0000-0000-0000000000f2")
	narrowPolicy := uuid.MustParse("35000000-0000-0000-0000-0000000000a1")
	broadPolicy := uuid.MustParse("35000000-0000-0000-0000-0000000000a2")
	analystRole := uuid.MustParse("10000000-0000-0000-0000-000000000002")

	_, err = pool.Exec(ctx, `
		INSERT INTO access_policies (id, name, description, status, domain_id, entity_type_scope)
		VALUES ($1,'perm_narrow','integration','active',$2,'Policy')
		ON CONFLICT (id) DO UPDATE SET entity_type_scope = EXCLUDED.entity_type_scope, domain_id = EXCLUDED.domain_id`,
		narrowPolicy, financeDomain)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO access_policies (id, name, description, status, domain_id, entity_type_scope)
		VALUES ($1,'perm_broad','integration','active',$2,NULL)
		ON CONFLICT (id) DO UPDATE SET entity_type_scope = EXCLUDED.entity_type_scope, domain_id = EXCLUDED.domain_id`,
		broadPolicy, marketingDomain)
	if err != nil {
		t.Fatal(err)
	}

	for _, pair := range []struct {
		id     uuid.UUID
		name   string
		policy uuid.UUID
	}{
		{financeDomain, "perm_finance", narrowPolicy},
		{marketingDomain, "perm_marketing", broadPolicy},
	} {
		_, err = pool.Exec(ctx, `
			INSERT INTO domains (id, name, owner_id, default_access_policy_id, status)
			VALUES ($1, $2, $3, $4, 'active')
			ON CONFLICT (id) DO NOTHING`, pair.id, pair.name, admin, pair.policy)
		if err != nil {
			t.Fatal(err)
		}
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO users (id, email, name, status) VALUES ($1,'finance_perm@test','Finance User','active')
		ON CONFLICT (id) DO NOTHING`, financeUser)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO users (id, email, name, status) VALUES ($1,'marketing_perm@test','Marketing User','active')
		ON CONFLICT (id) DO NOTHING`, marketingUser)
	if err != nil {
		t.Fatal(err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO domain_grants (user_id, domain_id, access_level, sensitivity_cap)
		VALUES ($1, $2, 'write', 4)
		ON CONFLICT (user_id, domain_id) DO UPDATE SET access_level = EXCLUDED.access_level, sensitivity_cap = EXCLUDED.sensitivity_cap`,
		financeUser, financeDomain)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO domain_grants (user_id, domain_id, access_level, sensitivity_cap)
		VALUES ($1, $2, 'write', 4)
		ON CONFLICT (user_id, domain_id) DO UPDATE SET access_level = EXCLUDED.access_level, sensitivity_cap = EXCLUDED.sensitivity_cap`,
		marketingUser, marketingDomain)
	if err != nil {
		t.Fatal(err)
	}

	for _, u := range []uuid.UUID{financeUser, marketingUser} {
		for _, dom := range []uuid.UUID{financeDomain, marketingDomain} {
			_, err = pool.Exec(ctx, `
				INSERT INTO user_role_bindings (user_id, role_id, scope_type, scope_id)
				SELECT $1, $2, 'domain', $3
				WHERE NOT EXISTS (
					SELECT 1 FROM user_role_bindings
					WHERE user_id = $1 AND role_id = $2 AND scope_type = 'domain' AND scope_id = $3
				)`, u, analystRole, dom)
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	repo := knowledge_core.NewEntityRepo(pool)
	financeEnt, err := repo.Create(ctx, knowledge_core.CreateEntityInput{
		Type:             "Policy",
		Title:            "finance_secret_title",
		OwnerID:          &admin,
		DomainID:         financeDomain,
		SensitivityLevel: identity_access.SensitivityPublicInternal,
		TruthMode:        "derived",
		LifecycleState:   "draft",
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("finance_user_can_view_finance_entity", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/entities/"+financeEnt.ID.String(), nil)
		req.Header.Set(httpcontext.HeaderPrincipalUserID, financeUser.String())
		resp, err := f.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 200 got %d: %s", resp.StatusCode, b)
		}
	})

	t.Run("marketing_user_cannot_view_finance_entity", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/entities/"+financeEnt.ID.String(), nil)
		req.Header.Set(httpcontext.HeaderPrincipalUserID, marketingUser.String())
		resp, err := f.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			b, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 403 got %d: %s", resp.StatusCode, b)
		}
	})

	insightEnt, err := repo.Create(ctx, knowledge_core.CreateEntityInput{
		Type:             "Insight",
		Title:            "blocked_by_type_scope",
		OwnerID:          &admin,
		DomainID:         financeDomain,
		SensitivityLevel: identity_access.SensitivityPublicInternal,
		TruthMode:        "derived",
		LifecycleState:   "draft",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO entity_acl (entity_id, principal_type, principal_id, effect)
		VALUES ($1,'user',$2,'allow')
		ON CONFLICT (entity_id, principal_type, principal_id) DO UPDATE SET effect = EXCLUDED.effect`,
		insightEnt.ID, financeUser)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("entity_acl_allow_bypasses_type_policy_for_view", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/entities/"+insightEnt.ID.String(), nil)
		req.Header.Set(httpcontext.HeaderPrincipalUserID, financeUser.String())
		resp, err := f.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 200 got %d: %s", resp.StatusCode, b)
		}
	})

	insightNoACL, err := repo.Create(ctx, knowledge_core.CreateEntityInput{
		Type:             "Insight",
		Title:            "insight_no_acl_title",
		OwnerID:          &admin,
		DomainID:         financeDomain,
		SensitivityLevel: identity_access.SensitivityPublicInternal,
		TruthMode:        "derived",
		LifecycleState:   "draft",
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("entity_type_blocks_insight_without_acl", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/entities/"+insightNoACL.ID.String(), nil)
		req.Header.Set(httpcontext.HeaderPrincipalUserID, financeUser.String())
		resp, err := f.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			b, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 403 got %d: %s", resp.StatusCode, b)
		}
	})

	t.Run("search_hides_insight_blocked_by_type_policy", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/search?domain_id="+financeDomain.String()+"&q=insight_no_acl", nil)
		req.Header.Set(httpcontext.HeaderPrincipalUserID, financeUser.String())
		resp, err := f.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 200 got %d: %s", resp.StatusCode, b)
		}
		body, _ := io.ReadAll(resp.Body)
		if stringContains(string(body), "insight_no_acl_title") {
			t.Fatalf("search leaked type-blocked entity: %s", body)
		}
	})

	_, err = pool.Exec(ctx, `
		INSERT INTO entity_acl (entity_id, principal_type, principal_id, effect)
		VALUES ($1,'user',$2,'deny')
		ON CONFLICT (entity_id, principal_type, principal_id) DO UPDATE SET effect = EXCLUDED.effect`,
		financeEnt.ID, financeUser)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("search_hides_entity_denied_by_acl", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/search?domain_id="+financeDomain.String()+"&q=finance_secret", nil)
		req.Header.Set(httpcontext.HeaderPrincipalUserID, financeUser.String())
		resp, err := f.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 200 got %d: %s", resp.StatusCode, b)
		}
		body, _ := io.ReadAll(resp.Body)
		if stringContains(string(body), "finance_secret_title") {
			t.Fatalf("search leaked denied entity title: %s", body)
		}
	})

	_, _ = pool.Exec(ctx, `DELETE FROM entity_acl WHERE entity_id = $1 AND principal_id = $2`, financeEnt.ID, financeUser)

	strictEnt, err := repo.Create(ctx, knowledge_core.CreateEntityInput{
		Type:             "Insight",
		Title:            "strictly_conf_entity",
		OwnerID:          &admin,
		DomainID:         financeDomain,
		SensitivityLevel: identity_access.SensitivityStrictlyConfidential,
		TruthMode:        "derived",
		LifecycleState:   "draft",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = pool.Exec(ctx, `
		UPDATE domain_grants SET sensitivity_cap = 2 WHERE user_id = $1 AND domain_id = $2`,
		financeUser, financeDomain)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("sensitivity_cap_blocks_strictly_confidential", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/entities/"+strictEnt.ID.String(), nil)
		req.Header.Set(httpcontext.HeaderPrincipalUserID, financeUser.String())
		resp, err := f.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			b, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 403 got %d: %s", resp.StatusCode, b)
		}
	})

	leaderEnt, err := repo.Create(ctx, knowledge_core.CreateEntityInput{
		Type:             "Policy",
		Title:            "leadership_restricted_title",
		OwnerID:          &admin,
		DomainID:         financeDomain,
		SensitivityLevel: identity_access.SensitivityLeadershipRestricted,
		TruthMode:        "derived",
		LifecycleState:   "draft",
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("sensitivity_cap_blocks_leadership_restricted", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/entities/"+leaderEnt.ID.String(), nil)
		req.Header.Set(httpcontext.HeaderPrincipalUserID, financeUser.String())
		resp, err := f.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			b, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 403 got %d: %s", resp.StatusCode, b)
		}
	})
}

func stringContains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && findSubstring(s, sub))
}

func findSubstring(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestPermissionFoundation_JobUndeclaredSourceFeed(t *testing.T) {
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
	feedA, err := deps.Ingestion.CreateSourceFeed(ctx, ingestion_connectors.CreateSourceFeedInput{
		ConnectorID:         telegramConnector,
		DisplayName:         "perm-feed-a",
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
	feedB, err := deps.Ingestion.CreateSourceFeed(ctx, ingestion_connectors.CreateSourceFeedInput{
		ConnectorID:         telegramConnector,
		DisplayName:         "perm-feed-b",
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
	_ = deps.Ingestion.Activate(ctx, feedA.ID)
	_ = deps.Ingestion.Activate(ctx, feedB.ID)

	scope, _ := json.Marshal(map[string]uuid.UUID{"source_feed_id": feedA.ID, "domain_id": domain})
	job, err := deps.Jobs.Create(ctx, knowledge_jobs.CreateJobInput{
		Name:              "perm digest job",
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
	_, err = pool.Exec(ctx, `DELETE FROM knowledge_job_sources WHERE knowledge_job_id = $1`, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO knowledge_job_sources (knowledge_job_id, source_type, source_id)
		VALUES ($1,'source_feed',$2)`, job.ID, feedB.ID)
	if err != nil {
		t.Fatal(err)
	}

	_, err = deps.Jobs.Run(ctx, job.ID, admin)
	if err == nil {
		t.Fatal("expected error when job declares wrong source feed")
	}
}

func TestPermissionFoundation_JobRunDeniedNonOperator(t *testing.T) {
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
	domain := uuid.MustParse("32000000-0000-0000-0000-000000000001")
	telegramConnector := uuid.MustParse("20000000-0000-0000-0000-000000000001")

	cfg, _ := json.Marshal(map[string]any{"bot_token": "fake", "allowed_chat_ids": []int64{-100123}})
	feed, err := deps.Ingestion.CreateSourceFeed(ctx, ingestion_connectors.CreateSourceFeedInput{
		ConnectorID:         telegramConnector,
		DisplayName:         "perm-job-run-feed",
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
		Name:              "perm operator job",
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

	req := httptest.NewRequest(http.MethodPost, "/knowledge-jobs/"+job.ID.String()+"/run", nil)
	req.Header.Set(httpcontext.HeaderPrincipalUserID, viewer.String())
	resp, err := f.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 403 got %d: %s", resp.StatusCode, b)
	}
}
