package httpserver

import (
	"github.com/gofiber/fiber/v2"
	"github.com/knowledgelayer/api/internal/app"
	"github.com/knowledgelayer/api/internal/config"
	klmcp "github.com/knowledgelayer/api/internal/mcp"
)

// Mount wires HTTP routes. Transport stays thin: handlers delegate to app.Deps services.
// Large route tables live in routes_register.go (mountAPIRoutes).
func Mount(f *fiber.App, d *app.Deps, cfgOpt ...config.Config) {
	cfg := config.Config{AppEnv: "local", AuthMode: "development_header"}
	if len(cfgOpt) > 0 {
		cfg = cfgOpt[0]
	}
	f.Use(principalMiddleware(cfg))
	f.Use(PrometheusHTTPRequestsMiddleware())

	// OAuth proxy (v0.5.0): mount before route groups so .well-known and
	// /oauth/* are reachable without a session principal. The principal
	// middleware above is non-rejecting, so RFC 8414's anonymous metadata
	// endpoint stays public as required.
	if d.OAuthProxy != nil {
		d.OAuthProxy.Mount(f)
	}

	// MCP endpoint (v0.5.1) — bearer-protected via OAuth proxy. Hardening
	// guarantees both are present together when MCP_ENABLED=true; this
	// nil-guard handles the local-dev path where MCP is on but the IDP
	// failed to discover and oauthSrv stayed nil.
	if d.MCPServer != nil && d.OAuthProxy != nil {
		klmcp.Mount(f, d.MCPServer, d.OAuthProxy)
	}

	mountHealthAndOps(f, d, cfg)
	mountSecondBrainWebhooks(f, d, cfg)
	mountToolGatewayRoutes(f, d)
	api := f.Group("/api")
	mountPresetCatalogRoutes(api, d)
	mountOnboardingRoutes(api, d)
	mountRoleBuilderRoutes(f, d)
	mountScenarioBuilderRoutes(f, d)
	mountIdentityAdmin(f, d)
	mountAuthSettingsRoutes(f, d, cfg)
	mountAPIRoutes(f, d, cfg)
}
