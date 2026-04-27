package qa

import (
	"testing"

	"github.com/google/uuid"
	"github.com/knowledgelayer/api/internal/knowledge_core"
)

func TestOrderEvidenceForAsk_approvedBeforeNone(t *testing.T) {
	root := uuid.MustParse("10000000-0000-0000-0000-000000000001")
	a := &knowledge_core.Entity{ID: uuid.MustParse("20000000-0000-0000-0000-000000000002"), TruthMode: "derived", LifecycleState: "draft", FreshnessStatus: "unknown", ApprovalStatus: "none"}
	b := &knowledge_core.Entity{ID: uuid.MustParse("30000000-0000-0000-0000-000000000003"), TruthMode: "derived", LifecycleState: "draft", FreshnessStatus: "unknown", ApprovalStatus: "approved"}
	r := &knowledge_core.Entity{ID: root, TruthMode: "derived", LifecycleState: "draft", FreshnessStatus: "unknown", ApprovalStatus: "none"}
	evidence := []*knowledge_core.Entity{r, a, b}
	OrderEvidenceForAsk(root, evidence)
	if evidence[0].ID != root {
		t.Fatalf("root not first")
	}
	if evidence[1].ID != b.ID {
		t.Fatalf("expected approved related before none, got %s", evidence[1].ID)
	}
}

func TestOrderEvidenceForAsk_rootPinned(t *testing.T) {
	root := uuid.MustParse("10000000-0000-0000-0000-000000000001")
	weak := &knowledge_core.Entity{ID: uuid.MustParse("20000000-0000-0000-0000-000000000002"), TruthMode: "derived", LifecycleState: "draft", FreshnessStatus: "stale"}
	strong := &knowledge_core.Entity{ID: uuid.MustParse("30000000-0000-0000-0000-000000000003"), TruthMode: "canonical_in_platform", LifecycleState: "published", FreshnessStatus: "fresh"}
	r := &knowledge_core.Entity{ID: root, TruthMode: "derived", LifecycleState: "draft", FreshnessStatus: "unknown"}
	evidence := []*knowledge_core.Entity{r, weak, strong}
	OrderEvidenceForAsk(root, evidence)
	if evidence[0].ID != root {
		t.Fatalf("root not first")
	}
	if evidence[1].ID != strong.ID {
		t.Fatalf("expected stronger related before weaker, got %s then %s", evidence[1].ID, evidence[2].ID)
	}
}
