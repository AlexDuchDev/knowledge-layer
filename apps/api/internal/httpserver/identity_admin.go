package httpserver

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/knowledgelayer/api/internal/app"
	"github.com/knowledgelayer/api/internal/httpcontext"
	"github.com/knowledgelayer/api/internal/identity_access"
)

// requireCanManageIdentity allows users who may publish in at least one granted domain (global admin pattern).
func requireCanManageIdentity(c *fiber.Ctx, d *app.Deps, principal uuid.UUID) error {
	doms, err := d.Access.DomainIDsWithGrant(c.Context(), principal)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	for _, dom := range doms {
		did := dom
		dec, err := d.Access.Evaluate(c.Context(), identity_access.EvaluateInput{
			PrincipalID:  principal,
			Action:       "publish",
			ResourceType: "domain",
			DomainID:     &did,
		})
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		if dec.Allow && dec.SensitivityOK {
			return nil
		}
	}
	return fiber.NewError(fiber.StatusForbidden, "access denied")
}

func requirePublishOnDomain(c *fiber.Ctx, d *app.Deps, principal uuid.UUID, domainID uuid.UUID) error {
	dec, err := d.Access.Evaluate(c.Context(), identity_access.EvaluateInput{
		PrincipalID:  principal,
		Action:       "publish",
		ResourceType: "domain",
		DomainID:     &domainID,
	})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if !dec.Allow || !dec.SensitivityOK {
		return fiber.NewError(fiber.StatusForbidden, "access denied")
	}
	return nil
}

func mountIdentityAdmin(f *fiber.App, d *app.Deps) {
	parseLimitOffset := func(c *fiber.Ctx) (int, int) {
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

	f.Post("/users", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if err := requireCanManageIdentity(c, d, principal); err != nil {
			return err
		}
		var body struct {
			Email string `json:"email"`
			Name  string `json:"name"`
		}
		if err := c.BodyParser(&body); err != nil || body.Email == "" || body.Name == "" {
			return fiber.NewError(fiber.StatusBadRequest, "email and name required")
		}
		u, err := d.Identity.CreateUser(c.Context(), body.Email, body.Name)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("user.created", "user", &principal, "user", &u.ID, nil))
		return c.Status(fiber.StatusCreated).JSON(u)
	})

	f.Patch("/users/:id", func(c *fiber.Ctx) error {
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
		var body struct {
			Email         *string    `json:"email"`
			Name          *string    `json:"name"`
			Status        *string    `json:"status"`
			PrimaryTeamID *uuid.UUID `json:"primary_team_id"`
		}
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		u, err := d.Identity.PatchUser(c.Context(), id, body.Email, body.Name, body.Status, body.PrimaryTeamID)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("user.updated", "user", &principal, "user", &id, nil))
		return c.JSON(u)
	})

	f.Get("/teams", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if err := requireCanManageIdentity(c, d, principal); err != nil {
			return err
		}
		limit, offset := parseLimitOffset(c)
		list, err := d.Identity.ListTeams(c.Context(), limit, offset)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(list)
	})

	f.Post("/teams", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if err := requireCanManageIdentity(c, d, principal); err != nil {
			return err
		}
		var body struct {
			Name        string     `json:"name"`
			Description *string    `json:"description"`
			OwnerID     *uuid.UUID `json:"owner_id"`
		}
		if err := c.BodyParser(&body); err != nil || body.Name == "" {
			return fiber.NewError(fiber.StatusBadRequest, "name required")
		}
		t, err := d.Identity.CreateTeam(c.Context(), body.Name, body.Description, body.OwnerID)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("team.created", "user", &principal, "team", &t.ID, nil))
		return c.Status(fiber.StatusCreated).JSON(t)
	})

	f.Patch("/teams/:id", func(c *fiber.Ctx) error {
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
		var body struct {
			Name        *string `json:"name"`
			Description *string `json:"description"`
			Status      *string `json:"status"`
		}
		_ = c.BodyParser(&body)
		t, err := d.Identity.PatchTeam(c.Context(), id, body.Name, body.Description, body.Status)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("team.updated", "user", &principal, "team", &id, nil))
		return c.JSON(t)
	})

	f.Get("/user-team-memberships", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if err := requireCanManageIdentity(c, d, principal); err != nil {
			return err
		}
		limit, offset := parseLimitOffset(c)
		var teamID *uuid.UUID
		var userID *uuid.UUID
		if v := c.Query("team_id"); v != "" {
			id, err := uuid.Parse(v)
			if err == nil {
				teamID = &id
			}
		}
		if v := c.Query("user_id"); v != "" {
			id, err := uuid.Parse(v)
			if err == nil {
				userID = &id
			}
		}
		list, err := d.Identity.ListTeamMemberships(c.Context(), teamID, userID, limit, offset)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(list)
	})

	f.Post("/user-team-memberships", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if err := requireCanManageIdentity(c, d, principal); err != nil {
			return err
		}
		var body struct {
			UserID         uuid.UUID `json:"user_id"`
			TeamID         uuid.UUID `json:"team_id"`
			MembershipType string    `json:"membership_type"`
		}
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		m, err := d.Identity.AddTeamMembership(c.Context(), body.UserID, body.TeamID, body.MembershipType)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("team.membership_added", "user", &principal, "team", &body.TeamID, nil))
		return c.Status(fiber.StatusCreated).JSON(m)
	})

	f.Delete("/user-team-memberships/:id", func(c *fiber.Ctx) error {
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
		if err := d.Identity.DeleteTeamMembership(c.Context(), id); err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("team.membership_removed", "user", &principal, "user_team_membership", &id, nil))
		return c.SendStatus(fiber.StatusNoContent)
	})

	f.Get("/domain-grants", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if err := requireCanManageIdentity(c, d, principal); err != nil {
			return err
		}
		limit, offset := parseLimitOffset(c)
		var userID *uuid.UUID
		var domainID *uuid.UUID
		if v := c.Query("user_id"); v != "" {
			id, err := uuid.Parse(v)
			if err == nil {
				userID = &id
			}
		}
		if v := c.Query("domain_id"); v != "" {
			id, err := uuid.Parse(v)
			if err == nil {
				domainID = &id
			}
		}
		list, err := d.Identity.ListDomainGrants(c.Context(), userID, domainID, limit, offset)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(list)
	})

	f.Post("/domain-grants", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		var body struct {
			UserID         uuid.UUID `json:"user_id"`
			DomainID       uuid.UUID `json:"domain_id"`
			AccessLevel    string    `json:"access_level"`
			SensitivityCap int       `json:"sensitivity_cap"`
		}
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		if body.AccessLevel == "" {
			body.AccessLevel = "read"
		}
		if err := requirePublishOnDomain(c, d, principal, body.DomainID); err != nil {
			return err
		}
		g, err := d.Identity.UpsertDomainGrant(c.Context(), body.UserID, body.DomainID, body.AccessLevel, body.SensitivityCap, &principal)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("domain_grant.upserted", "user", &principal, "domain_grant", &g.ID, nil))
		return c.Status(fiber.StatusCreated).JSON(g)
	})

	f.Patch("/domain-grants/:id", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		g0, err := d.Identity.GetDomainGrant(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		if err := requirePublishOnDomain(c, d, principal, g0.DomainID); err != nil {
			return err
		}
		var body struct {
			AccessLevel    *string    `json:"access_level"`
			SensitivityCap *int       `json:"sensitivity_cap"`
			ExpiresAt      *time.Time `json:"expires_at"`
		}
		_ = c.BodyParser(&body)
		g, err := d.Identity.PatchDomainGrant(c.Context(), id, body.AccessLevel, body.SensitivityCap, body.ExpiresAt)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("domain_grant.updated", "user", &principal, "domain_grant", &id, nil))
		return c.JSON(g)
	})

	f.Delete("/domain-grants/:id", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		g0, err := d.Identity.GetDomainGrant(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		if err := requirePublishOnDomain(c, d, principal, g0.DomainID); err != nil {
			return err
		}
		if err := d.Identity.DeleteDomainGrant(c.Context(), id); err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("domain_grant.deleted", "user", &principal, "domain_grant", &id, nil))
		return c.SendStatus(fiber.StatusNoContent)
	})

	f.Get("/user-role-bindings", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if err := requireCanManageIdentity(c, d, principal); err != nil {
			return err
		}
		limit, offset := parseLimitOffset(c)
		var userID *uuid.UUID
		if v := c.Query("user_id"); v != "" {
			id, err := uuid.Parse(v)
			if err == nil {
				userID = &id
			}
		}
		list, err := d.Identity.ListUserRoleBindings(c.Context(), userID, limit, offset)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(list)
	})

	f.Post("/user-role-bindings", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if err := requireCanManageIdentity(c, d, principal); err != nil {
			return err
		}
		var body struct {
			UserID    uuid.UUID  `json:"user_id"`
			RoleID    uuid.UUID  `json:"role_id"`
			ScopeType string     `json:"scope_type"`
			ScopeID   *uuid.UUID `json:"scope_id"`
		}
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		if body.ScopeType == "" {
			body.ScopeType = "global"
		}
		if body.ScopeType == "domain" && body.ScopeID != nil {
			if err := requirePublishOnDomain(c, d, principal, *body.ScopeID); err != nil {
				return err
			}
		}
		b, err := d.Identity.CreateUserRoleBinding(c.Context(), body.UserID, body.RoleID, body.ScopeType, body.ScopeID, &principal)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("role_binding.created", "user", &principal, "user_role_binding", &b.ID, nil))
		return c.Status(fiber.StatusCreated).JSON(b)
	})

	f.Patch("/user-role-bindings/:id", func(c *fiber.Ctx) error {
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
		prev, err := d.Identity.GetUserRoleBinding(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		var body struct {
			ScopeType *string    `json:"scope_type"`
			ScopeID   *uuid.UUID `json:"scope_id"`
			ExpiresAt *time.Time `json:"expires_at"`
		}
		_ = c.BodyParser(&body)
		st := prev.ScopeType
		if body.ScopeType != nil {
			st = *body.ScopeType
		}
		sid := prev.ScopeID
		if body.ScopeID != nil {
			sid = body.ScopeID
		}
		if st == "domain" && sid != nil {
			if err := requirePublishOnDomain(c, d, principal, *sid); err != nil {
				return err
			}
		}
		b, err := d.Identity.PatchUserRoleBinding(c.Context(), id, body.ScopeType, body.ScopeID, body.ExpiresAt)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("role_binding.updated", "user", &principal, "user_role_binding", &id, nil))
		return c.JSON(b)
	})

	f.Delete("/user-role-bindings/:id", func(c *fiber.Ctx) error {
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
		prev, err := d.Identity.GetUserRoleBinding(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		if prev.ScopeType == "domain" && prev.ScopeID != nil {
			if err := requirePublishOnDomain(c, d, principal, *prev.ScopeID); err != nil {
				return err
			}
		}
		if err := d.Identity.DeleteUserRoleBinding(c.Context(), id); err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("role_binding.deleted", "user", &principal, "user_role_binding", &id, nil))
		return c.SendStatus(fiber.StatusNoContent)
	})

	f.Post("/domains", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if err := requireCanManageIdentity(c, d, principal); err != nil {
			return err
		}
		var body struct {
			Name                    string     `json:"name"`
			Description             *string    `json:"description"`
			OwnerID                 *uuid.UUID `json:"owner_id"`
			DefaultSensitivityLevel int        `json:"default_sensitivity_level"`
		}
		if err := c.BodyParser(&body); err != nil || body.Name == "" {
			return fiber.NewError(fiber.StatusBadRequest, "name required")
		}
		dom, _, err := d.Identity.CreateDomainWithPolicy(c.Context(), body.Name, body.Description, body.OwnerID, body.DefaultSensitivityLevel)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("domain.created", "user", &principal, "domain", &dom.ID, nil))
		return c.Status(fiber.StatusCreated).JSON(dom)
	})

	f.Patch("/domains/:id", func(c *fiber.Ctx) error {
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
		if err := requirePublishOnDomain(c, d, principal, id); err != nil {
			return err
		}
		var body struct {
			Name        *string    `json:"name"`
			Description *string    `json:"description"`
			Status      *string    `json:"status"`
			OwnerID     *uuid.UUID `json:"owner_id"`
		}
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		dom, err := d.Identity.PatchDomain(c.Context(), id, body.Name, body.Description, body.Status, body.OwnerID)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("domain.updated", "user", &principal, "domain", &id, nil))
		return c.JSON(dom)
	})
}
