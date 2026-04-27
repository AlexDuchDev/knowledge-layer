package qa

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/knowledgelayer/api/internal/knowledge_core"
)

// AskGlobalFromEntityIDs answers using up to eight permitted entities discovered elsewhere (e.g. keyword search).
// When IncludeRelated is true, one-hop linked entities from the first root are added (same cap as entity-scoped ask).
func (s *Service) AskGlobalFromEntityIDs(ctx context.Context, principal uuid.UUID, orderedRootIDs []uuid.UUID, in AskEntityInput, canView func(*knowledge_core.Entity) error) (*AskEntityOutput, error) {
	q := strings.TrimSpace(in.Question)
	if q == "" && len(in.Images) == 0 {
		return nil, errors.New("question required (or attach images / audio_base64)")
	}
	if q == "" {
		q = "Answer using the attached image(s) and the evidence blocks below."
	}
	if len(orderedRootIDs) == 0 {
		return nil, errors.New("no search evidence: try a different query or broader filters")
	}

	var evidence []*knowledge_core.Entity
	seen := map[uuid.UUID]bool{}
	for _, id := range orderedRootIDs {
		if len(evidence) >= 8 {
			break
		}
		if seen[id] {
			continue
		}
		e, err := s.entities.Get(ctx, id)
		if err != nil {
			continue
		}
		if err := canView(e); err != nil {
			continue
		}
		seen[id] = true
		evidence = append(evidence, e)
	}
	if len(evidence) == 0 {
		return nil, errors.New("no permitted entities matched discovery results")
	}

	primaryID := evidence[0].ID
	if in.IncludeRelated {
		links, err := s.entities.ListLinks(ctx, primaryID)
		if err != nil {
			return nil, err
		}
		for _, l := range links {
			if len(evidence) >= 8 {
				break
			}
			other := l.ToEntityID
			if other == primaryID {
				other = l.FromEntityID
			}
			if seen[other] {
				continue
			}
			seen[other] = true
			e, err := s.entities.Get(ctx, other)
			if err != nil {
				continue
			}
			if err := canView(e); err != nil {
				continue
			}
			evidence = append(evidence, e)
		}
	}

	OrderEvidenceForAsk(primaryID, evidence)

	seedStr := make([]string, 0, len(orderedRootIDs))
	for _, id := range orderedRootIDs {
		if len(seedStr) >= 16 {
			break
		}
		seedStr = append(seedStr, id.String())
	}

	return s.synthesizeFromEvidence(ctx, principal, evidence, q, in, map[string]any{
		"ask_mode":             "global",
		"include_related":      in.IncludeRelated,
		"discovery_entity_ids": seedStr,
		"anchor_entity_id":     primaryID.String(),
	})
}

// AskGlobalFromContextPieces synthesizes from retrieval-ordered context (chunks and/or entity fallbacks).
func (s *Service) AskGlobalFromContextPieces(ctx context.Context, principal uuid.UUID, pieces []ContextPiece, in AskEntityInput, canView func(*knowledge_core.Entity) error) (*AskEntityOutput, error) {
	q := strings.TrimSpace(in.Question)
	if q == "" && len(in.Images) == 0 {
		return nil, errors.New("question required (or attach images / audio_base64)")
	}
	if q == "" {
		q = "Answer using the attached image(s) and the evidence blocks below."
	}
	if len(pieces) == 0 {
		return nil, errors.New("no context pieces")
	}

	seen := map[uuid.UUID]bool{}
	var evidence []*knowledge_core.Entity
	for _, p := range pieces {
		if seen[p.EntityID] {
			continue
		}
		e, err := s.entities.Get(ctx, p.EntityID)
		if err != nil {
			continue
		}
		if err := canView(e); err != nil {
			continue
		}
		seen[p.EntityID] = true
		evidence = append(evidence, e)
	}
	if len(evidence) == 0 {
		return nil, errors.New("no permitted entities matched context pieces")
	}

	primaryID := pieces[0].EntityID
	OrderEvidenceForAsk(primaryID, evidence)

	seedStr := make([]string, 0, len(pieces))
	for _, p := range pieces {
		if len(seedStr) >= 32 {
			break
		}
		seedStr = append(seedStr, p.EntityID.String())
	}

	return s.synthesizeFromContextPieces(ctx, principal, evidence, pieces, q, in, map[string]any{
		"ask_mode":             "global",
		"include_related":      in.IncludeRelated,
		"discovery_entity_ids": seedStr,
		"anchor_entity_id":     primaryID.String(),
		"context":              "retrieval_chunks",
	})
}
