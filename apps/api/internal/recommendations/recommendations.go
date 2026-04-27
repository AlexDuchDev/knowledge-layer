package recommendations

import (
	"context"
	"fmt"
	"sort"

	"github.com/google/uuid"
	"github.com/knowledgelayer/api/internal/knowledge_core"
)

// Item is a permission-checked recommendation with a human-readable reason (no black-box score exposed).
type Item struct {
	Entity knowledge_core.Entity `json:"entity"`
	Reason string                `json:"reason"`
}

type hubPeerLister interface {
	ListHubIDsContainingEntity(ctx context.Context, entityID uuid.UUID) ([]uuid.UUID, error)
	ListPeerEntityIDsInHubs(ctx context.Context, hubIDs []uuid.UUID, excludeEntity uuid.UUID, limit int) ([]uuid.UUID, error)
}

// ForEntity builds explainable recommendations: explicit links first, then curated hub peers, then same-domain published+approved with freshness bias.
func ForEntity(
	ctx context.Context,
	entities *knowledge_core.EntityRepo,
	hubs hubPeerLister,
	root *knowledge_core.Entity,
	limit int,
	canView func(context.Context, *knowledge_core.Entity) bool,
) ([]Item, error) {
	if limit < 1 {
		limit = 6
	}
	if limit > 24 {
		limit = 24
	}

	seen := make(map[uuid.UUID]struct{})
	var out []Item

	add := func(e *knowledge_core.Entity, reason string) {
		if e == nil || e.ID == root.ID {
			return
		}
		if _, ok := seen[e.ID]; ok {
			return
		}
		if !canView(ctx, e) {
			return
		}
		seen[e.ID] = struct{}{}
		out = append(out, Item{Entity: *e, Reason: reason})
	}

	links, err := entities.ListLinks(ctx, root.ID)
	if err != nil {
		return nil, fmt.Errorf("recommendations: list links: %w", err)
	}
	for _, l := range links {
		if len(out) >= limit {
			return out, nil
		}
		var other uuid.UUID
		if l.FromEntityID == root.ID {
			other = l.ToEntityID
		} else {
			other = l.FromEntityID
		}
		if other == uuid.Nil || other == root.ID {
			continue
		}
		if _, ok := seen[other]; ok {
			continue
		}
		ent, err := entities.Get(ctx, other)
		if err != nil {
			continue
		}
		add(ent, "linked:"+l.RelationType)
	}

	if hubs != nil && len(out) < limit {
		hubIDs, herr := hubs.ListHubIDsContainingEntity(ctx, root.ID)
		if herr == nil && len(hubIDs) > 0 {
			peerIDs, perr := hubs.ListPeerEntityIDsInHubs(ctx, hubIDs, root.ID, limit*2)
			if perr == nil {
				for _, pid := range peerIDs {
					if len(out) >= limit {
						break
					}
					if _, ok := seen[pid]; ok {
						continue
					}
					ent, err := entities.Get(ctx, pid)
					if err != nil {
						continue
					}
					add(ent, "curated:topic_hub_peer")
				}
			}
		}
	}

	if len(out) < limit {
		domainIDs := []uuid.UUID{root.DomainID}
		candidates, _ := entities.ListInDomains(ctx, map[string]string{
			"lifecycle_state": "published",
			"approval_status": "approved",
		}, domainIDs, 40, 0)
		ranked := rankByFreshness(candidates)
		for _, e := range ranked {
			if len(out) >= limit {
				break
			}
			if e.ID == root.ID {
				continue
			}
			reason := "same_domain:published_approved"
			switch e.FreshnessStatus {
			case "fresh":
				reason += ";fresh"
			case "review_due":
				reason += ";review_due"
			case "stale":
				reason += ";stale"
			}
			add(&e, reason)
		}
	}

	return out, nil
}

func rankByFreshness(list []knowledge_core.Entity) []knowledge_core.Entity {
	out := make([]knowledge_core.Entity, len(list))
	copy(out, list)
	order := map[string]int{"fresh": 0, "unknown": 1, "review_due": 2, "stale": 3}
	sort.SliceStable(out, func(i, j int) bool {
		ri := order[out[i].FreshnessStatus]
		if _, ok := order[out[i].FreshnessStatus]; !ok {
			ri = 1
		}
		rj := order[out[j].FreshnessStatus]
		if _, ok := order[out[j].FreshnessStatus]; !ok {
			rj = 1
		}
		if ri != rj {
			return ri < rj
		}
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
	return out
}

// ForBrowse returns published+approved entities of a type in granted domains, freshness-ranked (for knowledge type landing).
func ForBrowse(
	ctx context.Context,
	entities *knowledge_core.EntityRepo,
	domainIDs []uuid.UUID,
	entityType string,
	limit int,
	canView func(context.Context, *knowledge_core.Entity) bool,
) ([]Item, error) {
	if limit < 1 {
		limit = 6
	}
	if limit > 24 {
		limit = 24
	}
	filters := map[string]string{
		"type":            entityType,
		"lifecycle_state": "published",
		"approval_status": "approved",
	}
	list, err := entities.ListInDomains(ctx, filters, domainIDs, 80, 0)
	if err != nil {
		return nil, fmt.Errorf("recommendations: browse list: %w", err)
	}
	ranked := rankByFreshness(list)
	var out []Item
	for i := range ranked {
		e := &ranked[i]
		if !canView(ctx, e) {
			continue
		}
		reason := "browse:published_approved"
		switch e.FreshnessStatus {
		case "fresh":
			reason += ";fresh"
		case "review_due":
			reason += ";review_due"
		case "stale":
			reason += ";stale"
		}
		out = append(out, Item{Entity: *e, Reason: reason})
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

// HomeFreshPicks returns published+approved entities across granted domains, ranked by freshness (for Home).
func HomeFreshPicks(
	ctx context.Context,
	entities *knowledge_core.EntityRepo,
	domainIDs []uuid.UUID,
	limit int,
	canView func(context.Context, *knowledge_core.Entity) bool,
) ([]Item, error) {
	if limit < 1 {
		limit = 5
	}
	if limit > 12 {
		limit = 12
	}
	list, err := entities.ListInDomains(ctx, map[string]string{
		"lifecycle_state": "published",
		"approval_status": "approved",
	}, domainIDs, 80, 0)
	if err != nil {
		return nil, fmt.Errorf("recommendations: home picks: %w", err)
	}
	ranked := rankByFreshness(list)
	var out []Item
	for i := range ranked {
		e := &ranked[i]
		if !canView(ctx, e) {
			continue
		}
		out = append(out, Item{Entity: *e, Reason: "home:published_approved;freshness_ranked"})
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}
