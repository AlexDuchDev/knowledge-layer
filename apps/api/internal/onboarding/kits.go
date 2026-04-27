package onboarding

import (
	"fmt"
)

// DomainSetupKit describes repeatable onboarding defaults (documentation + light API hints).
type DomainSetupKit struct {
	ID                      string   `json:"id"`
	Title                   string   `json:"title"`
	Description             string   `json:"description"`
	RecommendedRoles        []string `json:"recommended_roles"`
	DefaultSensitivityLevel int      `json:"default_sensitivity_level"`
	SourceFeedHints         []string `json:"source_feed_hints"`
	JobTemplateIDs          []string `json:"job_template_ids"`
	GovernanceNotes         string   `json:"governance_notes"`
}

// BuiltinKits returns static kits aligned with current job templates and roles vocabulary.
func BuiltinKits() []DomainSetupKit {
	return []DomainSetupKit{
		{
			ID:                      "general",
			Title:                   "General team domain",
			Description:             "Balanced defaults for a cross-functional knowledge domain.",
			RecommendedRoles:        []string{"Domain Owner", "Reviewer", "Viewer"},
			DefaultSensitivityLevel: 1,
			SourceFeedHints:         []string{"Google Drive folder (policies)", "Slack export (optional)", "Meeting notes repository"},
			JobTemplateIDs:          []string{"weekly_digest"},
			GovernanceNotes:         "Start with draft + review for canonical decisions; use mirrored_authority for synced docs.",
		},
		{
			ID:                      "engineering",
			Title:                   "Engineering / product",
			Description:             "Emphasis on decisions, SOPs, and incident learnings.",
			RecommendedRoles:        []string{"Domain Owner", "Reviewer", "Operator"},
			DefaultSensitivityLevel: 1,
			SourceFeedHints:         []string{"Confluence or Notion space", "Git repository docs", "Incident tracker exports"},
			JobTemplateIDs:          []string{"weekly_digest"},
			GovernanceNotes:         "Pair decision entities with links to SOPs and runbooks; schedule digest for leadership.",
		},
	}
}

// GetKit returns a built-in kit by id.
func GetKit(id string) (DomainSetupKit, error) {
	for _, k := range BuiltinKits() {
		if k.ID == id {
			return k, nil
		}
	}
	return DomainSetupKit{}, fmt.Errorf("onboarding: unknown kit %q", id)
}
