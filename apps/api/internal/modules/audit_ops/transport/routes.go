package transport

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/knowledgelayer/api/internal/modules/audit_ops/app"
)

// RegisterReadRoutes mounts GET /audit-events and GET /audit-events/:id. Caller supplies gate (principal + identity-admin).
func RegisterReadRoutes(r fiber.Router, svc *app.Service, gate func(c *fiber.Ctx) error) {
	r.Get("/audit-events", func(c *fiber.Ctx) error {
		if err := gate(c); err != nil {
			return err
		}
		limit := 100
		if v := c.Query("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				limit = n
			}
		}
		list, err := svc.List(c.Context(), c.Query("event_type"), c.Query("target_type"), limit)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(list)
	})

	r.Get("/audit-events/:id", func(c *fiber.Ctx) error {
		if err := gate(c); err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		e, err := svc.Get(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		return c.JSON(e)
	})
}
