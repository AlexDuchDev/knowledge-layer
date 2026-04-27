package embeddings

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/knowledgelayer/api/internal/identity_access"
	"github.com/knowledgelayer/api/internal/llm"
)

// EntityViewEvaluator evaluates governed entity view (typically *permissions.Resolver).
type EntityViewEvaluator interface {
	Evaluate(ctx context.Context, in identity_access.EvaluateInput) (*identity_access.AccessDecision, error)
}

// Candidate is one semantic neighbor after SQL domain scoping and permission filtering.
type Candidate struct {
	ChunkID          uuid.UUID
	EntityID         uuid.UUID
	TextContent      string
	Ordinal          int
	Distance         float64
	DomainID         uuid.UUID
	SensitivityLevel int
	EntityType       string
	ApprovalStatus   string
	TruthMode        string
	LifecycleState   string
	FreshnessStatus  string
}

// Service stores and queries vector embeddings joined to chunks and entities.
type Service struct {
	pool  *pgxpool.Pool
	perms EntityViewEvaluator
}

func NewService(pool *pgxpool.Pool, perms EntityViewEvaluator) *Service {
	return &Service{pool: pool, perms: perms}
}

// ModelName returns the active embedding model id (OpenAI env or default).
func ModelName() string {
	if m := os.Getenv("OPENAI_EMBEDDING_MODEL"); m != "" {
		return m
	}
	return llm.DefaultEmbeddingModel
}

func vectorLiteral(v []float32) string {
	var b strings.Builder
	b.WriteByte('[')
	for i, x := range v {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, "%g", x)
	}
	b.WriteByte(']')
	return b.String()
}

// Upsert stores or replaces the embedding vector for a chunk and model.
func (s *Service) Upsert(ctx context.Context, chunkID uuid.UUID, model, modelVersion string, vec []float32) error {
	if len(vec) == 0 {
		return fmt.Errorf("embeddings: empty vector")
	}
	id := uuid.New()
	vecLit := vectorLiteral(vec)
	_, err := s.pool.Exec(ctx, `
		INSERT INTO embeddings (id, chunk_id, model, embedding, model_version, created_at)
		VALUES ($1, $2, $3, $4::vector, $5, now())
		ON CONFLICT (chunk_id, model) DO UPDATE SET
			embedding = EXCLUDED.embedding,
			model_version = EXCLUDED.model_version,
			created_at = now()`,
		id, chunkID, model, vecLit, modelVersion)
	return err
}

// SemanticNear returns nearest chunks restricted to granted domains in SQL, then filters with Evaluate(view) per entity.
func (s *Service) SemanticNear(ctx context.Context, principal uuid.UUID, grantedDomains []uuid.UUID, queryEmbedding []float32, model string, limit int) ([]Candidate, error) {
	if len(grantedDomains) == 0 || limit <= 0 {
		return nil, nil
	}
	if len(queryEmbedding) == 0 {
		return nil, fmt.Errorf("embeddings: missing query embedding")
	}
	// Oversample for permission second stage (same pattern as search filterHitsByEntityView).
	fetch := limit * 8
	if fetch < 24 {
		fetch = 24
	}
	if fetch > 200 {
		fetch = 200
	}
	vecLit := vectorLiteral(queryEmbedding)
	rows, err := s.pool.Query(ctx, `
		SELECT c.id, c.entity_id, c.text_content, c.ordinal,
			e.domain_id, e.sensitivity_level, e.type, e.approval_status,
			e.truth_mode, e.lifecycle_state, e.freshness_status,
			(em.embedding <=> $1::vector)::float8 AS dist
		FROM embeddings em
		JOIN chunks c ON c.id = em.chunk_id
		JOIN entities e ON e.id = c.entity_id AND e.archived_at IS NULL
		WHERE em.model = $2 AND e.domain_id = ANY($3::uuid[])
		ORDER BY dist ASC
		LIMIT $4`, vecLit, model, grantedDomains, fetch)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var raw []Candidate
	for rows.Next() {
		var c Candidate
		if err := rows.Scan(&c.ChunkID, &c.EntityID, &c.TextContent, &c.Ordinal,
			&c.DomainID, &c.SensitivityLevel, &c.EntityType, &c.ApprovalStatus,
			&c.TruthMode, &c.LifecycleState, &c.FreshnessStatus, &c.Distance); err != nil {
			return nil, err
		}
		raw = append(raw, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return filterCandidatesByEntityView(ctx, s.perms, principal, raw, limit)
}

func filterCandidatesByEntityView(ctx context.Context, perms EntityViewEvaluator, principal uuid.UUID, cands []Candidate, limit int) ([]Candidate, error) {
	if perms == nil {
		return nil, fmt.Errorf("embeddings: permission evaluator required for semantic retrieval")
	}
	if len(cands) == 0 {
		return nil, nil
	}
	var out []Candidate
	for _, c := range cands {
		if len(out) >= limit {
			break
		}
		eid := c.EntityID
		did := c.DomainID
		sens := c.SensitivityLevel
		et := c.EntityType
		dec, err := perms.Evaluate(ctx, identity_access.EvaluateInput{
			PrincipalID:      principal,
			Action:           "view",
			ResourceType:     "entity",
			ResourceID:       &eid,
			DomainID:         &did,
			SensitivityLevel: &sens,
			EntityType:       &et,
		})
		if err != nil {
			return nil, err
		}
		if dec.Allow && dec.SensitivityOK {
			out = append(out, c)
		}
	}
	return out, nil
}
