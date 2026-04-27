package httpserver

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/knowledgelayer/api/internal/app"
	"github.com/knowledgelayer/api/internal/httpcontext"
	"github.com/knowledgelayer/api/internal/presetcatalog"
)

func mountPresetCatalogRoutes(api fiber.Router, d *app.Deps) {
	if d.PresetCatalog == nil {
		return
	}
	pc := d.PresetCatalog

	api.Get("/presets", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if err := requireCanManageIdentity(c, d, principal); err != nil {
			return err
		}
		list, err := pc.Catalog.List(c.Context(), c.Query("type"), c.Query("category_axis"), c.Query("category_code"))
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(list)
	})

	api.Get("/presets/:id", func(c *fiber.Ctx) error {
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
		detail, err := pc.Catalog.GetDetail(c.Context(), id)
		if err != nil {
			if presetcatalog.IsNotFound(err) {
				return fiber.NewError(fiber.StatusNotFound, "preset not found")
			}
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(detail)
	})

	api.Get("/presets/:id/related", func(c *fiber.Ctx) error {
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
		rel, err := pc.Relationships.ListRelated(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(rel)
	})

	api.Post("/presets/:id/instantiate", func(c *fiber.Ctx) error {
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
		var body presetcatalog.InstantiateRequest
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		res, err := pc.Instantiation.Instantiate(c.Context(), id, principal, body)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("preset.instantiated", "user", &principal, res.TargetKind, &res.TargetID, nil))
		return c.Status(fiber.StatusCreated).JSON(res)
	})
}
