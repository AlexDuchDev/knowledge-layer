package main

import (
	"context"
	"fmt"

	"github.com/knowledgelayer/api/internal/app"
	"github.com/knowledgelayer/api/internal/knowledge_jobs"
)

// runSchemaInfo implements `kltools schema-info`. Read-only by construction;
// safe to run in any environment. Output is the operator's quick-glance map:
// pipeline-stage counts (raw_artifacts, normalized_records, entities, chunks,
// embeddings, pending-rebuild rows, pending-summary rows) plus the registered
// connector inventory and implemented job_type list.
//
// Pattern matches Hugr's `hugr-tools schema-info` — operators want one
// command that says "what does this instance look like right now?" without
// poking around in psql.
func runSchemaInfo(ctx context.Context, deps *app.Deps, args []string) error {
	pool := deps.Pool
	type row struct {
		label string
		query string
	}
	rows := []row{
		{"raw_artifacts", "SELECT count(*) FROM raw_artifacts"},
		{"normalized_records", "SELECT count(*) FROM normalized_records"},
		{"  pending chunk rebuild", "SELECT count(*) FROM normalized_records WHERE chunks_rebuilt_at IS NULL"},
		{"entities", "SELECT count(*) FROM entities WHERE archived_at IS NULL"},
		{"  pending synthesized_summary", "SELECT count(*) FROM entity_search_projection WHERE synthesized_summary IS NULL"},
		{"chunks", "SELECT count(*) FROM chunks"},
		{"  entity-rooted", "SELECT count(*) FROM chunks WHERE entity_id IS NOT NULL"},
		{"  normalized_record-rooted", "SELECT count(*) FROM chunks WHERE normalized_record_id IS NOT NULL"},
		{"embeddings", "SELECT count(*) FROM embeddings"},
		{"audit_events", "SELECT count(*) FROM audit_events"},
		{"source_feeds", "SELECT count(*) FROM source_feeds"},
		{"connectors", "SELECT count(*) FROM connectors"},
	}
	fmt.Println("=== Pipeline stage counts ===")
	for _, r := range rows {
		var n int64
		if err := pool.QueryRow(ctx, r.query).Scan(&n); err != nil {
			fmt.Printf("  %-32s ERROR: %v\n", r.label, err)
			continue
		}
		fmt.Printf("  %-32s %d\n", r.label, n)
	}

	fmt.Println()
	fmt.Println("=== Implemented job types ===")
	for _, jt := range knowledge_jobs.ImplementedKnowledgeJobTypes {
		fmt.Printf("  - %s\n", jt)
	}

	fmt.Println()
	fmt.Println("=== Registered connectors ===")
	connRows, err := pool.Query(ctx, `SELECT type, display_name, status FROM connectors ORDER BY type`)
	if err != nil {
		return err
	}
	defer connRows.Close()
	for connRows.Next() {
		var typ, name, status string
		if err := connRows.Scan(&typ, &name, &status); err != nil {
			return err
		}
		fmt.Printf("  - %-20s %-30s [%s]\n", typ, name, status)
	}

	fmt.Println()
	fmt.Println("=== Cache state ===")
	if deps.Cache == nil {
		fmt.Println("  cache: not constructed (NewDeps returned nil — bug)")
	} else {
		fmt.Println("  cache: constructed (Cache.Null when CACHE_L1_ENABLED=false)")
	}
	return connRows.Err()
}
