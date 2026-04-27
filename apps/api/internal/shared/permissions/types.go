// Package permissions holds shared types for permission resolution across modules.
// Enforcement SQL and the ordered pipeline live in identity_access; this package is for stable DTOs and input builders.
package permissions

import (
	"github.com/google/uuid"

	"github.com/knowledgelayer/api/internal/identity_access"
)

// TargetKind identifies the governed object class for documentation and tracing (maps to EvaluateInput.ResourceType).
type TargetKind string

const (
	TargetDomain     TargetKind = "domain"
	TargetEntityType TargetKind = "entity_type"
	TargetEntity     TargetKind = "entity"
	TargetSourceFeed TargetKind = "source_feed"
	TargetJob        TargetKind = "job"
	TargetJobOutput  TargetKind = "job_output"
)

// ResolutionResult is a portable view of AccessDecision for APIs, jobs, and retrieval layers.
type ResolutionResult struct {
	Allowed              bool        `json:"allowed"`
	ReasonCode           string      `json:"reason_code"`
	Reasons              []string    `json:"reasons"`
	MatchedPolicies      []uuid.UUID `json:"matched_policies,omitempty"`
	MatchedOverrides     []uuid.UUID `json:"matched_overrides,omitempty"`
	SensitivityOK        bool        `json:"sensitivity_ok"`
	EffectiveSensitivity *int        `json:"effective_sensitivity,omitempty"`
	ResolvedDomainID     *uuid.UUID  `json:"resolved_domain_id,omitempty"`
	// MatchedRuleCode mirrors AccessDecision for audit UIs (often equals ReasonCode).
	MatchedRuleCode string `json:"matched_rule_code,omitempty"`
	// SensitivityResult summarizes the sensitivity phase when evaluated (e.g. within_cap / exceeds_cap).
	SensitivityResult string   `json:"sensitivity_result,omitempty"`
	InternalTrace     []string `json:"internal_trace,omitempty"`
}

// FromAccessDecision maps the evaluator outcome (trace included only when non-empty, for internal consumers).
func FromAccessDecision(d *identity_access.AccessDecision) *ResolutionResult {
	if d == nil {
		return &ResolutionResult{Allowed: false, ReasonCode: identity_access.ReasonDenyMissingPrincipal, Reasons: []string{"nil decision"}}
	}
	r := &ResolutionResult{
		Allowed:              d.Allow,
		ReasonCode:           d.ReasonCode,
		Reasons:              append([]string(nil), d.Reasons...),
		MatchedPolicies:      append([]uuid.UUID(nil), d.MatchedPolicies...),
		MatchedOverrides:     append([]uuid.UUID(nil), d.MatchedOverrides...),
		SensitivityOK:        d.SensitivityOK,
		EffectiveSensitivity: d.EffectiveSensitivity,
		ResolvedDomainID:     d.ResolvedDomainID,
		MatchedRuleCode:      d.MatchedRuleCode,
		SensitivityResult:    d.SensitivityResult,
	}
	if len(d.Trace) > 0 {
		r.InternalTrace = append([]string(nil), d.Trace...)
	}
	return r
}
