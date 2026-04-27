package embeddings

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/knowledgelayer/api/internal/identity_access"
)

type stubEntityView struct {
	deny map[uuid.UUID]struct{}
}

func (s stubEntityView) Evaluate(ctx context.Context, in identity_access.EvaluateInput) (*identity_access.AccessDecision, error) {
	if in.ResourceID == nil {
		return &identity_access.AccessDecision{Allow: false, SensitivityOK: true}, nil
	}
	if _, banned := s.deny[*in.ResourceID]; banned {
		return &identity_access.AccessDecision{Allow: false, SensitivityOK: true}, nil
	}
	return &identity_access.AccessDecision{Allow: true, SensitivityOK: true}, nil
}

func TestFilterCandidatesByEntityView_dropsDeniedEntity(t *testing.T) {
	ctx := context.Background()
	principal := uuid.MustParse("40000000-0000-0000-0000-000000000001")
	allowed := uuid.MustParse("50000000-0000-0000-0000-000000000001")
	denied := uuid.MustParse("60000000-0000-0000-0000-000000000002")
	stub := stubEntityView{deny: map[uuid.UUID]struct{}{denied: {}}}
	cands := []Candidate{
		{ChunkID: uuid.New(), EntityID: denied, Distance: 0.1},
		{ChunkID: uuid.New(), EntityID: allowed, Distance: 0.2},
	}
	out, err := filterCandidatesByEntityView(ctx, stub, principal, cands, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].EntityID != allowed {
		t.Fatalf("expected single allowed entity, got %#v", out)
	}
}
