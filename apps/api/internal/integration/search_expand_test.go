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

func TestSearch_RelationExpand_NoLeakAndExpansion(t *testing.T) {
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

	defaultDomain := uuid.MustParse("32000000-0000-0000-0000-000000000001")
	isolatedDomain := uuid.MustParse("33000000-0000-0000-0000-000000000099")
	admin := uuid.MustParse("30000000-0000-0000-0000-000000000001")
	viewer := uuid.MustParse("30000000-0000-0000-0000-000000000002")
	policy := uuid.MustParse("31000000-0000-0000-0000-000000000001")

	_, err = pool.Exec(ctx, `
		INSERT INTO domains (id, name, owner_id, default_access_policy_id, status)
		VALUES ($1, 'isolated_search', $2, $3, 'active')
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

	repo := knowledge_core.NewEntityRepo(pool)

	entA, err := repo.Create(ctx, knowledge_core.CreateEntityInput{
		Type:             "Insight",
		Title:            "search_expand_A",
		OwnerID:          &viewer,
		DomainID:         defaultDomain,
		SensitivityLevel: 0,
		TruthMode:        "derived",
		LifecycleState:   "draft",
	})
	if err != nil {
		t.Fatal(err)
	}
	entB, err := repo.Create(ctx, knowledge_core.CreateEntityInput{
		Type:             "Insight",
		Title:            "search_expand_B_isolated",
		OwnerID:          &admin,
		DomainID:         isolatedDomain,
		SensitivityLevel: 0,
		TruthMode:        "derived",
		LifecycleState:   "draft",
	})
	if err != nil {
		t.Fatal(err)
	}
	entC, err := repo.Create(ctx, knowledge_core.CreateEntityInput{
		Type:             "Insight",
		Title:            "search_expand_C_same_domain",
		OwnerID:          &admin,
		DomainID:         defaultDomain,
		SensitivityLevel: 0,
		TruthMode:        "derived",
		LifecycleState:   "draft",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO entity_links (from_entity_id, from_entity_type, relation_type, to_entity_id, to_entity_type)
		VALUES ($1, 'Insight', 'related', $2, 'Insight')`, entA.ID, entB.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO entity_links (from_entity_id, from_entity_type, relation_type, to_entity_id, to_entity_type)
		VALUES ($1, 'Insight', 'see_also', $3, 'Insight')`, entA.ID, entC.ID)
	if err != nil {
		t.Fatal(err)
	}

	parseHits := func(body []byte) []struct {
		EntityID          string `json:"entity_id"`
		Title             string `json:"title"`
		RelationExpansion string `json:"relation_expansion"`
	} {
		var out struct {
			Hits []struct {
				EntityID          string `json:"entity_id"`
				Title             string `json:"title"`
				RelationExpansion string `json:"relation_expansion"`
			} `json:"hits"`
		}
		if err := json.Unmarshal(body, &out); err != nil {
			t.Fatalf("json: %v body=%s", err, string(body))
		}
		return out.Hits
	}

	t.Run("viewer_does_not_see_linked_entity_in_foreign_domain", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/search?owner_id="+viewer.String()+"&expand_relations=1", nil)
		req.Header.Set(httpcontext.HeaderPrincipalUserID, viewer.String())
		resp, err := f.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status %d", resp.StatusCode)
		}
		body, _ := io.ReadAll(resp.Body)
		hits := parseHits(body)
		for _, h := range hits {
			if h.EntityID == entB.ID.String() {
				t.Fatalf("leaked isolated entity B into viewer results")
			}
		}
		var sawA, sawC bool
		var cRel string
		for _, h := range hits {
			if h.EntityID == entA.ID.String() {
				sawA = true
			}
			if h.EntityID == entC.ID.String() {
				sawC = true
				cRel = h.RelationExpansion
			}
		}
		if !sawA {
			t.Fatal("expected primary hit A")
		}
		if !sawC {
			t.Fatal("expected expanded hit C in same domain via entity_link")
		}
		if cRel == "" {
			t.Fatal("expected relation_expansion to be set for expanded entity")
		}
	})

	t.Run("expand_off_returns_only_owner_rows", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/search?owner_id="+viewer.String(), nil)
		req.Header.Set(httpcontext.HeaderPrincipalUserID, viewer.String())
		resp, err := f.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		hits := parseHits(body)
		if len(hits) != 1 || hits[0].EntityID != entA.ID.String() {
			t.Fatalf("expected single hit A, got %+v", hits)
		}
	})
}
