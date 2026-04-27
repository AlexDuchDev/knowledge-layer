package httpserver

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/knowledgelayer/api/internal/app"
	"github.com/knowledgelayer/api/internal/httpcontext"
	"github.com/knowledgelayer/api/internal/onboarding"
)

func mountOnboardingRoutes(api fiber.Router, d *app.Deps) {
	if d.Onboarding == nil {
		return
	}
	svc := d.Onboarding

	api.Get("/onboarding/templates", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if err := requireCanManageIdentity(c, d, principal); err != nil {
			return err
		}
		list, err := svc.ListTemplates(c.Context())
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(list)
	})

	api.Post("/onboarding/sessions", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if err := requireCanManageIdentity(c, d, principal); err != nil {
			return err
		}
		sess, err := svc.CreateSession(c.Context(), principal)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.Status(fiber.StatusCreated).JSON(sess)
	})

	api.Get("/onboarding/sessions", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if err := requireCanManageIdentity(c, d, principal); err != nil {
			return err
		}
		limit := 50
		if q := c.Query("limit"); q != "" {
			if n, err := strconv.Atoi(q); err == nil && n > 0 && n <= 100 {
				limit = n
			}
		}
		list, err := svc.ListSessions(c.Context(), principal, limit)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(list)
	})

	api.Get("/onboarding/sessions/:id", func(c *fiber.Ctx) error {
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
		sess, err := svc.GetSession(c.Context(), id, principal)
		if err != nil {
			if onboarding.IsNotFound(err) {
				return fiber.NewError(fiber.StatusNotFound, "session not found")
			}
			if onboarding.IsForbidden(err) {
				return fiber.NewError(fiber.StatusForbidden, "forbidden")
			}
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(sess)
	})

	api.Patch("/onboarding/sessions/:id", func(c *fiber.Ctx) error {
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
		var body onboarding.SessionPatch
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		sess, err := svc.PatchSession(c.Context(), id, principal, body)
		if err != nil {
			if onboarding.IsNotFound(err) {
				return fiber.NewError(fiber.StatusNotFound, "session not found")
			}
			if onboarding.IsForbidden(err) {
				return fiber.NewError(fiber.StatusForbidden, "forbidden")
			}
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		return c.JSON(sess)
	})

	api.Post("/onboarding/sessions/:id/select-template", func(c *fiber.Ctx) error {
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
			TemplateCode string `json:"template_code"`
		}
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		sess, err := svc.SelectTemplate(c.Context(), id, principal, body.TemplateCode)
		if err != nil {
			if onboarding.IsNotFound(err) {
				return fiber.NewError(fiber.StatusNotFound, "session or template not found")
			}
			if onboarding.IsForbidden(err) {
				return fiber.NewError(fiber.StatusForbidden, "forbidden")
			}
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		return c.JSON(sess)
	})

	api.Post("/onboarding/sessions/:id/preview", func(c *fiber.Ctx) error {
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
		prev, err := svc.PreviewLaunch(c.Context(), id, principal)
		if err != nil {
			if onboarding.IsNotFound(err) {
				return fiber.NewError(fiber.StatusNotFound, "session not found")
			}
			if onboarding.IsForbidden(err) {
				return fiber.NewError(fiber.StatusForbidden, "forbidden")
			}
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		return c.JSON(prev)
	})

	api.Post("/onboarding/sessions/:id/launch", func(c *fiber.Ctx) error {
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
		res, err := svc.Launch(c.Context(), id, principal)
		if err != nil {
			if onboarding.IsNotFound(err) {
				return fiber.NewError(fiber.StatusNotFound, "session not found")
			}
			if onboarding.IsForbidden(err) {
				return fiber.NewError(fiber.StatusForbidden, "forbidden")
			}
			if onboarding.IsAlreadyLaunched(err) {
				return fiber.NewError(fiber.StatusConflict, err.Error())
			}
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("onboarding.launched", "user", &principal, "onboarding_session", &id, nil))
		return c.JSON(res)
	})
}
