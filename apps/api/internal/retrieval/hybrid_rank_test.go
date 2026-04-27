package retrieval

import "testing"

func TestHybridFusionScore_approvedBeatsDraftWhenNormalizedScoresTie(t *testing.T) {
	kw, sem := 0.5, 0.5
	wKw, wSem, pW := 0.45, 0.55, 0.02
	penApproved := GovernancePenalty("derived", "draft", "unknown", "approved")
	penDraft := GovernancePenalty("derived", "draft", "unknown", "none")
	sApproved := HybridFusionScore(kw, sem, penApproved, wKw, wSem, pW)
	sDraft := HybridFusionScore(kw, sem, penDraft, wKw, wSem, pW)
	if sApproved <= sDraft {
		t.Fatalf("expected approved score higher than draft, got approved=%v draft=%v", sApproved, sDraft)
	}
}
