package mcp

import (
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/google/uuid"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// BearerVerifier is the subset of oauth_proxy.Server the MCP middleware uses.
// Interface form so v0.5.1 doesn't carry a hard import-cycle risk between
// mcp and oauth_proxy.
type BearerVerifier interface {
	VerifyBearer(rawToken string) (uuid.UUID, error)
}

// Mount wires the MCP server onto a Fiber router. Two HTTP surfaces:
//
//	POST /mcp        — streamable HTTP transport (mcp-go's standard)
//
// Bearer auth runs as Fiber middleware before the underlying http.Handler.
// Tokens are minted by /oauth/token (v0.5.0); the middleware extracts the
// principal UUID and passes it via context to the access-guard.
func Mount(r fiber.Router, srv *mcpserver.MCPServer, bv BearerVerifier) {
	streamable := mcpserver.NewStreamableHTTPServer(srv)

	mw := bearerMiddleware(bv)

	// We expose POST + GET on the same path; mcp-go's StreamableHTTPServer
	// dispatches both (POST for tool calls, GET for SSE notifications when
	// supported). Adaptor bridges http.Handler into Fiber.
	r.Use("/mcp", mw)
	r.All("/mcp", adaptor.HTTPHandler(http.HandlerFunc(streamable.ServeHTTP)))
}

// bearerMiddleware extracts and verifies the OAuth bearer, sets the
// principal UUID in the request context so the access-guard can read it.
// Reject paths return 401 with an OAuth-shaped WWW-Authenticate header so
// MCP clients know to re-authorize.
func bearerMiddleware(bv BearerVerifier) fiber.Handler {
	return func(c *fiber.Ctx) error {
		auth := c.Get(fiber.HeaderAuthorization)
		if !strings.HasPrefix(auth, "Bearer ") {
			c.Set(fiber.HeaderWWWAuthenticate, `Bearer realm="mcp", error="invalid_token"`)
			return fiber.NewError(fiber.StatusUnauthorized, "Authorization: Bearer ... required")
		}
		tok := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
		principal, err := bv.VerifyBearer(tok)
		if err != nil {
			c.Set(fiber.HeaderWWWAuthenticate, `Bearer realm="mcp", error="invalid_token"`)
			return fiber.NewError(fiber.StatusUnauthorized, "invalid bearer: "+err.Error())
		}
		// Stash on Fiber Locals AND on the standard context the underlying
		// http.Handler sees. The MCP server reads from context, so the
		// adaptor-converted request must carry it.
		ctx := WithPrincipal(c.UserContext(), principal)
		c.SetUserContext(ctx)
		return c.Next()
	}
}
