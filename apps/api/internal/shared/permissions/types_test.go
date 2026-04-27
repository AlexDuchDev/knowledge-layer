package permissions

import (
	"testing"

	"github.com/google/uuid"
	"github.com/knowledgelayer/api/internal/identity_access"
)

func TestFromAccessDecision(t *testing.T) {
	did := uuid.MustParse("32000000-0000-0000-0000-000000000001")
	s := 2
	d := &identity_access.AccessDecision{
		Allow:                true,
		ReasonCode:           identity_access.ReasonAllowOK,
		MatchedRuleCode:      identity_access.ReasonAllowOK,
		SensitivityResult:    "within_cap(resource_level=1,grant_cap=2)",
		Reasons:              []string{"ok"},
		Trace:                []string{"step:1"},
		SensitivityOK:        true,
		ResolvedDomainID:     &did,
		EffectiveSensitivity: &s,
	}
	r := FromAccessDecision(d)
	if !r.Allowed || r.ReasonCode != identity_access.ReasonAllowOK {
		t.Fatalf("unexpected %+v", r)
	}
	if r.MatchedRuleCode != identity_access.ReasonAllowOK || r.SensitivityResult == "" {
		t.Fatalf("expected matched rule + sensitivity summary, got %+v", r)
	}
	if r.InternalTrace == nil || r.InternalTrace[0] != "step:1" {
		t.Fatalf("expected trace, got %+v", r.InternalTrace)
	}
}
