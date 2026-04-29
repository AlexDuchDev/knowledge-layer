package retrieval

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/knowledgelayer/api/internal/embeddings"
	"github.com/knowledgelayer/api/internal/search"
)

// Embedder produces query vectors (typically *llm.OpenAIClient).
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

// Service orchestrates keyword, semantic, and hybrid retrieval for governed ask.
type Service struct {
	search   *search.Service
	emb      *embeddings.Service
	embedder Embedder
}

func NewService(searchSvc *search.Service, emb *embeddings.Service, embedder Embedder) *Service {
	return &Service{search: searchSvc, emb: emb, embedder: embedder}
}

func normalizeMode(mode string) string {
	m := strings.ToLower(strings.TrimSpace(mode))
	switch m {
	case "", "keyword", "keyword_only":
		return "keyword_only"
	case "semantic", "semantic_only":
		return "semantic_only"
	case "hybrid":
		return "hybrid"
	default:
		return "keyword_only"
	}
}

func hybridWeights() (wKw, wSem, penaltyW float64) {
	wKw = 0.45
	wSem = 0.55
	penaltyW = 0.02
	if v := os.Getenv("RETRIEVAL_HYBRID_W_KEYWORD"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			wKw = f
		}
	}
	if v := os.Getenv("RETRIEVAL_HYBRID_W_SEMANTIC"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			wSem = f
		}
	}
	if v := os.Getenv("RETRIEVAL_HYBRID_PENALTY_WEIGHT"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			penaltyW = f
		}
	}
	return wKw, wSem, penaltyW
}

// RankedContextPiece is one row of fused retrieval for chunk-aware QA.
type RankedContextPiece struct {
	EntityID       uuid.UUID
	ChunkID        uuid.UUID
	Text           string
	HybridScore    float64
	TruthMode      string
	LifecycleState string
	Freshness      string
	ApprovalStatus string
}

// RetrieveContext builds ordered context pieces for AskGlobal (permission filtering is applied inside search / embeddings).
func (s *Service) RetrieveContext(ctx context.Context, principal uuid.UUID, question string, searchFilters map[string]string, mode string, maxPieces int) ([]RankedContextPiece, map[string]any, error) {
	if maxPieces <= 0 {
		maxPieces = 12
	}
	metrics := map[string]any{"retrieval_mode": normalizeMode(mode)}
	m := normalizeMode(mode)

	switch m {
	case "keyword_only":
		if s.search == nil {
			return nil, metrics, fmt.Errorf("retrieval: search unavailable")
		}
		hits, err := s.search.Search(ctx, principal, searchFilters)
		if err != nil {
			return nil, metrics, err
		}
		metrics["keyword_hits"] = len(hits)
		var pieces []RankedContextPiece
		for i, h := range hits {
			if len(pieces) >= maxPieces {
				break
			}
			pieces = append(pieces, RankedContextPiece{
				EntityID:       h.EntityID,
				Text:           "",
				HybridScore:    1.0 / float64(i+1),
				TruthMode:      h.TruthMode,
				LifecycleState: h.LifecycleState,
				Freshness:      h.FreshnessStatus,
				ApprovalStatus: h.ApprovalStatus,
			})
		}
		return pieces, metrics, nil

	case "semantic_only", "hybrid":
		if s.search == nil {
			return nil, metrics, fmt.Errorf("retrieval: search service required for domain scope")
		}
		if s.embedder == nil {
			return nil, metrics, fmt.Errorf("retrieval: embedding client unavailable (set OPENAI_API_KEY or OPENROUTER_API_KEY, or OPENAI_MOCK=1)")
		}
		if s.emb == nil {
			return nil, metrics, fmt.Errorf("retrieval: embeddings service unavailable")
		}
		vec, err := s.embedder.Embed(ctx, question)
		if err != nil {
			return nil, metrics, err
		}
		model := embeddings.ModelName()
		granted, err := s.search.GrantedDomainsForPrincipal(ctx, principal)
		if err != nil {
			return nil, metrics, err
		}
		if len(granted) == 0 {
			metrics["semantic_candidates"] = 0
			return nil, metrics, nil
		}
		semLimit := maxPieces * 3
		if semLimit < 24 {
			semLimit = 24
		}
		sem, err := s.emb.SemanticNear(ctx, principal, granted, vec, model, semLimit)
		if err != nil {
			return nil, metrics, err
		}
		metrics["semantic_candidates"] = len(sem)

		if m == "semantic_only" {
			var pieces []RankedContextPiece
			for _, c := range sem {
				if len(pieces) >= maxPieces {
					break
				}
				// v0.3.0: skip normalized_record-rooted candidates — downstream
				// Ask citation flow assumes EntityID resolves via entities.Get.
				// Embeddings + retrieval-via-SemanticNear DO surface them
				// (chunks are built and embedded by the backfill loop) so admin
				// tooling and pure-vector queries see them; v0.3.1 wires them
				// through to Ask with synthesized citations from the source
				// normalized_record (channel + timestamp + author_ref).
				if c.EntityID == uuid.Nil {
					continue
				}
				pieces = append(pieces, RankedContextPiece{
					EntityID:       c.EntityID,
					ChunkID:        c.ChunkID,
					Text:           c.TextContent,
					HybridScore:    SemanticSimilarity(c.Distance),
					TruthMode:      c.TruthMode,
					LifecycleState: c.LifecycleState,
					Freshness:      c.FreshnessStatus,
					ApprovalStatus: c.ApprovalStatus,
				})
			}
			return pieces, metrics, nil
		}

		// hybrid
		if s.search == nil {
			return nil, metrics, fmt.Errorf("retrieval: search unavailable for hybrid mode")
		}
		kwHits, err := s.search.Search(ctx, principal, searchFilters)
		if err != nil {
			return nil, metrics, err
		}
		metrics["keyword_hits"] = len(kwHits)

		kwScore := map[uuid.UUID]float64{}
		for i, h := range kwHits {
			if _, ok := kwScore[h.EntityID]; !ok {
				kwScore[h.EntityID] = 1.0 / float64(i+1)
			}
		}
		bestSem := map[uuid.UUID]float64{}
		bestCand := map[uuid.UUID]embeddings.Candidate{}
		for _, c := range sem {
			// v0.3.0: same caveat as semantic_only — drop nil-EntityID
			// (normalized_record-rooted) candidates from the hybrid fusion.
			// They're retrievable but not yet wired into Ask citations.
			if c.EntityID == uuid.Nil {
				continue
			}
			sim := SemanticSimilarity(c.Distance)
			if prev, ok := bestSem[c.EntityID]; !ok || sim > prev {
				bestSem[c.EntityID] = sim
				bestCand[c.EntityID] = c
			}
		}
		maxKW, maxSem := 0.0, 0.0
		for _, v := range kwScore {
			if v > maxKW {
				maxKW = v
			}
		}
		for _, v := range bestSem {
			if v > maxSem {
				maxSem = v
			}
		}
		if maxKW <= 0 {
			maxKW = 1
		}
		if maxSem <= 0 {
			maxSem = 1
		}

		type fusedRow struct {
			id      uuid.UUID
			score   float64
			kwNorm  float64
			semNorm float64
			pen     int
		}
		seen := map[uuid.UUID]bool{}
		var rows []fusedRow
		wKw0, wSem0, pW0 := hybridWeights()
		for eid := range kwScore {
			seen[eid] = true
			kn := kwScore[eid] / maxKW
			sn := 0.0
			if v, ok := bestSem[eid]; ok {
				sn = v / maxSem
			}
			pen := 0
			if h := hitByEntityID(kwHits, eid); h != nil {
				pen = GovernancePenaltyFromHit(*h)
			} else if c, ok := bestCand[eid]; ok {
				pen = GovernancePenalty(c.TruthMode, c.LifecycleState, c.FreshnessStatus, c.ApprovalStatus)
			}
			rows = append(rows, fusedRow{id: eid, kwNorm: kn, semNorm: sn, pen: pen, score: HybridFusionScore(kn, sn, pen, wKw0, wSem0, pW0)})
		}
		for eid := range bestSem {
			if seen[eid] {
				continue
			}
			sn := bestSem[eid] / maxSem
			c := bestCand[eid]
			pen := GovernancePenalty(c.TruthMode, c.LifecycleState, c.FreshnessStatus, c.ApprovalStatus)
			rows = append(rows, fusedRow{id: eid, kwNorm: 0, semNorm: sn, pen: pen, score: HybridFusionScore(0, sn, pen, wKw0, wSem0, pW0)})
		}

		sort.SliceStable(rows, func(i, j int) bool {
			if rows[i].score != rows[j].score {
				return rows[i].score > rows[j].score
			}
			return rows[i].id.String() < rows[j].id.String()
		})

		var pieces []RankedContextPiece
		for _, r := range rows {
			if len(pieces) >= maxPieces {
				break
			}
			var text string
			var chunk uuid.UUID
			var tm, life, fresh, appr string
			if c, ok := bestCand[r.id]; ok {
				chunk = c.ChunkID
				text = c.TextContent
				tm, life, fresh, appr = c.TruthMode, c.LifecycleState, c.FreshnessStatus, c.ApprovalStatus
			}
			if h := hitByEntityID(kwHits, r.id); h != nil {
				if tm == "" {
					tm = h.TruthMode
					life = h.LifecycleState
					fresh = h.FreshnessStatus
					appr = h.ApprovalStatus
				}
			}
			pieces = append(pieces, RankedContextPiece{
				EntityID:       r.id,
				ChunkID:        chunk,
				Text:           text,
				HybridScore:    r.score,
				TruthMode:      tm,
				LifecycleState: life,
				Freshness:      fresh,
				ApprovalStatus: appr,
			})
		}
		metrics["fused_entities"] = len(rows)
		return pieces, metrics, nil

	default:
		return nil, metrics, fmt.Errorf("retrieval: unknown mode")
	}
}

func hitByEntityID(hits []search.Hit, id uuid.UUID) *search.Hit {
	for i := range hits {
		if hits[i].EntityID == id {
			return &hits[i]
		}
	}
	return nil
}
