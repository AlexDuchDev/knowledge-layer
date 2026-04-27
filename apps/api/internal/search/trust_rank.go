package search

import (
	"sort"
	"strings"
)

// trustRank returns lower-is-better ordering for explainable trust-aware ranking.
func trustRank(h Hit) int {
	r := 0
	switch h.TruthMode {
	case "canonical_in_platform":
		r += 0
	case "mirrored_authority":
		r += 10
	default:
		r += 20
	}
	if h.LifecycleState == "published" {
		r += 0
	} else if h.LifecycleState == "pending_review" || h.LifecycleState == "in_review" {
		r += 5
	} else {
		r += 8
	}
	switch h.FreshnessStatus {
	case "fresh", "current", "ok":
		r += 0
	case "unknown":
		r += 3
	default:
		r += 6
	}
	switch strings.ToLower(strings.TrimSpace(h.ApprovalStatus)) {
	case "approved":
		r += 0
	default:
		r += 6
	}
	if h.RelationExpansion != "" {
		r += 15
	}
	return r
}

// SortHitsByTrust reorders hits for display: stronger trust first, stable tie-break by entity_id.
func SortHitsByTrust(hits []Hit) {
	sort.SliceStable(hits, func(i, j int) bool {
		ri, rj := trustRank(hits[i]), trustRank(hits[j])
		if ri != rj {
			return ri < rj
		}
		return hits[i].EntityID.String() < hits[j].EntityID.String()
	})
}
