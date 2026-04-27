package extract

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/knowledgelayer/api/internal/ai/prompts"
	"github.com/knowledgelayer/api/internal/chunks"
	"github.com/knowledgelayer/api/internal/graphrag"
	neo4jgraph "github.com/knowledgelayer/api/internal/graphrag/graphdb/neo4j"
	"github.com/knowledgelayer/api/internal/llm"
)

type Service struct {
	Pool   *pgxpool.Pool
	Chunks *chunks.Service
	Graph  *graphrag.GraphStore
	LLM    *llm.OpenAIClient
}

type entityExtractResult struct {
	Entities []struct {
		Name                  string   `json:"name"`
		Type                  string   `json:"type"`
		Aliases               []string `json:"aliases,omitempty"`
		Confidence            float64  `json:"confidence"`
		EvidenceChunkOrdinals []int    `json:"evidence_chunk_ordinals"`
	} `json:"entities"`
	Relations []struct {
		From                  string  `json:"from"`
		To                    string  `json:"to"`
		Predicate             string  `json:"predicate"`
		Confidence            float64 `json:"confidence"`
		EvidenceChunkOrdinals []int   `json:"evidence_chunk_ordinals"`
	} `json:"relations"`
}

func (s *Service) ExtractEntityGraph(ctx context.Context, entityID uuid.UUID) error {
	if s == nil || s.Pool == nil || s.Chunks == nil || s.Graph == nil || s.Graph.Repo == nil || s.LLM == nil {
		return fmt.Errorf("graphrag extract: missing deps")
	}
	if entityID == uuid.Nil {
		return fmt.Errorf("graphrag extract: entity_id required")
	}

	var domainID uuid.UUID
	err := s.Pool.QueryRow(ctx, `SELECT domain_id FROM entities WHERE id=$1 AND archived_at IS NULL`, entityID).Scan(&domainID)
	if err != nil {
		return fmt.Errorf("graphrag extract: load entity domain: %w", err)
	}

	list, err := s.Chunks.ListByEntity(ctx, entityID)
	if err != nil {
		return fmt.Errorf("graphrag extract: list chunks: %w", err)
	}
	if len(list) == 0 {
		return nil
	}

	fingerprint := inputFingerprint(list)
	runID, runInserted, err := s.startRun(ctx, entityID, "v1", fingerprint)
	if err != nil {
		return err
	}
	if !runInserted {
		return nil
	}
	defer func() {
		_ = s.finishRun(ctx, runID, "completed", nil, nil)
	}()

	for _, ch := range list {
		if err := s.Graph.Repo.UpsertChunk(ctx, neo4jgraph.UpsertChunkInput{
			DomainID:   domainID,
			ChunkID:    ch.ID,
			EntityID:   ch.EntityID,
			TextHash:   "",
			Ordinal:    ch.Ordinal,
			TokenCount: ch.TokenCount,
		}); err != nil {
			_ = s.finishRun(ctx, runID, "failed", intPtr(0), strPtr(err.Error()))
			return err
		}
	}

	// Phase 4.1.1 follow-up: load from the central registry. The legacy
	// internal/graphrag/prompts/ package can be removed once no other
	// callers reference it.
	p, err := prompts.Get("graphrag_entity_extract.v1")
	if err != nil {
		_ = s.finishRun(ctx, runID, "failed", intPtr(0), strPtr(err.Error()))
		return err
	}
	user := renderChunks(list)
	schema := extractionResponseFormat()

	out, meta, err := s.LLM.ChatScenarioWithMetaAndResponseFormat(ctx, "graphrag_entity_extract", p.SystemPrompt, p.Render(map[string]string{"chunks": user}), schema)
	if err != nil {
		_ = s.finishRun(ctx, runID, "failed", intPtr(0), strPtr(err.Error()))
		return fmt.Errorf("graphrag extract: llm: %w", err)
	}

	var parsed entityExtractResult
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		_ = s.finishRun(ctx, runID, "failed", intPtr(0), strPtr("invalid json output"))
		return fmt.Errorf("graphrag extract: parse json: %w", err)
	}

	extractorVer := "v1"
	nameToID := map[string]uuid.UUID{}
	for _, e := range parsed.Entities {
		canon := canonicalName(e.Name)
		if canon == "" {
			continue
		}
		if _, ok := nameToID[canon]; ok {
			continue
		}
		id := stableGraphEntityID(domainID, canon)
		nameToID[canon] = id

		chunkIDs := evidenceOrdinalsToChunkIDs(list, e.EvidenceChunkOrdinals)
		if err := s.Graph.Repo.UpsertEntity(ctx, neo4jgraph.UpsertEntityInput{
			DomainID:       domainID,
			EntityID:       id,
			CanonicalName:  canon,
			EntityType:     e.Type,
			Aliases:        e.Aliases,
			Confidence:     e.Confidence,
			ExtractorVer:   extractorVer,
			SourceChunkIDs: chunkIDs,
		}); err != nil {
			_ = s.finishRun(ctx, runID, "failed", intPtr(0), strPtr(err.Error()))
			return err
		}
	}

	for _, r := range parsed.Relations {
		from := canonicalName(r.From)
		to := canonicalName(r.To)
		pred := strings.TrimSpace(r.Predicate)
		if from == "" || to == "" || pred == "" {
			continue
		}
		fromID, ok1 := nameToID[from]
		toID, ok2 := nameToID[to]
		if !ok1 || !ok2 {
			continue
		}
		if err := s.Graph.Repo.UpsertRelated(ctx, neo4jgraph.UpsertRelatedInput{
			DomainID:     domainID,
			FromEntityID: fromID,
			ToEntityID:   toID,
			Predicate:    pred,
			Confidence:   r.Confidence,
			ExtractorVer: extractorVer,
		}); err != nil {
			_ = s.finishRun(ctx, runID, "failed", intPtr(0), strPtr(err.Error()))
			return err
		}
	}

	_ = meta // reserved for audit hooks later
	return nil
}

func renderChunks(list []chunks.Chunk) string {
	var sb strings.Builder
	for _, ch := range list {
		sb.WriteString(fmt.Sprintf("Chunk %d:\n%s\n\n", ch.Ordinal, strings.TrimSpace(ch.TextContent)))
	}
	return sb.String()
}

func canonicalName(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "\"")
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	return s
}

func inputFingerprint(list []chunks.Chunk) string {
	h := sha256.New()
	for _, ch := range list {
		_, _ = h.Write([]byte(fmt.Sprintf("%d:", ch.Ordinal)))
		_, _ = h.Write([]byte(ch.TextContent))
		_, _ = h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func stableGraphEntityID(domainID uuid.UUID, canonicalName string) uuid.UUID {
	ns := uuid.NewSHA1(uuid.NameSpaceOID, []byte(domainID.String()))
	return uuid.NewSHA1(ns, []byte(strings.ToLower(strings.TrimSpace(canonicalName))))
}

func evidenceOrdinalsToChunkIDs(list []chunks.Chunk, ordinals []int) []uuid.UUID {
	set := map[int]bool{}
	for _, o := range ordinals {
		set[o] = true
	}
	var out []uuid.UUID
	for _, ch := range list {
		if set[ch.Ordinal] {
			out = append(out, ch.ID)
		}
	}
	return out
}

func extractionResponseFormat() any {
	return map[string]any{
		"type": "json_schema",
		"json_schema": map[string]any{
			"name":   "graph_extraction",
			"strict": true,
			"schema": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"properties": map[string]any{
					"entities": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type":                 "object",
							"additionalProperties": false,
							"properties": map[string]any{
								"name":       map[string]any{"type": "string"},
								"type":       map[string]any{"type": "string"},
								"aliases":    map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
								"confidence": map[string]any{"type": "number"},
								"evidence_chunk_ordinals": map[string]any{
									"type":  "array",
									"items": map[string]any{"type": "integer"},
								},
							},
							"required": []string{"name", "type", "confidence", "evidence_chunk_ordinals"},
						},
					},
					"relations": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type":                 "object",
							"additionalProperties": false,
							"properties": map[string]any{
								"from":       map[string]any{"type": "string"},
								"to":         map[string]any{"type": "string"},
								"predicate":  map[string]any{"type": "string"},
								"confidence": map[string]any{"type": "number"},
								"evidence_chunk_ordinals": map[string]any{
									"type":  "array",
									"items": map[string]any{"type": "integer"},
								},
							},
							"required": []string{"from", "to", "predicate", "confidence", "evidence_chunk_ordinals"},
						},
					},
				},
				"required": []string{"entities", "relations"},
			},
		},
	}
}

func (s *Service) startRun(ctx context.Context, entityID uuid.UUID, extractorVersion, fingerprint string) (uuid.UUID, bool, error) {
	var id uuid.UUID
	err := s.Pool.QueryRow(ctx, `
		INSERT INTO graphrag_extraction_runs (entity_id, extractor_version, input_fingerprint, status, started_at)
		VALUES ($1,$2,$3,'running',now())
		ON CONFLICT (entity_id, extractor_version, input_fingerprint) DO NOTHING
		RETURNING id`,
		entityID, extractorVersion, fingerprint,
	).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, false, nil
	}
	if err != nil {
		return uuid.Nil, false, fmt.Errorf("graphrag extract: start run: %w", err)
	}
	return id, true, nil
}

func (s *Service) finishRun(ctx context.Context, runID uuid.UUID, status string, tokensOut *int, errText *string) error {
	if runID == uuid.Nil {
		return nil
	}
	_, err := s.Pool.Exec(ctx, `
		UPDATE graphrag_extraction_runs
		SET status=$2, finished_at=now(), tokens_out=COALESCE($3, tokens_out), error=$4
		WHERE id=$1`,
		runID, status, tokensOut, errText,
	)
	return err
}

func intPtr(v int) *int       { return &v }
func strPtr(s string) *string { return &s }
