package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/knowledgelayer/api/internal/app"
	"github.com/knowledgelayer/api/internal/config"
	"github.com/knowledgelayer/api/internal/db"
	"github.com/knowledgelayer/api/internal/httpserver"
	"github.com/knowledgelayer/api/internal/instancebootstrap"
)

func main() {
	cfg := config.Load()
	if err := config.ValidateAPI(cfg); err != nil {
		log.Fatalf("config: %v", err)
	}
	ctx := context.Background()

	if err := db.MigrateUp(cfg.DatabaseURL); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	deps, err := app.NewDeps(pool, cfg)
	if err != nil {
		log.Fatalf("deps: %v", err)
	}
	if deps.GraphRAG != nil {
		if err := deps.GraphRAG.EnsureSchema(ctx); err != nil {
			log.Fatalf("neo4j schema: %v", err)
		}
		defer func() {
			_ = deps.GraphRAG.Close(ctx)
		}()
	}
	if err := instancebootstrap.Ensure(ctx, deps, cfg); err != nil {
		log.Fatalf("instance bootstrap: %v", err)
	}
	if err := deps.Search.EnsureOpenSearchIndex(ctx); err != nil {
		log.Printf("opensearch index: %v", err)
	}

	// BodyLimit 64 MiB: accommodates manual file uploads (50 MiB cap enforced
	// in the upload handler) plus headers/multipart overhead. Default is 4 MiB,
	// which would reject any non-trivial PDF/DOCX before our handler runs.
	f := fiber.New(fiber.Config{
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		BodyLimit:    64 * 1024 * 1024,
	})
	f.Use(recover.New())
	corsOrigins := os.Getenv("CORS_ALLOW_ORIGINS")
	if corsOrigins == "" {
		corsOrigins = "http://localhost:3000"
	}
	f.Use(cors.New(cors.Config{
		AllowOrigins:     corsOrigins,
		AllowCredentials: true,
		AllowHeaders:     "Origin, Content-Type, Accept, X-Principal-User-ID, Cookie, Authorization",
	}))

	httpserver.Mount(f, deps, cfg)

	addr := ":" + cfg.APIPort
	go func() {
		if err := f.Listen(addr); err != nil {
			log.Fatalf("listen: %v", err)
		}
	}()
	fmt.Fprintf(os.Stderr, "api listening on %s\n", addr)

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	_ = f.ShutdownWithContext(ctx)
}
