package retrieval_intelligence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/knowledgelayer/api/internal/ai/privacy"
	"github.com/knowledgelayer/api/internal/answertrace"
	"github.com/knowledgelayer/api/internal/graphrag"
	"github.com/knowledgelayer/api/internal/identity_access"
	"github.com/knowledgelayer/api/internal/knowledge_core"
	"github.com/knowledgelayer/api/internal/qa"
	"github.com/knowledgelayer/api/internal/retrieval"
	"github.com/knowledgelayer/api/internal/search"
)

// ErrAccessDenied is returned by the canView callback when an entity is not viewable
// by the current principal. qa.Service treats it (and any non-nil error) as a signal
// to silently drop the entity from synthesis context.
var ErrAccessDenied = errors.New("retrieval_intelligence: access denied")

// Service is the façade for permission-aware retrieval and ask orchestration.
// It delegates keyword/hybrid search to search.Service, vector retrieval to
// retrieval.Service, and synthesis to qa.Service. Permission enforcement is
// owned here: callers must NOT pre-build canView callbacks; the façade
// constructs them from its embedded AccessEvaluator so that no retrieval path
// (including graph-expanded chunks) can bypass the 9-step access pipeline.
type Service struct {
	pool      *pgxpool.Pool
	entities  *knowledge_core.EntityRepo
	access    *identity_access.AccessEvaluator
	search    *search.Service
	qa        *qa.Service
	traces    *answertrace.Repo
	retriever *retrieval.Service
	graph     *graphrag.GraphStore
}

func NewService(pool *pgxpool.Pool, entities *knowledge_core.EntityRepo, access *identity_access.AccessEvaluator, searchSvc *search.Service, traces *answertrace.Repo, retriever *retrieval.Service, privacyGW *privacy.PrivacyGateway, graph *graphrag.GraphStore) *Service {
	return &Service{
		pool:      pool,
		entities:  entities,
		access:    access,
		search:    searchSvc,
		qa:        qa.NewService(entities, privacyGW),
		traces:    traces,
		retriever: retriever,
		graph:     graph,
	}
}

// buildCanView returns a closure that evaluates access for a single entity using
// the centralized 9-step pipeline. It is the single source of truth for "may
// this principal see this entity in retrieval results"; all Ask paths build it
// internally so a caller cannot accidentally omit the check.
func (s *Service) buildCanView(ctx context.Context, principal uuid.UUID, action string) func(*knowledge_core.Entity) error {
	return func(e *knowledge_core.Entity) error {
		if e == nil {
			return ErrAccessDenied
		}
		domainID := e.DomainID
		sens := e.SensitivityLevel
		rid := e.ID
		et := e.Type
		dec, err := s.access.Evaluate(ctx, identity_access.EvaluateInput{
			PrincipalID:      principal,
			Action:           action,
			ResourceType:     "entity",
			ResourceID:       &rid,
			DomainID:         &domainID,
			SensitivityLevel: &sens,
			EntityType:       &et,
		})
		if err != nil {
			return err
		}
		if !dec.Allow || !dec.SensitivityOK {
			return ErrAccessDenied
		}
		return nil
	}
}

// GlobalAskTrace carries persisted audit fields for POST /ask.
type GlobalAskTrace struct {
	Metrics              map[string]any
	SupportingChunksJSON json.RawMessage
	RetrievalMode        string
}

func buildSupportingChunksJSON(pieces []retrieval.RankedContextPiece) json.RawMessage {
	type row struct {
		EntityID string `json:"entity_id"`
		ChunkID  string `json:"chunk_id,omitempty"`
	}
	var rows []row
	for _, p := range pieces {
		r := row{EntityID: p.EntityID.String()}
		if p.ChunkID != uuid.Nil {
			r.ChunkID = p.ChunkID.String()
		}
		rows = append(rows, r)
	}
	b, _ := json.Marshal(rows)
	return b
}

// graphExpandContextPieces enriches the seed pieces with co-mention chunks
// from the GraphRAG store. Expanded chunks are subjected to the SAME canView
// callback as the seeds: any chunk whose owning entity the principal cannot
// view is dropped before append. This closes the access-before-retrieval leak
// that previously allowed cross-domain co-mention chunks to reach the LLM
// context (the qa.Service entity-level filter only catches entities the
// retriever surfaced; expanded chunks bypassed it because their entity was
// never on the retriever's path).
func (s *Service) graphExpandContextPieces(ctx context.Context, pieces []retrieval.RankedContextPiece, canView func(*knowledge_core.Entity) error) ([]retrieval.RankedContextPiece, map[string]any) {
	if s == nil || s.graph == nil || s.graph.Repo == nil || s.pool == nil {
		return pieces, nil
	}
	if len(pieces) == 0 {
		return pieces, nil
	}

	var domainID uuid.UUID
	if err := s.pool.QueryRow(ctx, `SELECT domain_id FROM entities WHERE id=$1`, pieces[0].EntityID).Scan(&domainID); err != nil {
		return pieces, nil
	}

	seedChunkIDs := make([]uuid.UUID, 0, len(pieces))
	seedSeen := map[uuid.UUID]bool{}
	for _, p := range pieces {
		if p.ChunkID == uuid.Nil {
			continue
		}
		if seedSeen[p.ChunkID] {
			continue
		}
		seedSeen[p.ChunkID] = true
		seedChunkIDs = append(seedChunkIDs, p.ChunkID)
	}
	if len(seedChunkIDs) == 0 {
		return pieces, nil
	}

	extraChunkIDs, g, err := s.graph.Repo.ExpandChunksByCoMention(ctx, domainID, seedChunkIDs, 12)
	if err != nil || len(extraChunkIDs) == 0 {
		return pieces, nil
	}

	need := make([]uuid.UUID, 0, len(extraChunkIDs))
	for _, id := range extraChunkIDs {
		if id == uuid.Nil || seedSeen[id] {
			continue
		}
		seedSeen[id] = true
		need = append(need, id)
	}
	if len(need) == 0 {
		return pieces, map[string]any{"nodes": g.Nodes, "edges": g.Edges}
	}

	type row struct {
		ChunkID          uuid.UUID
		EntityID         uuid.UUID
		Text             string
		EntityType       string
		EntityDomainID   uuid.UUID
		SensitivityLevel int
	}
	// JOIN entities so the same principal-aware canView callback that gates
	// seed entities can also gate graph-expanded chunks (otherwise a chunk
	// from a denied entity in the same domain could reach LLM context).
	rows, err := s.pool.Query(ctx, `
		SELECT c.id, c.entity_id, c.text_content, e.type, e.domain_id, e.sensitivity_level
		FROM chunks c
		JOIN entities e ON e.id = c.entity_id
		WHERE c.id = ANY($1)
		ORDER BY c.ordinal ASC
		LIMIT 24`, need)
	if err != nil {
		return pieces, map[string]any{"nodes": g.Nodes, "edges": g.Edges}
	}
	defer rows.Close()

	var extras []retrieval.RankedContextPiece
	denied := 0
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.ChunkID, &r.EntityID, &r.Text, &r.EntityType, &r.EntityDomainID, &r.SensitivityLevel); err != nil {
			return pieces, map[string]any{"nodes": g.Nodes, "edges": g.Edges}
		}
		if canView != nil {
			ent := &knowledge_core.Entity{
				ID:               r.EntityID,
				Type:             r.EntityType,
				DomainID:         r.EntityDomainID,
				SensitivityLevel: r.SensitivityLevel,
			}
			if err := canView(ent); err != nil {
				denied++
				continue
			}
		}
		extras = append(extras, retrieval.RankedContextPiece{
			EntityID:    r.EntityID,
			ChunkID:     r.ChunkID,
			Text:        r.Text,
			HybridScore: 0.01,
		})
	}
	if err := rows.Err(); err != nil {
		return pieces, map[string]any{"nodes": g.Nodes, "edges": g.Edges}
	}

	meta := map[string]any{"nodes": g.Nodes, "edges": g.Edges}
	if denied > 0 {
		meta["denied_expansion_chunks"] = denied
	}
	return append(pieces, extras...), meta
}

func (s *Service) SearchScoped(ctx context.Context, principal uuid.UUID, filters map[string]string) ([]search.Hit, error) {
	if s.search == nil {
		return []search.Hit{}, nil
	}
	return s.search.Search(ctx, principal, filters)
}

func (s *Service) AskEntity(ctx context.Context, principal, entityID uuid.UUID, in qa.AskEntityInput) (*qa.AskEntityOutput, error) {
	return s.qa.AskEntity(ctx, principal, entityID, in, s.buildCanView(ctx, principal, "view"))
}

// PreprocessAskMultimodal decodes optional audio and normalizes question (see qa.Service).
func (s *Service) PreprocessAskMultimodal(ctx context.Context, in *qa.AskEntityInput) error {
	return s.qa.PreprocessAskMultimodal(ctx, in)
}

// AskGlobal runs discovery then synthesis. retrievalMode: keyword_only (default), semantic_only, hybrid.
// Permission filtering of retrieved entities is performed internally via buildCanView; callers must not
// supply their own canView callback (it would not be applied to graph-expanded chunks anyway — see 1.1.1).
func (s *Service) AskGlobal(ctx context.Context, principal uuid.UUID, askIn qa.AskEntityInput, searchFilters map[string]string, retrievalMode string) (*qa.AskEntityOutput, *GlobalAskTrace, error) {
	canView := s.buildCanView(ctx, principal, "view")
	t0 := time.Now()
	mode := strings.TrimSpace(strings.ToLower(retrievalMode))
	question := strings.TrimSpace(askIn.Question)
	switch mode {
	case "semantic", "semantic_only", "hybrid":
		if s.retriever == nil {
			return nil, &GlobalAskTrace{Metrics: map[string]any{"retrieval_mode": mode}, SupportingChunksJSON: []byte("[]"), RetrievalMode: mode}, fmt.Errorf("vector retrieval is not configured")
		}
		pieces, metrics, err := s.retriever.RetrieveContext(ctx, principal, question, searchFilters, mode, 12)
		if metrics == nil {
			metrics = map[string]any{}
		}
		metrics["latency_retrieval_ms"] = time.Since(t0).Milliseconds()
		if err != nil {
			return nil, &GlobalAskTrace{Metrics: metrics, SupportingChunksJSON: []byte("[]"), RetrievalMode: mode}, err
		}
		if len(pieces) == 0 {
			return nil, &GlobalAskTrace{Metrics: metrics, SupportingChunksJSON: []byte("[]"), RetrievalMode: mode}, errors.New("no retrieval results for the current mode; try keyword_only or broaden filters")
		}

		var evidenceGraph map[string]any
		pieces, evidenceGraph = s.graphExpandContextPieces(ctx, pieces, canView)

		chunksJSON := buildSupportingChunksJSON(pieces)
		var cp []qa.ContextPiece
		for _, p := range pieces {
			cp = append(cp, qa.ContextPiece{EntityID: p.EntityID, ChunkID: p.ChunkID, Text: p.Text})
		}
		out, err := s.qa.AskGlobalFromContextPieces(ctx, principal, cp, askIn, canView)
		if err != nil {
			return nil, &GlobalAskTrace{Metrics: metrics, SupportingChunksJSON: chunksJSON, RetrievalMode: traceRetrievalMode(metrics, mode)}, err
		}
		out.EvidenceGraph = evidenceGraph
		return out, &GlobalAskTrace{Metrics: metrics, SupportingChunksJSON: chunksJSON, RetrievalMode: traceRetrievalMode(metrics, mode)}, nil

	default:
		// keyword_only (default)
		hits, err := s.SearchScoped(ctx, principal, searchFilters)
		metrics := map[string]any{
			"retrieval_mode":       "keyword_only",
			"latency_retrieval_ms": time.Since(t0).Milliseconds(),
		}
		if err != nil {
			return nil, &GlobalAskTrace{Metrics: metrics, SupportingChunksJSON: []byte("[]"), RetrievalMode: "keyword_only"}, err
		}
		metrics["keyword_hits"] = len(hits)
		const maxSeeds = 12
		seeds := make([]uuid.UUID, 0, maxSeeds)
		for _, h := range hits {
			if len(seeds) >= maxSeeds {
				break
			}
			seeds = append(seeds, h.EntityID)
		}
		out, err := s.qa.AskGlobalFromEntityIDs(ctx, principal, seeds, askIn, canView)
		if err != nil {
			return nil, &GlobalAskTrace{Metrics: metrics, SupportingChunksJSON: []byte("[]"), RetrievalMode: "keyword_only"}, err
		}
		return out, &GlobalAskTrace{Metrics: metrics, SupportingChunksJSON: []byte("[]"), RetrievalMode: "keyword_only"}, nil
	}
}

func traceRetrievalMode(metrics map[string]any, fallback string) string {
	if metrics != nil {
		if v, ok := metrics["retrieval_mode"].(string); ok && v != "" {
			return v
		}
	}
	return fallback
}

func (s *Service) PersistAnswerTrace(ctx context.Context, row answertrace.Row) error {
	if s.traces == nil {
		return nil
	}
	return s.traces.Insert(ctx, row)
}

func (s *Service) LogSearchInteraction(ctx context.Context, principal uuid.UUID, filters map[string]string, hitCount int) {
	if s.search != nil {
		s.search.LogSearchInteraction(ctx, principal, filters, hitCount)
	}
}

func OpenAIModelFromEnv() string {
	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = "gpt-4o-mini"
	}
	if os.Getenv("OPENAI_MOCK") == "1" {
		model = "mock"
	}
	return model
}

// BuildGlobalAskSearchFilters maps a global-ask JSON body to search filter keys.
func BuildGlobalAskSearchFilters(question, domainID, typ, truthMode, lifecycleState, freshnessStatus, approvalStatus string) map[string]string {
	return map[string]string{
		"q":                strings.TrimSpace(question),
		"domain_id":        strings.TrimSpace(domainID),
		"type":             strings.TrimSpace(typ),
		"truth_mode":       strings.TrimSpace(truthMode),
		"lifecycle_state":  strings.TrimSpace(lifecycleState),
		"freshness_status": strings.TrimSpace(freshnessStatus),
		"approval_status":  strings.TrimSpace(approvalStatus),
	}
}
