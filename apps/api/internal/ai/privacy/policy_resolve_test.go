package privacy

import (
	"testing"

	"github.com/google/uuid"
)

func TestResolveFromRules_GlobalEmailTokenize(t *testing.T) {
	rules := []PolicyRule{
		{ScopeKind: ScopeGlobal, EntityType: EntityEmail, Action: ActionTokenize, RehydrationMode: RehydrationPartial, Enabled: true, Priority: 50},
	}
	pctx := PolicyContext{Scenario: "ask_global", OutputType: "qa_answer"}
	ep := resolveFromRules(rules, pctx)
	if ep.ActionFor(EntityEmail) != ActionTokenize {
		t.Fatalf("email action: got %v", ep.ActionFor(EntityEmail))
	}
}

func TestResolveFromRules_DomainOverridesGlobal(t *testing.T) {
	did := uuid.New()
	sid := did.String()
	rules := []PolicyRule{
		{ScopeKind: ScopeGlobal, EntityType: EntityCompanyName, Action: ActionTokenize, RehydrationMode: RehydrationPartial, Enabled: true, Priority: 40},
		{ScopeKind: ScopeDomain, ScopeID: &sid, EntityType: EntityCompanyName, Action: ActionKeep, RehydrationMode: RehydrationFull, Enabled: true, Priority: 10},
	}
	pctx := PolicyContext{DomainID: &did, OutputType: "qa_answer"}
	ep := resolveFromRules(rules, pctx)
	if ep.ActionFor(EntityCompanyName) != ActionKeep {
		t.Fatalf("expected domain keep, got %v", ep.ActionFor(EntityCompanyName))
	}
}

func TestResolveFromRules_OutputTypeOverridesDomain(t *testing.T) {
	did := uuid.New()
	ds := did.String()
	out := "qa_answer"
	rules := []PolicyRule{
		{ScopeKind: ScopeDomain, ScopeID: &ds, EntityType: EntityEmail, Action: ActionKeep, RehydrationMode: RehydrationFull, Enabled: true, Priority: 99},
		{ScopeKind: ScopeOutputType, ScopeID: &out, EntityType: EntityEmail, Action: ActionRemove, RehydrationMode: RehydrationNone, Enabled: true, Priority: 1},
	}
	pctx := PolicyContext{DomainID: &did, OutputType: "qa_answer"}
	ep := resolveFromRules(rules, pctx)
	if ep.ActionFor(EntityEmail) != ActionRemove {
		t.Fatalf("expected output_type remove, got %v", ep.ActionFor(EntityEmail))
	}
}

func TestResolveFromRules_DisallowPerType(t *testing.T) {
	rules := []PolicyRule{
		{ScopeKind: ScopeGlobal, EntityType: EntitySecuritySecret, Action: ActionDisallowAI, RehydrationMode: RehydrationNone, Enabled: true, Priority: 100},
		{ScopeKind: ScopeGlobal, EntityType: EntityEmail, Action: ActionTokenize, RehydrationMode: RehydrationPartial, Enabled: true, Priority: 50},
	}
	pctx := PolicyContext{}
	ep := resolveFromRules(rules, pctx)
	if ep.ActionFor(EntitySecuritySecret) != ActionDisallowAI {
		t.Fatalf("secret: got %v", ep.ActionFor(EntitySecuritySecret))
	}
	if ep.ActionFor(EntityEmail) != ActionTokenize {
		t.Fatalf("email: got %v", ep.ActionFor(EntityEmail))
	}
}

func TestMinRehydration(t *testing.T) {
	if MinRehydration(RehydrationNone, RehydrationFull) != RehydrationNone {
		t.Fatal()
	}
	if MinRehydration(RehydrationFull, RehydrationPartial) != RehydrationPartial {
		t.Fatal()
	}
}
