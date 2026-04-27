package knowledge_jobs

import "testing"

func TestIsKnowledgeJobProcessorImplemented_expandedRunners(t *testing.T) {
	for _, jobType := range []string{
		"weekly_digest",
		"decision_extraction",
		"planning_summary",
		"stale_scan",
		"support_trends_extraction",
	} {
		if !IsKnowledgeJobProcessorImplemented(jobType) {
			t.Fatalf("expected %s to be implemented", jobType)
		}
	}
	if IsKnowledgeJobProcessorImplemented("incident_summary") {
		t.Fatal("incident_summary should still be partial")
	}
}
