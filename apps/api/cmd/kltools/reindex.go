package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/google/uuid"

	"github.com/knowledgelayer/api/internal/app"
)

// runReindex implements `kltools reindex`. Defaults to dry-run.
//
// Modes:
//
//	--entity UUID            rebuild chunks for one entity, re-enqueue embeddings
//	--all-pending-records    drain normalized_records WHERE chunks_rebuilt_at IS NULL
//	                         (mirrors the connectorworker periodic backfill but
//	                         lets an operator force-converge on demand)
//
// The --all-pending-records mode caps at --batch-size per invocation. Run
// repeatedly until schema-info shows the pending count is zero.
func runReindex(ctx context.Context, deps *app.Deps, args []string) error {
	fs := flag.NewFlagSet("reindex", flag.ContinueOnError)
	var (
		yes               = fs.Bool("yes", false, "actually run (default: dry-run)")
		entityStr         = fs.String("entity", "", "rebuild chunks for one entity UUID")
		allPendingRecords = fs.Bool("all-pending-records", false, "drain normalized_records with chunks_rebuilt_at IS NULL (capped at --batch-size)")
		batchSize         = fs.Int("batch-size", 100, "rows processed per invocation (1-500)")
	)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *batchSize < 1 || *batchSize > 500 {
		return fmt.Errorf("--batch-size must be between 1 and 500")
	}
	if *entityStr == "" && !*allPendingRecords {
		return fmt.Errorf("specify either --entity UUID or --all-pending-records")
	}

	if !*yes {
		fmt.Println("kltools reindex — DRY RUN (pass --yes to actually reindex)")
		if e := *entityStr; e != "" {
			fmt.Printf("  effect: would rebuild chunks for entity %s\n", e)
		}
		if *allPendingRecords {
			fmt.Printf("  effect: would drain up to %d normalized_records pending chunk rebuild\n", *batchSize)
		}
		return nil
	}

	if e := *entityStr; e != "" {
		id, err := uuid.Parse(e)
		if err != nil {
			return fmt.Errorf("--entity: %w", err)
		}
		ids, err := deps.Chunks.RebuildEntityChunks(ctx, id)
		if err != nil {
			return err
		}
		fmt.Printf("kltools reindex — rebuilt %d chunks for entity %s\n", len(ids), id)
		// Embedding tasks are enqueued only when Redis is configured; the
		// chunk service handles that decision internally. Re-trigger the
		// hook to also enqueue.
		if err := deps.Chunks.OnEntityPersisted(ctx, id); err != nil {
			fmt.Printf("kltools reindex — embedding enqueue (best-effort) failed: %v\n", err)
		}
		return nil
	}

	processed, failures, err := deps.Chunks.RebuildPendingNormalizedRecords(ctx, *batchSize)
	if err != nil {
		return err
	}
	fmt.Printf("kltools reindex — processed=%d failures=%d (batch_size=%d)\n", processed, len(failures), *batchSize)
	for i, f := range failures {
		if i >= 5 {
			fmt.Printf("  ... %d more failures suppressed\n", len(failures)-i)
			break
		}
		fmt.Printf("  failure: %v\n", f)
	}
	return nil
}
