package qa

import (
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/knowledgelayer/api/internal/knowledge_core"
)

func entityTrustRank(e *knowledge_core.Entity) int {
	r := 0
	switch e.TruthMode {
	case "canonical_in_platform":
		r += 0
	case "mirrored_authority":
		r += 10
	default:
		r += 20
	}
	if e.LifecycleState == "published" {
		r += 0
	} else if e.LifecycleState == "pending_review" || e.LifecycleState == "in_review" {
		r += 5
	} else {
		r += 8
	}
	switch e.FreshnessStatus {
	case "fresh", "current", "ok":
		r += 0
	case "unknown":
		r += 3
	default:
		r += 6
	}
	switch strings.ToLower(strings.TrimSpace(e.ApprovalStatus)) {
	case "approved":
		r += 0
	default:
		r += 6
	}
	return r
}

// OrderEvidenceForAsk keeps root first, sorts related entities by trust rank (explainable heuristic).
func OrderEvidenceForAsk(rootID uuid.UUID, evidence []*knowledge_core.Entity) {
	if len(evidence) <= 1 {
		return
	}
	root := evidence[0]
	if root.ID != rootID {
		// find root and move to front
		for i, e := range evidence {
			if e.ID == rootID {
				evidence[0], evidence[i] = evidence[i], evidence[0]
				root = evidence[0]
				break
			}
		}
	}
	rest := evidence[1:]
	sort.SliceStable(rest, func(i, j int) bool {
		ri, rj := entityTrustRank(rest[i]), entityTrustRank(rest[j])
		if ri != rj {
			return ri < rj
		}
		return rest[i].ID.String() < rest[j].ID.String()
	})
	copy(evidence[1:], rest)
	_ = root
}
