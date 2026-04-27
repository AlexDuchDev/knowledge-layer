package retrieval

import (
	"strings"

	"github.com/knowledgelayer/api/internal/search"
)

// GovernancePenalty is a lower-is-better trust score (aligned with search.trustRank).
func GovernancePenalty(truth, lifecycle, fresh, approval string) int {
	r := 0
	switch truth {
	case "canonical_in_platform":
		r += 0
	case "mirrored_authority":
		r += 10
	default:
		r += 20
	}
	if lifecycle == "published" {
		r += 0
	} else if lifecycle == "pending_review" || lifecycle == "in_review" {
		r += 5
	} else {
		r += 8
	}
	switch fresh {
	case "fresh", "current", "ok":
		r += 0
	case "unknown":
		r += 3
	default:
		r += 6
	}
	switch strings.ToLower(strings.TrimSpace(approval)) {
	case "approved":
		r += 0
	default:
		r += 6
	}
	return r
}

// GovernancePenaltyFromHit wraps search.Hit signals.
func GovernancePenaltyFromHit(h search.Hit) int {
	return GovernancePenalty(h.TruthMode, h.LifecycleState, h.FreshnessStatus, h.ApprovalStatus)
}

// SemanticSimilarity maps pgvector cosine distance to [0,1] (higher is better).
func SemanticSimilarity(cosineDistance float64) float64 {
	s := 1.0 - cosineDistance/2.0
	if s < 0 {
		return 0
	}
	if s > 1 {
		return 1
	}
	return s
}

// HybridFusionScore combines normalized keyword and semantic legs minus a scaled governance penalty.
func HybridFusionScore(kwNorm, semNorm float64, penalty int, wKw, wSem, penaltyWeight float64) float64 {
	return wKw*kwNorm + wSem*semNorm - penaltyWeight*float64(penalty)
}
