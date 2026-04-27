package retrieval_intelligence

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/knowledgelayer/api/internal/knowledge_core"
)

// GraphExploreNeighbour is one related entity surfaced from the seed entity
// via co-mention edges in Neo4j, after permission filtering.
type GraphExploreNeighbour struct {
	Entity       *knowledge_core.Entity `json:"entity"`
	MentionCount int                    `json:"mention_count"`
}

// GraphExploreResult is the response body for /entities/:id/graph-explore.
// Denied counts are surfaced so operators can see when permission filtering
// trimmed the graph (a non-zero number is a hint that the principal lacks
// access to neighbouring entities, not that the graph is sparse).
type GraphExploreResult struct {
	SeedEntityID uuid.UUID                `json:"seed_entity_id"`
	Neighbours   []GraphExploreNeighbour  `json:"neighbours"`
	Returned     int                      `json:"returned"`
	DeniedCount  int                      `json:"denied_count"`
	Truncated    bool                     `json:"truncated"`
}

// ErrGraphNotConfigured is returned when GraphRAG is not wired (no NEO4J_URL).
var ErrGraphNotConfigured = errors.New("retrieval_intelligence: graph store not configured")

// GraphExplore returns up to maxNodes neighbours of seedEntityID via shared
// chunk co-mention, filtered by the principal's view permission. The seed
// entity must itself be viewable; otherwise ErrAccessDenied bubbles up.
//
// "Bounded" means: capped at maxNodes (default 24, hard ceiling 100), single
// hop only, all results in the seed's domain. This deliberately avoids the
// open-ended "explore the whole graph" pattern that would be hostile to
// access control and to context windows.
func (s *Service) GraphExplore(ctx context.Context, principal, seedEntityID uuid.UUID, maxNodes int) (*GraphExploreResult, error) {
	if s == nil || s.entities == nil {
		return nil, fmt.Errorf("retrieval_intelligence: service not initialised")
	}
	if s.graph == nil || s.graph.Repo == nil {
		return nil, ErrGraphNotConfigured
	}
	if maxNodes <= 0 {
		maxNodes = 24
	}
	if maxNodes > 100 {
		maxNodes = 100
	}

	canView := s.buildCanView(ctx, principal, "view")

	seed, err := s.entities.Get(ctx, seedEntityID)
	if err != nil {
		return nil, fmt.Errorf("retrieval_intelligence: load seed entity: %w", err)
	}
	if err := canView(seed); err != nil {
		return nil, ErrAccessDenied
	}

	related, err := s.graph.Repo.RelatedEntitiesByCoMention(ctx, seed.DomainID, seedEntityID, maxNodes)
	if err != nil {
		return nil, err
	}

	result := &GraphExploreResult{SeedEntityID: seedEntityID}
	for _, r := range related {
		ent, gerr := s.entities.Get(ctx, r.EntityID)
		if gerr != nil {
			// Missing entity in postgres (eventual-consistency lag with neo4j) — skip silently.
			continue
		}
		if err := canView(ent); err != nil {
			result.DeniedCount++
			continue
		}
		result.Neighbours = append(result.Neighbours, GraphExploreNeighbour{
			Entity:       ent,
			MentionCount: r.MentionCount,
		})
	}
	result.Returned = len(result.Neighbours)
	result.Truncated = len(related) >= maxNodes
	return result, nil
}
