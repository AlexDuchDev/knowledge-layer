package privacy

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestRehydrateAllowedForType(t *testing.T) {
	ep := &EffectivePolicy{
		ByEntityType: map[SensitiveEntityType]EffectiveRule{
			EntityEmail: {RehydrationMode: RehydrationPartial},
		},
		EffectiveRehydration: RehydrationPartial,
	}
	if !rehydrateAllowedForType(ep, EntityEmail, RehydrationPartial) {
		t.Fatal("partial should allow partial-type")
	}
	if rehydrateAllowedForType(ep, EntityEmail, RehydrationFull) {
		t.Fatal("full should not allow partial-only type")
	}
	if rehydrateAllowedForType(ep, EntityPhone, RehydrationPartial) {
		t.Fatal("unknown type should not rehydrate")
	}
}

func TestOutputPolicyForcesNone(t *testing.T) {
	if !outputPolicyForcesNone("auto_publish", 0) {
		t.Fatal()
	}
	if outputPolicyForcesNone("reviewed_publish", 0) {
		t.Fatal()
	}
	if !outputPolicyForcesNone("", 9) {
		t.Fatal("high sensitivity forces none")
	}
}

func TestRehydrateFromTokenizer_None(t *testing.T) {
	tok := NewPlaceholderTokenizer()
	_ = tok.Placeholder(EntityEmail, "a@b.co")
	ep := &EffectivePolicy{
		ByEntityType: map[SensitiveEntityType]EffectiveRule{
			EntityEmail: {RehydrationMode: RehydrationFull},
		},
		EffectiveRehydration: RehydrationFull,
	}
	out, changed, err := RehydrateFromTokenizer("x EMAIL_1 y", tok, ep, RehydrationNone, nil, context.Background(), uuid.Nil, nil, "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if changed || out != "x EMAIL_1 y" {
		t.Fatalf("got %q", out)
	}
}

func TestRehydrateFromTokenizer_Partial(t *testing.T) {
	tok := NewPlaceholderTokenizer()
	_ = tok.Placeholder(EntityEmail, "a@b.co")
	ep := &EffectivePolicy{
		ByEntityType: map[SensitiveEntityType]EffectiveRule{
			EntityEmail: {RehydrationMode: RehydrationPartial},
		},
		EffectiveRehydration: RehydrationPartial,
	}
	out, changed, err := RehydrateFromTokenizer("x EMAIL_1 y", tok, ep, RehydrationPartial, nil, context.Background(), uuid.Nil, nil, "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if !changed || out != "x a@b.co y" {
		t.Fatalf("got %q changed=%v", out, changed)
	}
}
