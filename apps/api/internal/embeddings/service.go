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
//
// As of v0.3.0 a chunk can be sourced either from an entity (legacy) or from a
// normalized_record (chat / meeting / docs surface that has not been promoted
// to an entity). For normalized-record candidates EntityID is uuid.Nil and
// NormalizedRecordID is non-nil; the entity-only fields (EntityType,
// ApprovalStatus, TruthMode, LifecycleState, FreshnessStatus) are populated
// with synthesized defaults appropriate for raw-source chunks. Callers that
// previously hard-required EntityID must now check `c.EntityID == uuid.Nil`
// and route normalized-record candidates separately.
type Candidate struct {
	ChunkID            uuid.UUID
	EntityID           uuid.UUID
	NormalizedRecordID uuid.UUID
	TextContent        string
	Ordinal            int
	Distance           float64
	DomainID           uuid.UUID
	SensitivityLevel   int
	EntityType         string
	ApprovalStatus     string
	TruthMode          string
	LifecycleState     string
	FreshnessStatus    string
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
	// v0.3.0: union the two chunk surfaces. Entity-rooted chunks pull
	// governance fields (approval, lifecycle, etc.) from `entities`;
	// normalized_record-rooted chunks synthesize sensible defaults
	// (pre_approved / mirrored_authority / published / fresh) and inherit
	// domain + sensitivity from the source feed. The order ensures the
	// distance comparator works across both sets in a single ORDER BY.
	rows, err := s.pool.Query(ctx, `
		SELECT chunk_id, entity_id, normalized_record_id, text_content, ordinal,
			domain_id, sensitivity_level, entity_type, approval_status,
			truth_mode, lifecycle_state, freshness_status, dist
		FROM (
			SELECT c.id AS chunk_id, c.entity_id, NULL::uuid AS normalized_record_id,
				c.text_content, c.ordinal,
				e.domain_id, e.sensitivity_level, e.type AS entity_type, e.approval_status,
				e.truth_mode, e.lifecycle_state, e.freshness_status,
				(em.embedding <=> $1::vector)::float8 AS dist
			FROM embeddings em
			JOIN chunks c ON c.id = em.chunk_id AND c.entity_id IS NOT NULL
			JOIN entities e ON e.id = c.entity_id AND e.archived_at IS NULL
			WHERE em.model = $2 AND e.domain_id = ANY($3::uuid[])

			UNION ALL

			SELECT c.id, NULL::uuid, c.normalized_record_id,
				c.text_content, c.ordinal,
				sf.domain_id, sf.sensitivity_level,
				'NormalizedRecord' AS entity_type,
				'pre_approved' AS approval_status,
				'mirrored_authority' AS truth_mode,
				'published' AS lifecycle_state,
				'fresh' AS freshness_status,
				(em.embedding <=> $1::vector)::float8
			FROM embeddings em
			JOIN chunks c ON c.id = em.chunk_id AND c.normalized_record_id IS NOT NULL
			JOIN normalized_records nr ON nr.id = c.normalized_record_id
			JOIN source_feeds sf ON sf.id = nr.source_feed_id
			WHERE em.model = $2 AND sf.domain_id = ANY($3::uuid[])
		) ranked
		ORDER BY dist ASC
		LIMIT $4`, vecLit, model, grantedDomains, fetch)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var raw []Candidate
	for rows.Next() {
		var c Candidate
		var entID, normID *uuid.UUID
		if err := rows.Scan(&c.ChunkID, &entID, &normID, &c.TextContent, &c.Ordinal,
			&c.DomainID, &c.SensitivityLevel, &c.EntityType, &c.ApprovalStatus,
			&c.TruthMode, &c.LifecycleState, &c.FreshnessStatus, &c.Distance); err != nil {
			return nil, err
		}
		if entID != nil {
			c.EntityID = *entID
		}
		if normID != nil {
			c.NormalizedRecordID = *normID
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
