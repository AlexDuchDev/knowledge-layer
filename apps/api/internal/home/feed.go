package home

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/knowledgelayer/api/internal/contenthub"
	"github.com/knowledgelayer/api/internal/identity_access"
	"github.com/knowledgelayer/api/internal/knowledge_core"
	"github.com/knowledgelayer/api/internal/recommendations"
	"github.com/knowledgelayer/api/internal/review"
	"github.com/knowledgelayer/api/internal/surfacing"
)

// Entity type strings aligned with web presets / docs.
const (
	TypeDecision       = "decision"
	TypePolicy         = "policy"
	TypeProcessSOP     = "process_sop"
	TypeMeetingSummary = "meeting_summary"
	TypeInsight        = "insight"
	TypeDigestEntity   = "digest"
)

// Feed bundles permission-safe home sections.
type Feed struct {
	CanPublish          bool                      `json:"can_publish"`
	HasReviewerSurface  bool                      `json:"has_reviewer_surface"`
	ImportantDecisions  []knowledge_core.Entity   `json:"important_decisions"`
	RecentApproved      []knowledge_core.Entity   `json:"recent_approved_content"`
	FeaturedCollections *FeaturedCollectionsPlace `json:"featured_collections"`
	RecentDigests       []knowledge_core.Entity   `json:"recent_digests"`
	PendingReviews      []review.OpenTaskPreview  `json:"pending_reviews,omitempty"`
	// RecommendedReads are explainable, permission-checked picks (freshness-ranked published content).
	RecommendedReads []recommendations.Item `json:"recommended_reads,omitempty"`
	// FromFollowedScopes surfaces published content from domains, hubs, and topic slices the user follows (surfacing only).
	FromFollowedScopes []knowledge_core.Entity `json:"from_followed_scopes,omitempty"`
	// RecentInYourWork is recently updated entities in granted domains (lightweight activity signal).
	RecentInYourWork []knowledge_core.Entity `json:"recent_in_your_work,omitempty"`
}

// FeaturedCollectionsPlace is non-nil so clients can render a section; items fill when content hubs exist.
type FeaturedCollectionsPlace struct {
	Message string `json:"message"`
	HubsURL string `json:"hubs_url"`
}

type Builder struct {
	Access     *identity_access.AccessEvaluator
	Entities   *knowledge_core.EntityRepo
	Review     *review.Service
	Follows    *surfacing.FollowRepo
	ContentHub *contenthub.Service
}

func (b *Builder) Build(ctx context.Context, principal uuid.UUID, domainIDs []uuid.UUID) (*Feed, error) {
	if len(domainIDs) == 0 {
		return &Feed{
			ImportantDecisions: []knowledge_core.Entity{},
			RecentApproved:     []knowledge_core.Entity{},
			RecentDigests:      []knowledge_core.Entity{},
			RecommendedReads:   nil,
			FromFollowedScopes: nil,
			RecentInYourWork:   nil,
			FeaturedCollections: &FeaturedCollectionsPlace{
				Message: "Topic hubs will list curated collections here.",
				HubsURL: "/hubs",
			},
		}, nil
	}

	canPublish := false
	for _, dom := range domainIDs {
		did := dom
		dec, err := b.Access.Evaluate(ctx, identity_access.EvaluateInput{
			PrincipalID:  principal,
			Action:       "publish",
			ResourceType: "domain",
			DomainID:     &did,
		})
		if err != nil {
			return nil, err
		}
		if dec.Allow && dec.SensitivityOK {
			canPublish = true
			break
		}
	}

	decisions, _ := b.Entities.ListInDomains(ctx, map[string]string{"type": TypeDecision}, domainIDs, 5, 0)
	approved, _ := b.Entities.ListInDomains(ctx, map[string]string{
		"lifecycle_state": "published",
		"approval_status": "approved",
	}, domainIDs, 5, 0)

	// Digest entities: explicit type or weekly digest insights (legacy job output).
	digests, _ := b.Entities.ListInDomains(ctx, map[string]string{"type": TypeDigestEntity}, domainIDs, 5, 0)
	if len(digests) < 5 {
		insights, _ := b.Entities.ListInDomains(ctx, map[string]string{"type": "Insight"}, domainIDs, 8, 0)
		for _, e := range insights {
			if len(digests) >= 5 {
				break
			}
			if digestLike(&e) {
				digests = append(digests, e)
			}
		}
	}

	pending, _ := b.Review.ListOpenInDomains(ctx, domainIDs, principal, canPublish, 8)

	recs, _ := recommendations.HomeFreshPicks(ctx, b.Entities, domainIDs, 6, func(viewCtx context.Context, e *knowledge_core.Entity) bool {
		return b.entityViewOK(viewCtx, principal, e)
	})

	followed, _ := b.followedScopeEntities(ctx, principal, 12)
	recentWork, _ := b.Entities.ListInDomains(ctx, map[string]string{}, domainIDs, 8, 0)
	var recentFiltered []knowledge_core.Entity
	for i := range recentWork {
		e := &recentWork[i]
		if b.entityViewOK(ctx, principal, e) {
			recentFiltered = append(recentFiltered, *e)
		}
		if len(recentFiltered) >= 6 {
			break
		}
	}

	return &Feed{
		CanPublish:         canPublish,
		HasReviewerSurface: canPublish,
		ImportantDecisions: decisions,
		RecentApproved:     approved,
		FeaturedCollections: &FeaturedCollectionsPlace{
			Message: "Browse curated topic hubs.",
			HubsURL: "/hubs",
		},
		RecentDigests:      digests,
		PendingReviews:     pending,
		RecommendedReads:   recs,
		FromFollowedScopes: followed,
		RecentInYourWork:   recentFiltered,
	}, nil
}

func (b *Builder) entityViewOK(viewCtx context.Context, principal uuid.UUID, e *knowledge_core.Entity) bool {
	domainID := e.DomainID
	sens := e.SensitivityLevel
	rid := e.ID
	dec, err := b.Access.Evaluate(viewCtx, identity_access.EvaluateInput{
		PrincipalID:      principal,
		Action:           "view",
		ResourceType:     "entity",
		ResourceID:       &rid,
		DomainID:         &domainID,
		SensitivityLevel: &sens,
	})
	return err == nil && dec.Allow && dec.SensitivityOK
}

func (b *Builder) followedScopeEntities(ctx context.Context, principal uuid.UUID, cap int) ([]knowledge_core.Entity, error) {
	if cap < 1 {
		cap = 8
	}
	if b.Follows == nil || b.ContentHub == nil {
		return nil, nil
	}
	followRows, err := b.Follows.ListByUser(ctx, principal)
	if err != nil {
		return nil, err
	}
	seen := make(map[uuid.UUID]struct{})
	var out []knowledge_core.Entity
	for _, f := range followRows {
		if len(out) >= cap {
			break
		}
		switch f.ScopeType {
		case "domain":
			list, _ := b.Entities.ListInDomains(ctx, map[string]string{
				"lifecycle_state": "published",
				"approval_status": "approved",
			}, []uuid.UUID{f.RefID}, 5, 0)
			for i := range list {
				e := &list[i]
				if _, ok := seen[e.ID]; ok {
					continue
				}
				if !b.entityViewOK(ctx, principal, e) {
					continue
				}
				seen[e.ID] = struct{}{}
				out = append(out, *e)
				if len(out) >= cap {
					return out, nil
				}
			}
		case "content_hub":
			items, _ := b.ContentHub.ListItems(ctx, f.RefID)
			for _, it := range items {
				ent, gerr := b.Entities.Get(ctx, it.EntityID)
				if gerr != nil {
					continue
				}
				if _, ok := seen[ent.ID]; ok {
					continue
				}
				if !b.entityViewOK(ctx, principal, ent) {
					continue
				}
				seen[ent.ID] = struct{}{}
				out = append(out, *ent)
				if len(out) >= cap {
					return out, nil
				}
			}
		case "knowledge_topic":
			if f.EntityType == "" {
				continue
			}
			list, _ := b.Entities.ListInDomains(ctx, map[string]string{
				"type":            f.EntityType,
				"lifecycle_state": "published",
				"approval_status": "approved",
			}, []uuid.UUID{f.RefID}, 5, 0)
			for i := range list {
				e := &list[i]
				if _, ok := seen[e.ID]; ok {
					continue
				}
				if !b.entityViewOK(ctx, principal, e) {
					continue
				}
				seen[e.ID] = struct{}{}
				out = append(out, *e)
				if len(out) >= cap {
					return out, nil
				}
			}
		case "digest_stream":
			list, _ := b.Entities.ListInDomains(ctx, map[string]string{"type": TypeDigestEntity}, []uuid.UUID{f.RefID}, 6, 0)
			if len(list) < 4 {
				insights, _ := b.Entities.ListInDomains(ctx, map[string]string{"type": TypeInsight}, []uuid.UUID{f.RefID}, 8, 0)
				for _, e := range insights {
					if digestLike(&e) {
						list = append(list, e)
					}
				}
			}
			for i := range list {
				e := &list[i]
				if _, ok := seen[e.ID]; ok {
					continue
				}
				if !b.entityViewOK(ctx, principal, e) {
					continue
				}
				seen[e.ID] = struct{}{}
				out = append(out, *e)
				if len(out) >= cap {
					return out, nil
				}
			}
		}
	}
	return out, nil
}

func digestLike(e *knowledge_core.Entity) bool {
	if e.Title != "" && strings.HasPrefix(strings.TrimSpace(e.Title), "Weekly digest") {
		return true
	}
	if e.Body != nil && strings.Contains(*e.Body, "Weekly digest") {
		return true
	}
	return false
}
