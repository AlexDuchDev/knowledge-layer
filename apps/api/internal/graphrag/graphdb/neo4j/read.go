package neo4j

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type EvidenceGraph struct {
	Nodes []map[string]any `json:"nodes"`
	Edges []map[string]any `json:"edges"`
}

// RelatedEntity is one neighbor surfaced by the entity-level co-mention
// expansion. Score is mention_count: how many distinct chunks mention both
// the seed entity and this neighbor (higher = stronger graph signal).
type RelatedEntity struct {
	EntityID     uuid.UUID
	MentionCount int
}

// ExpandChunksByCoMention finds additional chunks connected via co-mentions:
// seed_chunk -> Entity -> other_chunk. Returns distinct chunk IDs and a compact evidence graph.
func (r Repo) ExpandChunksByCoMention(ctx context.Context, domainID uuid.UUID, seedChunkIDs []uuid.UUID, limit int) ([]uuid.UUID, EvidenceGraph, error) {
	if r.Driver == nil {
		return nil, EvidenceGraph{}, fmt.Errorf("neo4j repo: missing driver")
	}
	if domainID == uuid.Nil {
		return nil, EvidenceGraph{}, fmt.Errorf("neo4j repo: domain_id is required")
	}
	if len(seedChunkIDs) == 0 {
		return nil, EvidenceGraph{}, nil
	}
	if limit <= 0 {
		limit = 20
	}
	seed := make([]string, 0, len(seedChunkIDs))
	for _, id := range seedChunkIDs {
		if id != uuid.Nil {
			seed = append(seed, id.String())
		}
	}

	session := r.Driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	res, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		q := `
MATCH (c:Chunk)-[:MENTIONS]->(e:Entity)<-[:MENTIONS]-(c2:Chunk)
WHERE c.domain_id = $domain_id
  AND c2.domain_id = $domain_id
  AND c.chunk_id IN $seed_chunk_ids
RETURN DISTINCT c2.chunk_id AS chunk_id,
       e.entity_id AS entity_id,
       e.canonical_name AS entity_name
LIMIT $limit`
		cur, err := tx.Run(ctx, q, map[string]any{
			"domain_id":      domainID.String(),
			"seed_chunk_ids": seed,
			"limit":          limit,
		})
		if err != nil {
			return nil, err
		}

		var chunkIDs []uuid.UUID
		var g EvidenceGraph
		nodeSeen := map[string]bool{}
		edgeSeen := map[string]bool{}
		for cur.Next(ctx) {
			rec := cur.Record()
			chStr, _ := rec.Get("chunk_id")
			entStr, _ := rec.Get("entity_id")
			entName, _ := rec.Get("entity_name")
			chunkID, _ := uuid.Parse(fmt.Sprintf("%v", chStr))
			entityID := fmt.Sprintf("%v", entStr)
			entityName := fmt.Sprintf("%v", entName)
			if chunkID != uuid.Nil {
				chunkIDs = append(chunkIDs, chunkID)
				chKey := "Chunk:" + chunkID.String()
				if !nodeSeen[chKey] {
					nodeSeen[chKey] = true
					g.Nodes = append(g.Nodes, map[string]any{"id": chKey, "kind": "chunk", "chunk_id": chunkID.String()})
				}
			}
			if entityID != "" && entityID != "<nil>" {
				eKey := "Entity:" + entityID
				if !nodeSeen[eKey] {
					nodeSeen[eKey] = true
					g.Nodes = append(g.Nodes, map[string]any{"id": eKey, "kind": "entity", "entity_id": entityID, "name": entityName})
				}
				if chunkID != uuid.Nil {
					ek := eKey + "->" + "Chunk:" + chunkID.String()
					if !edgeSeen[ek] {
						edgeSeen[ek] = true
						g.Edges = append(g.Edges, map[string]any{"from": eKey, "to": "Chunk:" + chunkID.String(), "kind": "mentions"})
					}
				}
			}
		}
		if err := cur.Err(); err != nil {
			return nil, err
		}
		return struct {
			ChunkIDs []uuid.UUID
			Graph    EvidenceGraph
		}{ChunkIDs: chunkIDs, Graph: g}, nil
	})
	if err != nil {
		return nil, EvidenceGraph{}, fmt.Errorf("neo4j expand: %w", err)
	}
	out := res.(struct {
		ChunkIDs []uuid.UUID
		Graph    EvidenceGraph
	})
	return out.ChunkIDs, out.Graph, nil
}

// RelatedEntitiesByCoMention surfaces neighbours of seedEntityID via shared
// chunk mentions: seed_entity <-[:MENTIONS]- chunk -[:MENTIONS]-> neighbour.
// Results are ordered by descending mention_count and capped by limit.
// Permission filtering is the caller's responsibility (see retrieval_intelligence.GraphExplore).
func (r Repo) RelatedEntitiesByCoMention(ctx context.Context, domainID, seedEntityID uuid.UUID, limit int) ([]RelatedEntity, error) {
	if r.Driver == nil {
		return nil, fmt.Errorf("neo4j repo: missing driver")
	}
	if domainID == uuid.Nil || seedEntityID == uuid.Nil {
		return nil, fmt.Errorf("neo4j repo: domain_id and seed_entity_id required")
	}
	if limit <= 0 {
		limit = 24
	}
	session := r.Driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	res, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		q := `
MATCH (seed:Entity {entity_id: $seed_id})<-[:MENTIONS]-(c:Chunk)-[:MENTIONS]->(neighbour:Entity)
WHERE c.domain_id = $domain_id
  AND neighbour.entity_id <> $seed_id
RETURN neighbour.entity_id AS entity_id, count(DISTINCT c) AS mention_count
ORDER BY mention_count DESC
LIMIT $limit`
		cur, err := tx.Run(ctx, q, map[string]any{
			"seed_id":   seedEntityID.String(),
			"domain_id": domainID.String(),
			"limit":     limit,
		})
		if err != nil {
			return nil, err
		}
		var out []RelatedEntity
		for cur.Next(ctx) {
			rec := cur.Record()
			entStr, _ := rec.Get("entity_id")
			mc, _ := rec.Get("mention_count")
			eid, perr := uuid.Parse(fmt.Sprintf("%v", entStr))
			if perr != nil {
				continue
			}
			count := 0
			if n, ok := mc.(int64); ok {
				count = int(n)
			}
			out = append(out, RelatedEntity{EntityID: eid, MentionCount: count})
		}
		if err := cur.Err(); err != nil {
			return nil, err
		}
		return out, nil
	})
	if err != nil {
		return nil, fmt.Errorf("neo4j related entities: %w", err)
	}
	return res.([]RelatedEntity), nil
}
