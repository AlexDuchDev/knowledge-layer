package integration

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/knowledgelayer/api/internal/app"
	"github.com/knowledgelayer/api/internal/config"
	"github.com/knowledgelayer/api/internal/db"
	"github.com/knowledgelayer/api/internal/identity_access"
	"github.com/knowledgelayer/api/internal/role_builder"
)

func TestRoleBuilder_DomainAllowlistAffectsEvaluate(t *testing.T) {
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

	domA := uuid.New()
	domB := uuid.New()
	userU := uuid.New()

	_, err = pool.Exec(ctx, `
		INSERT INTO domains (id, name, description, default_access_policy_id, default_sensitivity_level, status)
		VALUES ($1,'rb_dom_a','',NULL,0,'active')`, domA)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO domains (id, name, description, default_access_policy_id, default_sensitivity_level, status)
		VALUES ($1,'rb_dom_b','',NULL,0,'active')`, domB)
	if err != nil {
		t.Fatal(err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO users (id, email, name, status) VALUES ($1,$2,'RB User','active')`, userU, fmt.Sprintf("rb_%s@test.local", userU.String()))
	if err != nil {
		t.Fatal(err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO domain_grants (user_id, domain_id, access_level, sensitivity_cap)
		VALUES ($1,$2,'admin',3),($1,$3,'admin',3)`, userU, domA, domB)
	if err != nil {
		t.Fatal(err)
	}

	roleID := uuid.New()
	_, err = pool.Exec(ctx, `
		INSERT INTO roles (id, code, name, description, category, active, scope_model, is_preset, is_system)
		VALUES ($1,'rb_narrow','RB Narrow','','domain',true,'global',false,false)`, roleID)
	if err != nil {
		t.Fatal(err)
	}

	viewID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	_, err = pool.Exec(ctx, `
		INSERT INTO role_action_permissions (role_id, action_permission_id) VALUES ($1,$2)`, roleID, viewID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `INSERT INTO role_domain_bindings (role_id, domain_id) VALUES ($1,$2)`, roleID, domA)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO role_governance_permissions (role_id) VALUES ($1) ON CONFLICT (role_id) DO NOTHING`, roleID)
	if err != nil {
		t.Fatal(err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO user_role_bindings (user_id, role_id, scope_type)
		VALUES ($1,$2,'global')`, userU, roleID)
	if err != nil {
		t.Fatal(err)
	}

	access := identity_access.NewAccessEvaluator(pool)

	decA, err := access.Evaluate(ctx, identity_access.EvaluateInput{
		PrincipalID:  userU,
		Action:       "view",
		ResourceType: "domain",
		DomainID:     &domA,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !decA.Allow {
		t.Fatalf("expected allow on domA, got %+v", decA)
	}

	decB, err := access.Evaluate(ctx, identity_access.EvaluateInput{
		PrincipalID:  userU,
		Action:       "view",
		ResourceType: "domain",
		DomainID:     &domB,
	})
	if err != nil {
		t.Fatal(err)
	}
	if decB.Allow {
		t.Fatalf("expected deny on domB (role domain allowlist), got %+v", decB)
	}
}

func TestRoleBuilder_AssignmentPrivilegedRequiresManagePermissions(t *testing.T) {
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
	viewer := uuid.MustParse("30000000-0000-0000-0000-000000000002")
	adminRole := uuid.MustParse("10000000-0000-0000-0000-000000000001")

	_, err = deps.RoleBuilder.Assignments.Assign(ctx, viewer, viewer, adminRole, "global", nil, nil)
	if err == nil {
		t.Fatal("viewer assigning admin (privileged) role should fail")
	}
}

func TestRoleBuilder_CloneIsIndependent(t *testing.T) {
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
	src := uuid.MustParse("c1000001-0000-4000-8000-000000000005") // executive viewer preset
	srcFull, err := deps.RoleBuilder.Definitions.Get(ctx, src)
	if err != nil {
		t.Fatal(err)
	}

	newID, err := deps.RoleBuilder.Definitions.Clone(ctx, src, fmt.Sprintf("exec_clone_%s", uuid.NewString()[:8]), "Exec Clone Test", nil, "operational", nil)
	if err != nil {
		t.Fatal(err)
	}
	if newID == uuid.Nil || newID == src {
		t.Fatal("bad clone id")
	}

	if len(srcFull.ActionCodes) < 2 {
		t.Fatalf("need at least 2 actions on source, got %v", srcFull.ActionCodes)
	}
	newCodes := append([]string(nil), srcFull.ActionCodes[:len(srcFull.ActionCodes)-1]...)
	jobsWrite := make([]role_builder.JobPermissionWrite, 0, len(srcFull.JobPermissions))
	for _, j := range srcFull.JobPermissions {
		jobsWrite = append(jobsWrite, role_builder.JobPermissionWrite{
			JobID: j.JobID, CanRun: j.CanRun, CanConfigure: j.CanConfigure, CanReviewJobOutput: j.CanReviewJobOutput,
		})
	}
	gov := srcFull.Governance
	bind := role_builder.RoleWriteInput{
		ActionCodes:    newCodes,
		DomainIDs:      append([]uuid.UUID(nil), srcFull.DomainIDs...),
		EntityTypes:    append([]string(nil), srcFull.EntityTypes...),
		SourceScopes:   append([]role_builder.SourceScopeRef(nil), srcFull.SourceScopes...),
		ScenarioKeys:   append([]string(nil), srcFull.ScenarioKeys...),
		DashboardKeys:  append([]string(nil), srcFull.DashboardKeys...),
		JobPermissions: jobsWrite,
		Governance:     &gov,
	}
	if err := deps.RoleBuilder.Definitions.Patch(ctx, newID, nil, nil, nil, nil, nil, nil, &bind); err != nil {
		t.Fatal(err)
	}

	cloneFull, err := deps.RoleBuilder.Definitions.Get(ctx, newID)
	if err != nil {
		t.Fatal(err)
	}
	if len(cloneFull.ActionCodes) != len(newCodes) {
		t.Fatalf("clone actions %v want %v", cloneFull.ActionCodes, newCodes)
	}

	srcAfter, err := deps.RoleBuilder.Definitions.Get(ctx, src)
	if err != nil {
		t.Fatal(err)
	}
	if len(srcAfter.ActionCodes) != len(srcFull.ActionCodes) {
		t.Fatalf("source preset mutated: was %d now %d", len(srcFull.ActionCodes), len(srcAfter.ActionCodes))
	}
}
