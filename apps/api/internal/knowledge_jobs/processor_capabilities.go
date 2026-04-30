package knowledge_jobs

import "strings"

// ImplementedKnowledgeJobTypes lists job_type values that have a runtime processor
// in runOrchestrator.executeProcessor. Keep in sync with orchestrator.go.
var ImplementedKnowledgeJobTypes = []string{
	"weekly_digest",
	"decision_extraction",
	"planning_summary",
	"stale_scan",
	"support_trends_extraction",
	"entity_summarize",
}

// JobTypeCapability describes executor support for a catalog job_type (API + docs alignment).
type JobTypeCapability struct {
	JobType           string `json:"job_type"`
	Implemented       bool   `json:"implemented"`
	FailClosedMessage string `json:"fail_closed_message,omitempty"`
}

// KnowledgeJobsEngineMetadata is returned by GET /knowledge-jobs/engine-metadata.
type KnowledgeJobsEngineMetadata struct {
	ImplementedJobTypes []string            `json:"implemented_job_types"`
	JobTypeCapabilities []JobTypeCapability `json:"job_type_capabilities"`
}

// IsKnowledgeJobProcessorImplemented reports whether a queued run can execute for job_type.
func IsKnowledgeJobProcessorImplemented(jobType string) bool {
	jt := strings.TrimSpace(strings.ToLower(jobType))
	switch jt {
	case "weekly_digest":
		return true
	case "decision_extraction":
		return true
	case "planning_summary":
		return true
	case "stale_scan":
		return true
	case "support_trends_extraction":
		return true
	case "entity_summarize":
		return true
	default:
		return false
	}
}

// FailClosedMessageForJobType is the user-facing reason when a type is not runnable.
func FailClosedMessageForJobType(jobType string) string {
	if IsKnowledgeJobProcessorImplemented(jobType) {
		return ""
	}
	return "This job_type has no runtime processor yet; runs will fail until implemented."
}

// KnowledgeJobsEngineMetadataResponse builds the public engine contract for clients.
func KnowledgeJobsEngineMetadataResponse() KnowledgeJobsEngineMetadata {
	impl := make([]string, len(ImplementedKnowledgeJobTypes))
	copy(impl, ImplementedKnowledgeJobTypes)
	return KnowledgeJobsEngineMetadata{
		ImplementedJobTypes: impl,
		JobTypeCapabilities: listJobTypeCapabilitiesFromCatalog(),
	}
}

func listJobTypeCapabilitiesFromCatalog() []JobTypeCapability {
	seen := make(map[string]struct{})
	out := make([]JobTypeCapability, 0, len(jobTemplates))
	for _, t := range jobTemplates {
		jt := strings.TrimSpace(t.JobType)
		if jt == "" {
			continue
		}
		if _, ok := seen[jt]; ok {
			continue
		}
		seen[jt] = struct{}{}
		out = append(out, JobTypeCapability{
			JobType:           jt,
			Implemented:       IsKnowledgeJobProcessorImplemented(jt),
			FailClosedMessage: FailClosedMessageForJobType(jt),
		})
	}
	return out
}
