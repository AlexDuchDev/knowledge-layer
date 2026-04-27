package privacy

import (
	"context"
	"strings"
)

// SensitiveDataPolicyService resolves effective policy from stored rules and context.
type SensitiveDataPolicyService struct {
	repo *PolicyRepo
}

// NewSensitiveDataPolicyService creates a resolver.
func NewSensitiveDataPolicyService(repo *PolicyRepo) *SensitiveDataPolicyService {
	return &SensitiveDataPolicyService{repo: repo}
}

// ruleMatchesContext returns whether a rule's scope matches the invocation context.
func ruleMatchesContext(r PolicyRule, ctx PolicyContext) bool {
	switch r.ScopeKind {
	case ScopeGlobal:
		return r.ScopeID == nil || strings.TrimSpace(*r.ScopeID) == ""
	case ScopeDomain:
		if ctx.DomainID == nil || r.ScopeID == nil {
			return false
		}
		return strings.EqualFold(strings.TrimSpace(*r.ScopeID), ctx.DomainID.String())
	case ScopeSourceFeed:
		if ctx.SourceFeedID == nil || r.ScopeID == nil {
			return false
		}
		return strings.EqualFold(strings.TrimSpace(*r.ScopeID), ctx.SourceFeedID.String())
	case ScopeScenario:
		if r.ScopeID == nil {
			return false
		}
		return strings.EqualFold(strings.TrimSpace(*r.ScopeID), strings.TrimSpace(ctx.Scenario))
	case ScopeJobType:
		if r.ScopeID == nil {
			return false
		}
		return strings.EqualFold(strings.TrimSpace(*r.ScopeID), strings.TrimSpace(ctx.JobType))
	case ScopeOutputType:
		if r.ScopeID == nil {
			return false
		}
		return strings.EqualFold(strings.TrimSpace(*r.ScopeID), strings.TrimSpace(ctx.OutputType))
	default:
		return false
	}
}

// Resolve merges rules: for each entity type, highest scope tier wins; tie-breaker: higher priority.
// disallow_ai on any matching rule for any type sets DisallowAI on the whole invocation (fail-closed).
func (s *SensitiveDataPolicyService) Resolve(ctx context.Context, pctx PolicyContext) (*EffectivePolicy, error) {
	if s == nil || s.repo == nil {
		return defaultEffectivePolicy(), nil
	}
	rules, err := s.repo.ListEnabled(ctx)
	if err != nil {
		return nil, err
	}

	// Collect matching rules per entity type with best tier+priority.
	type winner struct {
		tier     int
		priority int
		rule     PolicyRule
	}
	best := map[SensitiveEntityType]*winner{}

	for _, r := range rules {
		if !ruleMatchesContext(r, pctx) {
			continue
		}
		tier := scopeTier(r.ScopeKind)
		w, ok := best[r.EntityType]
		if !ok || tier > w.tier || (tier == w.tier && r.Priority > w.priority) {
			best[r.EntityType] = &winner{tier: tier, priority: r.Priority, rule: r}
		}
	}

	out := &EffectivePolicy{
		ByEntityType:         make(map[SensitiveEntityType]EffectiveRule),
		EffectiveRehydration: RehydrationFull,
	}

	for et, w := range best {
		out.ByEntityType[et] = EffectiveRule{
			Action:          w.rule.Action,
			RehydrationMode: w.rule.RehydrationMode,
			SourceTier:      w.tier,
		}
		out.EffectiveRehydration = MinRehydration(out.EffectiveRehydration, w.rule.RehydrationMode)
	}

	if len(best) == 0 {
		out.EffectiveRehydration = RehydrationPartial
	}

	return out, nil
}

// ActionFor returns the action for a type, or tokenize if unknown (safe default).
func (e *EffectivePolicy) ActionFor(t SensitiveEntityType) PolicyAction {
	if e == nil {
		return ActionTokenize
	}
	if r, ok := e.ByEntityType[t]; ok {
		return r.Action
	}
	return ActionTokenize
}

// RehydrationFor returns rehydration mode for a type when rehydrating that placeholder class.
func (e *EffectivePolicy) RehydrationFor(t SensitiveEntityType) RehydrationMode {
	if e == nil {
		return RehydrationNone
	}
	if r, ok := e.ByEntityType[t]; ok {
		return MinRehydration(r.RehydrationMode, e.EffectiveRehydration)
	}
	return MinRehydration(RehydrationNone, e.EffectiveRehydration)
}

func defaultEffectivePolicy() *EffectivePolicy {
	return &EffectivePolicy{
		ByEntityType:         map[SensitiveEntityType]EffectiveRule{},
		EffectiveRehydration: RehydrationPartial,
	}
}

// ResolveActionForType is a convenience when you only have the repo (tests).
func ResolveActionForType(rules []PolicyRule, pctx PolicyContext, t SensitiveEntityType) PolicyAction {
	ep := resolveFromRules(rules, pctx)
	return ep.ActionFor(t)
}

func resolveFromRules(rules []PolicyRule, pctx PolicyContext) *EffectivePolicy {
	type winner struct {
		tier     int
		priority int
		rule     PolicyRule
	}
	best := map[SensitiveEntityType]*winner{}
	for _, r := range rules {
		if !r.Enabled {
			continue
		}
		if !ruleMatchesContext(r, pctx) {
			continue
		}
		tier := scopeTier(r.ScopeKind)
		w, ok := best[r.EntityType]
		if !ok || tier > w.tier || (tier == w.tier && r.Priority > w.priority) {
			best[r.EntityType] = &winner{tier: tier, priority: r.Priority, rule: r}
		}
	}
	out := &EffectivePolicy{
		ByEntityType:         make(map[SensitiveEntityType]EffectiveRule),
		EffectiveRehydration: RehydrationFull,
	}
	for et, w := range best {
		out.ByEntityType[et] = EffectiveRule{
			Action:          w.rule.Action,
			RehydrationMode: w.rule.RehydrationMode,
			SourceTier:      w.tier,
		}
		out.EffectiveRehydration = MinRehydration(out.EffectiveRehydration, w.rule.RehydrationMode)
	}
	if len(best) == 0 {
		out.EffectiveRehydration = RehydrationPartial
	}
	return out
}
