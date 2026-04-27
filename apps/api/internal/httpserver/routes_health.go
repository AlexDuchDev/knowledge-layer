package httpserver

import (
	"context"
	"crypto/subtle"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/knowledgelayer/api/internal/app"
	"github.com/knowledgelayer/api/internal/config"
	"github.com/knowledgelayer/api/internal/httpcontext"
	"github.com/knowledgelayer/api/internal/platform/metrics"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// PrometheusHTTPRequestsMiddleware increments http_requests_total after each request (method + route path pattern).
func PrometheusHTTPRequestsMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		err := c.Next()
		route := "unspecified"
		if r := c.Route(); r != nil && r.Path != "" {
			route = r.Path
		}
		metrics.HTTPRequestsCounter().WithLabelValues(c.Method(), route).Inc()
		return err
	}
}

func mountHealthAndOps(f *fiber.App, d *app.Deps, cfg config.Config) {
	f.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	f.Get("/ops/health", func(c *fiber.Ctx) error {
		reqCtx := c.Context()
		if cfg.IsLocalDev() {
			return c.JSON(opsHealthDetailed(reqCtx, d))
		}
		if cfg.OpsAuthToken != "" {
			if !opsBearerAuthorized(c, cfg.OpsAuthToken) {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
			}
			return c.JSON(opsHealthDetailed(reqCtx, d))
		}
		return c.JSON(opsHealthRedacted(reqCtx, d))
	})

	f.Get("/ops/failed-runs", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if err := requireCanManageIdentity(c, d, principal); err != nil {
			return err
		}
		ingest, err := d.Ingestion.ListRecentFailedIngestionRuns(c.Context(), 25)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		jobs, err := d.Jobs.ListRecentFailedJobRuns(c.Context(), 25)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(fiber.Map{
			"ingestion_runs": ingest,
			"job_runs":       jobs,
		})
	})

	f.Get("/metrics", func(c *fiber.Ctx) error {
		if !cfg.IsLocalDev() {
			if cfg.OpsAuthToken == "" || !opsBearerAuthorized(c, cfg.OpsAuthToken) {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
			}
		}
		h := promhttp.HandlerFor(metrics.Gatherer(), promhttp.HandlerOpts{EnableOpenMetrics: true})
		return adaptor.HTTPHandler(h)(c)
	})
}

func opsHealthDetailed(ctx context.Context, d *app.Deps) fiber.Map {
	dbSt := "ok"
	if err := d.Pool.Ping(ctx); err != nil {
		dbSt = err.Error()
	}
	return fiber.Map{
		"status":     "ok",
		"database":   dbSt,
		"opensearch": d.Search.OpenSearchHealth(ctx),
	}
}

func opsHealthRedacted(ctx context.Context, d *app.Deps) fiber.Map {
	dbOK := d.Pool.Ping(ctx) == nil
	osStr := d.Search.OpenSearchHealth(ctx)
	osOK := osStr == "ok" || osStr == "disabled"
	status := "ok"
	if !dbOK || !osOK {
		status = "degraded"
	}
	return fiber.Map{
		"status":        status,
		"database_ok":   dbOK,
		"opensearch_ok": osOK,
	}
}

func opsBearerAuthorized(c *fiber.Ctx, token string) bool {
	if token == "" {
		return false
	}
	h := c.Get("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(h, prefix) {
		return false
	}
	got := strings.TrimSpace(strings.TrimPrefix(h, prefix))
	if len(got) != len(token) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(token)) == 1
}
