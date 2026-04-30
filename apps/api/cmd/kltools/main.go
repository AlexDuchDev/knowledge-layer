// kltools is the operator CLI for Knowledge Layer. It runs against the same
// composition root as the API server (app.NewDeps) but exits after a single
// subcommand instead of starting an HTTP server.
//
// Subcommands:
//
//	summarize     — backfill entity_search_projection.synthesized_summary
//	                via the entity_summarize knowledge job processor.
//	reindex       — rebuild chunks (entity-rooted or normalized-record-rooted)
//	                and re-enqueue embedding tasks.
//	schema-info   — print pipeline-stage counts and the implemented job_type
//	                inventory. Read-only; safe in any environment.
//
// All write subcommands require an explicit --yes flag, otherwise they
// run as a dry preview. This is on purpose: the CLI ships inside the same
// container image as the API server, so it's easy to fat-finger a prod run.
//
// Run via: docker compose exec api /app/knowledge-tools <subcommand>
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/knowledgelayer/api/internal/app"
	"github.com/knowledgelayer/api/internal/config"
	"github.com/knowledgelayer/api/internal/db"
)

func main() {
	if len(os.Args) < 2 {
		printUsage(os.Stderr)
		os.Exit(2)
	}
	sub := os.Args[1]
	args := os.Args[2:]

	switch sub {
	case "summarize":
		runSubcommand(args, runSummarize)
	case "reindex":
		runSubcommand(args, runReindex)
	case "schema-info":
		runSubcommand(args, runSchemaInfo)
	case "help", "-h", "--help":
		printUsage(os.Stdout)
		return
	default:
		fmt.Fprintf(os.Stderr, "kltools: unknown subcommand %q\n\n", sub)
		printUsage(os.Stderr)
		os.Exit(2)
	}
}

// runSubcommand is the shared bootstrap for every subcommand: load config,
// open a small DB pool (4 conns — never starve the running API), build deps,
// dispatch. Errors crash the process so CI exit codes are correct.
func runSubcommand(args []string, fn func(ctx context.Context, deps *app.Deps, args []string) error) {
	ctx := context.Background()
	cfg := config.Load()
	dsn := cfg.DatabaseURL
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		dsn = "postgres://knowledge:knowledge@localhost:5432/knowledge?sslmode=disable"
	}
	// Run migrations on demand — the CLI is the only operator-side tool
	// that might be the first thing to touch a fresh DB before the API
	// starts. Idempotent so repeating is fine.
	if err := db.MigrateUp(dsn); err != nil {
		log.Fatalf("kltools: migrate: %v", err)
	}
	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Fatalf("kltools: db parse: %v", err)
	}
	// Cap at 4 connections so the CLI never starves the running API. The
	// API typically holds ~25 connections; this leaves them alone.
	poolCfg.MaxConns = 4
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		log.Fatalf("kltools: db: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("kltools: db ping: %v", err)
	}
	deps, err := app.NewDeps(pool, cfg)
	if err != nil {
		log.Fatalf("kltools: deps: %v", err)
	}
	defer func() {
		if deps.JobQueue != nil {
			_ = deps.JobQueue.Close()
		}
	}()

	if err := fn(ctx, deps, args); err != nil {
		log.Fatalf("kltools: %v", err)
	}
}

func printUsage(w *os.File) {
	fmt.Fprintf(w, `kltools — Knowledge Layer operator CLI

Usage:
  kltools <subcommand> [flags]

Subcommands:
  summarize       Backfill entity_search_projection.synthesized_summary via
                  the entity_summarize knowledge job processor. Routes through
                  the privacy gateway. Defaults to dry-run; pass --yes to write.
  reindex         Rebuild chunks for entities or normalized records. Re-enqueues
                  embedding tasks when Redis is configured. Defaults to dry-run.
  schema-info     Print pipeline-stage counts and the implemented job_type
                  inventory. Read-only; safe to run anywhere.

Common flags:
  --yes           Confirm a write subcommand. Without this, summarize and
                  reindex print what they would do and exit.
  --max-rows N    Upper bound on rows processed per run.
  --domain UUID   Restrict to a single domain (where applicable).
  --entity UUID   Process a specific entity (where applicable; repeatable).

Run a subcommand with --help for its specific flag set.
`)
}
