package search

import (
	"testing"

	"github.com/google/uuid"
)

func TestSortHitsByTrust_approvedBeforeNone(t *testing.T) {
	a := uuid.MustParse("10000000-0000-0000-0000-000000000001")
	b := uuid.MustParse("20000000-0000-0000-0000-000000000002")
	base := Hit{
		TruthMode: "derived", LifecycleState: "draft", FreshnessStatus: "unknown",
	}
	hits := []Hit{
		{EntityID: b, TruthMode: base.TruthMode, LifecycleState: base.LifecycleState, FreshnessStatus: base.FreshnessStatus, ApprovalStatus: "none"},
		{EntityID: a, TruthMode: base.TruthMode, LifecycleState: base.LifecycleState, FreshnessStatus: base.FreshnessStatus, ApprovalStatus: "approved"},
	}
	SortHitsByTrust(hits)
	if hits[0].EntityID != a {
		t.Fatalf("expected approved first, got %s", hits[0].EntityID)
	}
}

func TestSortHitsByTrust_canonicalBeforeDerived(t *testing.T) {
	a := uuid.MustParse("10000000-0000-0000-0000-000000000001")
	b := uuid.MustParse("20000000-0000-0000-0000-000000000002")
	hits := []Hit{
		{EntityID: b, TruthMode: "derived", LifecycleState: "draft", FreshnessStatus: "unknown"},
		{EntityID: a, TruthMode: "canonical_in_platform", LifecycleState: "published", FreshnessStatus: "fresh"},
	}
	SortHitsByTrust(hits)
	if hits[0].EntityID != a {
		t.Fatalf("expected canonical first, got %s", hits[0].EntityID)
	}
}
