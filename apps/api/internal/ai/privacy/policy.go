package privacy

import (
	"github.com/google/uuid"
)

// PolicyScopeKind defines where a rule applies.
type PolicyScopeKind string

const (
	ScopeGlobal     PolicyScopeKind = "global"
	ScopeDomain     PolicyScopeKind = "domain"
	ScopeSourceFeed PolicyScopeKind = "source_feed"
	ScopeScenario   PolicyScopeKind = "scenario"
	ScopeJobType    PolicyScopeKind = "job_type"
	ScopeOutputType PolicyScopeKind = "output_type"
)

// PolicyAction is applied to detected sensitive content before LLM.
type PolicyAction string

const (
	ActionKeep       PolicyAction = "keep"
	ActionTokenize   PolicyAction = "tokenize"
	ActionRemove     PolicyAction = "remove"
	ActionDisallowAI PolicyAction = "disallow_ai"
)

// RehydrationMode controls post-LLM restoration of placeholders.
type RehydrationMode string

const (
	RehydrationNone    RehydrationMode = "none"
	RehydrationPartial RehydrationMode = "partial"
	RehydrationFull    RehydrationMode = "full"
)

// PolicyRule is one row from ai_privacy_policy_rules.
type PolicyRule struct {
	ID              uuid.UUID
	ScopeKind       PolicyScopeKind
	ScopeID         *string
	EntityType      SensitiveEntityType
	Action          PolicyAction
	RehydrationMode RehydrationMode
	Priority        int
	Enabled         bool
}

// PolicyContext carries invocation scope for resolution.
type PolicyContext struct {
	DomainID     *uuid.UUID
	SourceFeedID *uuid.UUID
	Scenario     string // e.g. ask_entity, ask_global, ai_summarize
	JobType      string
	OutputType   string // e.g. qa_answer, summary, extraction
}

// EffectivePolicy is the merged result per entity type.
type EffectivePolicy struct {
	// ByEntityType maps each sensitive type to resolved action and rehydration.
	ByEntityType map[SensitiveEntityType]EffectiveRule
	// EffectiveRehydration is the most restrictive mode across all scoped rules that matched (for output caps).
	EffectiveRehydration RehydrationMode
}

// EffectiveRule is the winning rule for one entity type.
type EffectiveRule struct {
	Action          PolicyAction
	RehydrationMode RehydrationMode
	SourceTier      int // for debugging / traces
}

// scopeTier ranks specificity: higher wins. Order: output_type > job_type > scenario > source_feed > domain > global
func scopeTier(k PolicyScopeKind) int {
	switch k {
	case ScopeOutputType:
		return 6
	case ScopeJobType:
		return 5
	case ScopeScenario:
		return 4
	case ScopeSourceFeed:
		return 3
	case ScopeDomain:
		return 2
	case ScopeGlobal:
		return 1
	default:
		return 0
	}
}

func rehydrationRank(m RehydrationMode) int {
	switch m {
	case RehydrationFull:
		return 3
	case RehydrationPartial:
		return 2
	case RehydrationNone:
		return 1
	default:
		return 0
	}
}

// MinRehydration returns the more restrictive of two modes (none < partial < full for restriction).
func MinRehydration(a, b RehydrationMode) RehydrationMode {
	if rehydrationRank(a) <= rehydrationRank(b) {
		return a
	}
	return b
}
