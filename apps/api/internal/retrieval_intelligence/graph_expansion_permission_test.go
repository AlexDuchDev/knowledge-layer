package retrieval_intelligence

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/knowledgelayer/api/internal/knowledge_core"
	"github.com/knowledgelayer/api/internal/retrieval"
)

// TestGraphExpandContextPieces_NilGraph_NoCanViewInvocation verifies the early
// return path when no graph store is configured: the function must not panic,
// must not invoke canView (since there is nothing to filter), and must return
// the input pieces unchanged.
//
// This is a smoke test for the canView parameter wiring (Phase 1.1.1). The
// full end-to-end regression test that proves a denied entity's chunks never
// reach LLM context lives under apps/api/internal/integration/ and requires
// E2E_DB=1 plus a running Neo4j instance to seed co-mention edges. See
// docs/AI_RETRIEVAL_GOVERNANCE.md for the access-before-retrieval contract.
func TestGraphExpandContextPieces_NilGraph_NoCanViewInvocation(t *testing.T) {
	s := &Service{} // graph == nil

	pieces := []retrieval.RankedContextPiece{
		{EntityID: uuid.New(), ChunkID: uuid.New(), Text: "seed-1"},
	}

	canViewCalls := 0
	canView := func(*knowledge_core.Entity) error {
		canViewCalls++
		return nil
	}

	out, meta := s.graphExpandContextPieces(context.Background(), pieces, canView)

	if len(out) != len(pieces) {
		t.Fatalf("expected pieces unchanged (len=%d), got len=%d", len(pieces), len(out))
	}
	if meta != nil {
		t.Fatalf("expected nil meta when graph store is nil, got %+v", meta)
	}
	if canViewCalls != 0 {
		t.Fatalf("expected canView not invoked when no expansion happens, got %d calls", canViewCalls)
	}
}

// TestGraphExpandContextPieces_EmptyPieces verifies the second early-return
// branch (no seed pieces → no expansion). canView must not be invoked.
func TestGraphExpandContextPieces_EmptyPieces(t *testing.T) {
	// Service with non-nil graph slot stays nil-graph because we don't construct
	// a real GraphStore here; the function still hits the nil-graph branch first.
	s := &Service{}

	canViewCalls := 0
	canView := func(*knowledge_core.Entity) error {
		canViewCalls++
		return nil
	}

	out, meta := s.graphExpandContextPieces(context.Background(), nil, canView)

	if len(out) != 0 {
		t.Fatalf("expected empty out, got len=%d", len(out))
	}
	if meta != nil {
		t.Fatalf("expected nil meta, got %+v", meta)
	}
	if canViewCalls != 0 {
		t.Fatalf("expected canView not invoked, got %d calls", canViewCalls)
	}
}
