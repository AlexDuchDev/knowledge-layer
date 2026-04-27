package httpserver

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/knowledgelayer/api/internal/auth"
	"github.com/knowledgelayer/api/internal/config"
	"github.com/knowledgelayer/api/internal/httpcontext"
)

func principalMiddleware(cfg config.Config) fiber.Handler {
	sec := cfg.SessionSecretBytes()
	return func(c *fiber.Ctx) error {
		if cfg.AuthMode == "session" && len(sec) > 0 {
			if uid, ok := auth.SessionUserID(c, sec); ok {
				c.Locals(httpcontext.LocalsPrincipalID, uid)
				return c.Next()
			}
		}
		headerOK := cfg.AuthMode == "development_header" && cfg.IsLocalDev()
		if headerOK {
			c.Locals(httpcontext.LocalsHeaderPrincipalOK, true)
			raw := c.Get(httpcontext.HeaderPrincipalUserID)
			if raw != "" {
				if id, err := uuid.Parse(raw); err == nil {
					if c.Locals(httpcontext.LocalsPrincipalID) == nil {
						c.Locals(httpcontext.LocalsPrincipalID, id)
					}
				}
			}
		}
		return c.Next()
	}
}
