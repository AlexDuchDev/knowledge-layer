package knowledge_jobs

import (
	"strings"
)

// ProcessorOutputPlan describes how a processor should create entities, review tasks, and job_outputs rows.
type ProcessorOutputPlan struct {
	EntityLifecycle            string
	CreateReviewTask           bool
	JobOutputPublicationStatus string
}

// PlanWeeklyDigestOutput maps job publication_mode to digest runner behavior.
func PlanWeeklyDigestOutput(job *KnowledgeJob) ProcessorOutputPlan {
	mode := MergeNormalizedPublicationMode(job.PublicationMode)
	// Fail-safe: if review is required at job level, never auto-skip review unless mode is explicitly auto and not review_required (validator blocks that).
	switch mode {
	case PublicationModeDraftOnly:
		return ProcessorOutputPlan{
			EntityLifecycle:            "draft",
			CreateReviewTask:           false,
			JobOutputPublicationStatus: "draft_only",
		}
	case PublicationModeAutoPublish:
		return ProcessorOutputPlan{
			EntityLifecycle:            "published",
			CreateReviewTask:           false,
			JobOutputPublicationStatus: "published",
		}
	case PublicationModeReviewedPublish:
		fallthrough
	default:
		return ProcessorOutputPlan{
			EntityLifecycle:            "pending_review",
			CreateReviewTask:           true,
			JobOutputPublicationStatus: "pending_review",
		}
	}
}

// MergeNormalizedPublicationMode returns API-facing publication_mode (always canonical).
func MergeNormalizedPublicationMode(stored string) string {
	t := strings.TrimSpace(strings.ToLower(stored))
	if t == legacyPublicationModeColumnDraft || t == "" {
		return PublicationModeReviewedPublish
	}
	n := NormalizePublicationMode(stored)
	if n == "" || !isKnownMode(n) {
		return PublicationModeReviewedPublish
	}
	return n
}

func isKnownMode(s string) bool {
	switch strings.ToLower(s) {
	case PublicationModeDraftOnly, PublicationModeReviewedPublish, PublicationModeAutoPublish:
		return true
	default:
		return false
	}
}
