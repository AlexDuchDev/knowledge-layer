package httpserver

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/knowledgelayer/api/internal/app"
	"github.com/knowledgelayer/api/internal/httpcontext"
	"github.com/knowledgelayer/api/internal/role_builder"
)

func mountRoleBuilderRoutes(f *fiber.App, d *app.Deps) {
	rb := d.RoleBuilder
	if rb == nil {
		return
	}

	parseLimitOffsetRB := func(c *fiber.Ctx) (int, int) {
		limit := 50
		if v := c.Query("limit"); v != "" {
			if n, e := strconv.Atoi(v); e == nil && n > 0 && n <= 200 {
				limit = n
			}
		}
		offset := 0
		if v := c.Query("offset"); v != "" {
			if n, e := strconv.Atoi(v); e == nil && n >= 0 {
				offset = n
			}
		}
		return limit, offset
	}

	f.Get("/roles/presets", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if err := requireCanManageIdentity(c, d, principal); err != nil {
			return err
		}
		list, err := rb.Presets.List(c.Context())
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(list)
	})

	f.Get("/roles", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if err := requireCanManageIdentity(c, d, principal); err != nil {
			return err
		}
		list, err := rb.Definitions.List(c.Context())
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(list)
	})

	f.Post("/roles", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if err := requireCanManageIdentity(c, d, principal); err != nil {
			return err
		}
		var body roleWriteBody
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		in := body.toRoleWriteInput()
		id, err := rb.Definitions.Create(c.Context(), in)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("role.created", "user", &principal, "role", &id, nil))
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
	})

	f.Post("/roles/from-preset", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if err := requireCanManageIdentity(c, d, principal); err != nil {
			return err
		}
		var body struct {
			PresetKey   string  `json:"preset_key"`
			Code        string  `json:"code"`
			Name        string  `json:"name"`
			Description *string `json:"description"`
		}
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		id, err := rb.Presets.CreateFromPreset(c.Context(), body.PresetKey, body.Code, body.Name, body.Description)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("role.created_from_preset", "user", &principal, "role", &id, nil))
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
	})

	f.Get("/roles/:id/preview", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if err := requireCanManageIdentity(c, d, principal); err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		pv, err := rb.Preview.PreviewRole(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, err.Error())
		}
		return c.JSON(pv)
	})

	f.Get("/roles/:id/assignments", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if err := requireCanManageIdentity(c, d, principal); err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		limit, offset := parseLimitOffsetRB(c)
		list, err := rb.Assignments.ListByRole(c.Context(), id, limit, offset)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(list)
	})

	f.Post("/roles/:id/assignments", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if err := requireCanManageIdentity(c, d, principal); err != nil {
			return err
		}
		roleID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		var body struct {
			UserID    uuid.UUID  `json:"user_id"`
			ScopeType string     `json:"scope_type"`
			ScopeID   *uuid.UUID `json:"scope_id"`
			ExpiresAt *time.Time `json:"expires_at"`
		}
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		if body.ScopeType == "domain" && body.ScopeID != nil {
			if err := requirePublishOnDomain(c, d, principal, *body.ScopeID); err != nil {
				return err
			}
		}
		a, err := rb.Assignments.Assign(c.Context(), principal, body.UserID, roleID, body.ScopeType, body.ScopeID, body.ExpiresAt)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("role.assignment.created", "user", &principal, "user_role_binding", &a.ID, nil))
		return c.Status(fiber.StatusCreated).JSON(a)
	})

	f.Post("/roles/:id/clone", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if err := requireCanManageIdentity(c, d, principal); err != nil {
			return err
		}
		srcID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		var body struct {
			Code        string  `json:"code"`
			Name        string  `json:"name"`
			Description *string `json:"description"`
			Category    string  `json:"category"`
		}
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		id, err := rb.Definitions.Clone(c.Context(), srcID, body.Code, body.Name, body.Description, body.Category, nil)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("role.cloned", "user", &principal, "role", &id, nil))
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
	})

	f.Get("/roles/:id", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if err := requireCanManageIdentity(c, d, principal); err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		full, err := rb.Definitions.Get(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, err.Error())
		}
		return c.JSON(full)
	})

	f.Patch("/roles/:id", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if err := requireCanManageIdentity(c, d, principal); err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		var body patchRoleBody
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		var bind *role_builder.RoleWriteInput
		if body.Bindings != nil {
			b := body.Bindings.toRoleWriteInput()
			bind = &b
		}
		if err := rb.Definitions.Patch(c.Context(), id, body.Name, body.Code, body.Description, body.Category, body.Active, body.ScopeModel, bind); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("role.updated", "user", &principal, "role", &id, nil))
		full, err := rb.Definitions.Get(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(full)
	})

	f.Delete("/roles/:id", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if err := requireCanManageIdentity(c, d, principal); err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		if err := rb.Definitions.Delete(c.Context(), id); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("role.deleted", "user", &principal, "role", &id, nil))
		return c.SendStatus(fiber.StatusNoContent)
	})
}

// JSON DTOs for role builder HTTP (snake_case).

type roleWriteBody struct {
	Code           string            `json:"code"`
	Name           string            `json:"name"`
	Description    *string           `json:"description"`
	Category       string            `json:"category"`
	Active         *bool             `json:"active"`
	ScopeModel     string            `json:"scope_model"`
	ActionCodes    []string          `json:"action_codes"`
	DomainIDs      []uuid.UUID       `json:"allowed_domains"`
	EntityTypes    []string          `json:"allowed_entity_types"`
	SourceScopes   []sourceScopeBody `json:"allowed_source_scopes"`
	ScenarioKeys   []string          `json:"allowed_scenarios"`
	DashboardKeys  []string          `json:"allowed_dashboards"`
	JobPermissions []jobPermBody     `json:"job_permissions"`
	Governance     *governanceBody   `json:"governance"`
}

func (b roleWriteBody) toRoleWriteInput() role_builder.RoleWriteInput {
	in := role_builder.RoleWriteInput{
		Code:          b.Code,
		Name:          b.Name,
		Description:   b.Description,
		Category:      b.Category,
		Active:        b.Active,
		ScopeModel:    b.ScopeModel,
		ActionCodes:   b.ActionCodes,
		DomainIDs:     b.DomainIDs,
		EntityTypes:   b.EntityTypes,
		ScenarioKeys:  b.ScenarioKeys,
		DashboardKeys: b.DashboardKeys,
	}
	for _, s := range b.SourceScopes {
		in.SourceScopes = append(in.SourceScopes, role_builder.SourceScopeRef{ScopeKind: s.ScopeKind, ScopeRef: s.ScopeRef})
	}
	for _, j := range b.JobPermissions {
		in.JobPermissions = append(in.JobPermissions, role_builder.JobPermissionWrite{
			JobID: j.JobID, CanRun: j.CanRun, CanConfigure: j.CanConfigure, CanReviewJobOutput: j.CanReviewJobOutput,
		})
	}
	if b.Governance != nil {
		in.Governance = &role_builder.GovernanceRow{
			CanReviewOutputs:     b.Governance.CanReviewOutputs,
			CanApproveOutputs:    b.Governance.CanApproveOutputs,
			CanPublishOutputs:    b.Governance.CanPublishOutputs,
			CanOverridePolicies:  b.Governance.CanOverridePolicies,
			CanManageAssignments: b.Governance.CanManageAssignments,
		}
	}
	return in
}

type sourceScopeBody struct {
	ScopeKind string `json:"scope_kind"`
	ScopeRef  string `json:"scope_ref"`
}

type jobPermBody struct {
	JobID              uuid.UUID `json:"knowledge_job_id"`
	CanRun             bool      `json:"can_run"`
	CanConfigure       bool      `json:"can_configure"`
	CanReviewJobOutput bool      `json:"can_review_job_output"`
}

type governanceBody struct {
	CanReviewOutputs     bool `json:"can_review_outputs"`
	CanApproveOutputs    bool `json:"can_approve_outputs"`
	CanPublishOutputs    bool `json:"can_publish_outputs"`
	CanOverridePolicies  bool `json:"can_override_policies"`
	CanManageAssignments bool `json:"can_manage_assignments"`
}

type patchRoleBody struct {
	Name        *string        `json:"name"`
	Code        *string        `json:"code"`
	Description *string        `json:"description"`
	Category    *string        `json:"category"`
	Active      *bool          `json:"active"`
	ScopeModel  *string        `json:"scope_model"`
	Bindings    *roleWriteBody `json:"bindings"`
}
