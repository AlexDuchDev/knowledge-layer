package httpcontext

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

const HeaderPrincipalUserID = "X-Principal-User-ID"

// LocalsPrincipalID is the Fiber Locals key for the authenticated user (session or middleware).
const LocalsPrincipalID = "principal_user_id"

// LocalsHeaderPrincipalOK allows parsing X-Principal-User-ID (development only; set by middleware).
const LocalsHeaderPrincipalOK = "header_principal_allowed"

func PrincipalID(c *fiber.Ctx) (uuid.UUID, bool) {
	if v := c.Locals(LocalsPrincipalID); v != nil {
		if id, ok := v.(uuid.UUID); ok {
			return id, true
		}
	}
	if c.Locals(LocalsHeaderPrincipalOK) != true {
		return uuid.Nil, false
	}
	raw := c.Get(HeaderPrincipalUserID)
	if raw == "" {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}

func RequirePrincipal(c *fiber.Ctx) (uuid.UUID, error) {
	id, ok := PrincipalID(c)
	if !ok {
		return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "missing "+HeaderPrincipalUserID)
	}
	return id, nil
}
