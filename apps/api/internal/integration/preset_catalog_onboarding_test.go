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
)

func presetOnboardTestApp(t *testing.T) (*fiber.App, func()) {
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
	return f, func() { pool.Close() }
}

func TestPresetCatalogHTTP_listDetailRelatedInstantiateRole(t *testing.T) {
	f, cleanup := presetOnboardTestApp(t)
	defer cleanup()
	admin := uuid.MustParse("30000000-0000-0000-0000-000000000001")

	req := httptest.NewRequest(http.MethodGet, "/api/presets?type=role", nil)
	req.Header.Set(httpcontext.HeaderPrincipalUserID, admin.String())
	res, err := f.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("list presets: %d %s", res.StatusCode, raw)
	}
	var rows []map[string]any
	if err := json.Unmarshal(raw, &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 {
		t.Fatal("expected role catalog rows")
	}
	var entryID string
	for _, r := range rows {
		if r["code"] == "platform_admin" {
			id, _ := r["id"].(string)
			entryID = id
			break
		}
	}
	if entryID == "" {
		t.Fatal("platform_admin catalog entry missing")
	}

	dreq := httptest.NewRequest(http.MethodGet, "/api/presets/"+entryID, nil)
	dreq.Header.Set(httpcontext.HeaderPrincipalUserID, admin.String())
	dres, err := f.Test(dreq)
	if err != nil {
		t.Fatal(err)
	}
	defer dres.Body.Close()
	dbody, _ := io.ReadAll(dres.Body)
	if dres.StatusCode != http.StatusOK {
		t.Fatalf("detail: %d %s", dres.StatusCode, dbody)
	}

	rreq := httptest.NewRequest(http.MethodGet, "/api/presets/"+entryID+"/related", nil)
	rreq.Header.Set(httpcontext.HeaderPrincipalUserID, admin.String())
	rres, err := f.Test(rreq)
	if err != nil {
		t.Fatal(err)
	}
	defer rres.Body.Close()
	if rres.StatusCode != http.StatusOK {
		rb, _ := io.ReadAll(rres.Body)
		t.Fatalf("related: %d %s", rres.StatusCode, rb)
	}

	ext := uuid.NewString()[:8]
	instBody, _ := json.Marshal(map[string]any{
		"name": "Catalog test role " + ext,
		"code": "cat_test_role_" + ext,
	})
	ireq := httptest.NewRequest(http.MethodPost, "/api/presets/"+entryID+"/instantiate", bytes.NewReader(instBody))
	ireq.Header.Set("Content-Type", "application/json")
	ireq.Header.Set(httpcontext.HeaderPrincipalUserID, admin.String())
	ires, err := f.Test(ireq)
	if err != nil {
		t.Fatal(err)
	}
	defer ires.Body.Close()
	iraw, _ := io.ReadAll(ires.Body)
	if ires.StatusCode != http.StatusCreated {
		t.Fatalf("instantiate: %d %s", ires.StatusCode, iraw)
	}
}

func TestOnboardingHTTP_previewLaunchWithoutJobPreset(t *testing.T) {
	f, cleanup := presetOnboardTestApp(t)
	defer cleanup()
	admin := uuid.MustParse("30000000-0000-0000-0000-000000000001")

	lreq := httptest.NewRequest(http.MethodGet, "/api/presets?type=role", nil)
	lreq.Header.Set(httpcontext.HeaderPrincipalUserID, admin.String())
	lres, err := f.Test(lreq)
	if err != nil {
		t.Fatal(err)
	}
	defer lres.Body.Close()
	lraw, _ := io.ReadAll(lres.Body)
	if lres.StatusCode != http.StatusOK {
		t.Fatalf("list role presets: %d %s", lres.StatusCode, lraw)
	}
	var roleRows []map[string]any
	_ = json.Unmarshal(lraw, &roleRows)
	var roleID string
	for _, r := range roleRows {
		if r["code"] == "platform_admin" {
			roleID, _ = r["id"].(string)
			break
		}
	}
	sreq := httptest.NewRequest(http.MethodGet, "/api/presets?type=scenario", nil)
	sreq.Header.Set(httpcontext.HeaderPrincipalUserID, admin.String())
	sres, err := f.Test(sreq)
	if err != nil {
		t.Fatal(err)
	}
	defer sres.Body.Close()
	sraw, _ := io.ReadAll(sres.Body)
	if sres.StatusCode != http.StatusOK {
		t.Fatalf("list scenario presets: %d %s", sres.StatusCode, sraw)
	}
	var scenRows []map[string]any
	_ = json.Unmarshal(sraw, &scenRows)
	var scenID string
	for _, r := range scenRows {
		if r["code"] == "ask_allowed_knowledge" {
			scenID, _ = r["id"].(string)
			break
		}
	}
	if roleID == "" || scenID == "" {
		t.Fatal("catalog seed missing platform_admin or ask_allowed_knowledge")
	}

	cr := httptest.NewRequest(http.MethodPost, "/api/onboarding/sessions", nil)
	cr.Header.Set(httpcontext.HeaderPrincipalUserID, admin.String())
	cres, err := f.Test(cr)
	if err != nil {
		t.Fatal(err)
	}
	defer cres.Body.Close()
	cbody, _ := io.ReadAll(cres.Body)
	if cres.StatusCode != http.StatusCreated {
		t.Fatalf("create session: %d %s", cres.StatusCode, cbody)
	}
	var sess struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(cbody, &sess); err != nil {
		t.Fatal(err)
	}
	sid := sess.ID

	patch, _ := json.Marshal(map[string]any{
		"assignment": map[string]any{
			"initial_admin_user_id": admin.String(),
		},
		"selected_presets": []map[string]any{
			{"preset_catalog_entry_id": roleID, "slot": "role_0"},
			{"preset_catalog_entry_id": scenID, "slot": "scenario_0"},
		},
	})
	pr := httptest.NewRequest(http.MethodPatch, "/api/onboarding/sessions/"+sid, bytes.NewReader(patch))
	pr.Header.Set("Content-Type", "application/json")
	pr.Header.Set(httpcontext.HeaderPrincipalUserID, admin.String())
	pres, err := f.Test(pr)
	if err != nil {
		t.Fatal(err)
	}
	defer pres.Body.Close()
	pb, _ := io.ReadAll(pres.Body)
	if pres.StatusCode != http.StatusOK {
		t.Fatalf("patch: %d %s", pres.StatusCode, pb)
	}

	st, _ := json.Marshal(map[string]any{"template_code": "minimal"})
	sr := httptest.NewRequest(http.MethodPost, "/api/onboarding/sessions/"+sid+"/select-template", bytes.NewReader(st))
	sr.Header.Set("Content-Type", "application/json")
	sr.Header.Set(httpcontext.HeaderPrincipalUserID, admin.String())
	sres, errSel := f.Test(sr)
	if errSel != nil {
		t.Fatal(errSel)
	}
	defer sres.Body.Close()
	sb, _ := io.ReadAll(sres.Body)
	if sres.StatusCode != http.StatusOK {
		t.Fatalf("select-template: %d %s", sres.StatusCode, sb)
	}

	// Replace selections so launch skips weekly_digest (needs source feed scope).
	patch2, _ := json.Marshal(map[string]any{
		"selected_presets": []map[string]any{
			{"preset_catalog_entry_id": roleID, "slot": "role_0"},
			{"preset_catalog_entry_id": scenID, "slot": "scenario_0"},
		},
	})
	pr2 := httptest.NewRequest(http.MethodPatch, "/api/onboarding/sessions/"+sid, bytes.NewReader(patch2))
	pr2.Header.Set("Content-Type", "application/json")
	pr2.Header.Set(httpcontext.HeaderPrincipalUserID, admin.String())
	pres2, errPatch2 := f.Test(pr2)
	if errPatch2 != nil {
		t.Fatal(errPatch2)
	}
	defer pres2.Body.Close()
	if pres2.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(pres2.Body)
		t.Fatalf("patch2: %d %s", pres2.StatusCode, b)
	}

	pv := httptest.NewRequest(http.MethodPost, "/api/onboarding/sessions/"+sid+"/preview", nil)
	pv.Header.Set(httpcontext.HeaderPrincipalUserID, admin.String())
	pvres, errPv := f.Test(pv)
	if errPv != nil {
		t.Fatal(errPv)
	}
	defer pvres.Body.Close()
	pvb, _ := io.ReadAll(pvres.Body)
	if pvres.StatusCode != http.StatusOK {
		t.Fatalf("preview: %d %s", pvres.StatusCode, pvb)
	}
	var preview struct {
		ValidationIssues []string `json:"validation_issues"`
	}
	if err := json.Unmarshal(pvb, &preview); err != nil {
		t.Fatal(err)
	}
	if len(preview.ValidationIssues) > 0 {
		t.Fatalf("unexpected validation issues: %v", preview.ValidationIssues)
	}

	lr := httptest.NewRequest(http.MethodPost, "/api/onboarding/sessions/"+sid+"/launch", nil)
	lr.Header.Set(httpcontext.HeaderPrincipalUserID, admin.String())
	lres, errLaunch := f.Test(lr)
	if errLaunch != nil {
		t.Fatal(errLaunch)
	}
	defer lres.Body.Close()
	lb, _ := io.ReadAll(lres.Body)
	if lres.StatusCode != http.StatusOK {
		t.Fatalf("launch: %d %s", lres.StatusCode, lb)
	}
	var launched struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(lb, &launched); err != nil {
		t.Fatal(err)
	}
	if launched.Status != "launched" {
		t.Fatalf("status: %q", launched.Status)
	}

	lr2 := httptest.NewRequest(http.MethodPost, "/api/onboarding/sessions/"+sid+"/launch", nil)
	lr2.Header.Set(httpcontext.HeaderPrincipalUserID, admin.String())
	lres2, errLaunch2 := f.Test(lr2)
	if errLaunch2 != nil {
		t.Fatal(errLaunch2)
	}
	defer lres2.Body.Close()
	if lres2.StatusCode != http.StatusConflict {
		b, _ := io.ReadAll(lres2.Body)
		t.Fatalf("second launch: expected 409, got %d %s", lres2.StatusCode, b)
	}
}

func TestOnboardingHTTP_previewMissingAdmin(t *testing.T) {
	f, cleanup := presetOnboardTestApp(t)
	defer cleanup()
	admin := uuid.MustParse("30000000-0000-0000-0000-000000000001")

	cr := httptest.NewRequest(http.MethodPost, "/api/onboarding/sessions", nil)
	cr.Header.Set(httpcontext.HeaderPrincipalUserID, admin.String())
	cres, err := f.Test(cr)
	if err != nil {
		t.Fatal(err)
	}
	defer cres.Body.Close()
	cbody, _ := io.ReadAll(cres.Body)
	if cres.StatusCode != http.StatusCreated {
		t.Fatalf("create: %d %s", cres.StatusCode, cbody)
	}
	var sess struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(cbody, &sess)

	st, _ := json.Marshal(map[string]any{"template_code": "minimal"})
	sr := httptest.NewRequest(http.MethodPost, "/api/onboarding/sessions/"+sess.ID+"/select-template", bytes.NewReader(st))
	sr.Header.Set("Content-Type", "application/json")
	sr.Header.Set(httpcontext.HeaderPrincipalUserID, admin.String())
	sres, err := f.Test(sr)
	if err != nil {
		t.Fatal(err)
	}
	defer sres.Body.Close()
	if sres.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(sres.Body)
		t.Fatalf("select-template: %d %s", sres.StatusCode, b)
	}

	pv := httptest.NewRequest(http.MethodPost, "/api/onboarding/sessions/"+sess.ID+"/preview", nil)
	pv.Header.Set(httpcontext.HeaderPrincipalUserID, admin.String())
	pvres, err := f.Test(pv)
	if err != nil {
		t.Fatal(err)
	}
	defer pvres.Body.Close()
	pvb, _ := io.ReadAll(pvres.Body)
	if pvres.StatusCode != http.StatusOK {
		t.Fatalf("preview: %d %s", pvres.StatusCode, pvb)
	}
	var preview struct {
		ValidationIssues []string `json:"validation_issues"`
	}
	_ = json.Unmarshal(pvb, &preview)
	found := false
	for _, iss := range preview.ValidationIssues {
		if iss != "" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected validation issues when admin missing")
	}
}
