package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"

	"github.com/google/uuid"

	"github.com/knowledgelayer/api/internal/app"
	"github.com/knowledgelayer/api/internal/knowledge_jobs"
)

// runSummarize implements `kltools summarize`. Defaults to dry-run.
func runSummarize(ctx context.Context, deps *app.Deps, args []string) error {
	fs := flag.NewFlagSet("summarize", flag.ContinueOnError)
	var (
		yes       = fs.Bool("yes", false, "actually run (without this flag, only the plan is printed)")
		maxRows   = fs.Int("max-rows", 100, "upper bound on entities processed (hard cap 500)")
		domainStr = fs.String("domain", "", "restrict to a single domain UUID")
		entitiesS = stringSliceFlag{}
	)
	fs.Var(&entitiesS, "entity", "process a specific entity UUID (repeatable; bypasses --max-rows)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	// Build the job scope blob. EntitySummarizeScope shape lives in
	// knowledge_jobs/entity_summarize.go.
	scope := knowledge_jobs.EntitySummarizeScope{MaxRows: *maxRows}
	if d := *domainStr; d != "" {
		id, err := uuid.Parse(d)
		if err != nil {
			return fmt.Errorf("--domain: %w", err)
		}
		scope.DomainID = &id
	}
	for _, s := range entitiesS {
		id, err := uuid.Parse(s)
		if err != nil {
			return fmt.Errorf("--entity %q: %w", s, err)
		}
		scope.EntityIDs = append(scope.EntityIDs, id)
	}
	scopeJSON, _ := json.Marshal(scope)

	if !*yes {
		fmt.Println("kltools summarize — DRY RUN (pass --yes to actually summarize)")
		fmt.Printf("  scope: %s\n", string(scopeJSON))
		fmt.Println("  effect: would route each pending entity through privacy gateway,")
		fmt.Println("          write synthesized_summary to entity_search_projection.")
		return nil
	}
	if deps.PrivacyGateway == nil {
		return fmt.Errorf("privacy gateway not configured (set OPENAI_API_KEY or OPENROUTER_API_KEY)")
	}

	job := &knowledge_jobs.KnowledgeJob{
		JobType:         "entity_summarize",
		SourceScopeJSON: scopeJSON,
	}
	runID := uuid.New()
	// Use a synthetic operator UUID; audit will tag this as system actor.
	operator := uuid.MustParse("00000000-0000-0000-0000-00000000c11e")

	summarizer := knowledge_jobs.NewEntitySummarizer(deps.Pool, deps.PrivacyGateway)
	if err := summarizer.RunEntitySummarize(ctx, runID, job, operator); err != nil {
		return err
	}
	fmt.Printf("kltools summarize — completed (run_id=%s)\n", runID)
	return nil
}

// stringSliceFlag implements flag.Value for repeatable string flags
// (e.g. --entity X --entity Y).
type stringSliceFlag []string

func (s *stringSliceFlag) String() string     { return fmt.Sprintf("%v", []string(*s)) }
func (s *stringSliceFlag) Set(v string) error { *s = append(*s, v); return nil }
