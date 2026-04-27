package neo4j

import (
	"context"
	"fmt"
	"strings"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type EntityRow struct {
	EntityID      string   `json:"entity_id"`
	CanonicalName string   `json:"canonical_name"`
	EntityType    string   `json:"entity_type"`
	Aliases       []string `json:"aliases,omitempty"`
	Confidence    float64  `json:"confidence,omitempty"`
	ExtractorVer  string   `json:"extractor_version,omitempty"`
}

func (r Repo) SearchEntities(ctx context.Context, domainID string, q string, limit int) ([]EntityRow, error) {
	if r.Driver == nil {
		return nil, fmt.Errorf("neo4j repo: missing driver")
	}
	q = strings.TrimSpace(q)
	if q == "" {
		return []EntityRow{}, nil
	}
	if limit <= 0 || limit > 50 {
		limit = 25
	}
	session := r.Driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	res, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		cur, err := tx.Run(ctx, `
MATCH (e:Entity)
WHERE e.domain_id = $domain_id AND toLower(e.canonical_name) CONTAINS toLower($q)
RETURN e.entity_id AS entity_id,
       e.canonical_name AS canonical_name,
       e.entity_type AS entity_type,
       e.aliases AS aliases,
       e.confidence AS confidence,
       e.extractor_version AS extractor_version
ORDER BY e.confidence DESC
LIMIT $limit
`, map[string]any{"domain_id": domainID, "q": q, "limit": limit})
		if err != nil {
			return nil, err
		}
		var out []EntityRow
		for cur.Next(ctx) {
			rec := cur.Record()
			row := EntityRow{
				EntityID:      fmt.Sprintf("%v", rec.Values[0]),
				CanonicalName: fmt.Sprintf("%v", rec.Values[1]),
				EntityType:    fmt.Sprintf("%v", rec.Values[2]),
				ExtractorVer:  fmt.Sprintf("%v", rec.Values[5]),
			}
			if a, ok := rec.Get("aliases"); ok {
				if arr, ok := a.([]any); ok {
					for _, v := range arr {
						row.Aliases = append(row.Aliases, fmt.Sprintf("%v", v))
					}
				}
			}
			if c, ok := rec.Get("confidence"); ok {
				if f, ok := c.(float64); ok {
					row.Confidence = f
				}
			}
			out = append(out, row)
		}
		return out, cur.Err()
	})
	if err != nil {
		return nil, err
	}
	if list, ok := res.([]EntityRow); ok {
		return list, nil
	}
	return []EntityRow{}, nil
}

type NeighborEdge struct {
	FromEntityID string  `json:"from_entity_id"`
	ToEntityID   string  `json:"to_entity_id"`
	Predicate    string  `json:"predicate"`
	Confidence   float64 `json:"confidence,omitempty"`
}

type NeighborsResult struct {
	Center    EntityRow      `json:"center"`
	Neighbors []EntityRow    `json:"neighbors"`
	Edges     []NeighborEdge `json:"edges"`
}

func (r Repo) Neighbors(ctx context.Context, domainID string, entityID string, limit int) (*NeighborsResult, error) {
	if r.Driver == nil {
		return nil, fmt.Errorf("neo4j repo: missing driver")
	}
	if strings.TrimSpace(entityID) == "" {
		return nil, fmt.Errorf("neo4j repo: missing entity_id")
	}
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	session := r.Driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	res, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		cur, err := tx.Run(ctx, `
MATCH (c:Entity {entity_id: $entity_id})
WHERE c.domain_id = $domain_id
OPTIONAL MATCH (c)-[r:RELATED]-(n:Entity)
WHERE n.domain_id = $domain_id
RETURN c, collect({n:n, r:r}) AS rows
`, map[string]any{"domain_id": domainID, "entity_id": entityID})
		if err != nil {
			return nil, err
		}
		if !cur.Next(ctx) {
			return nil, fmt.Errorf("not found")
		}
		rec := cur.Record()
		cNodeAny, _ := rec.Get("c")
		centerNode, _ := cNodeAny.(neo4j.Node)
		center := EntityRow{
			EntityID:      fmt.Sprintf("%v", centerNode.Props["entity_id"]),
			CanonicalName: fmt.Sprintf("%v", centerNode.Props["canonical_name"]),
			EntityType:    fmt.Sprintf("%v", centerNode.Props["entity_type"]),
			ExtractorVer:  fmt.Sprintf("%v", centerNode.Props["extractor_version"]),
		}
		rowsAny, _ := rec.Get("rows")
		out := &NeighborsResult{Center: center}
		if arr, ok := rowsAny.([]any); ok {
			for _, item := range arr {
				if len(out.Neighbors) >= limit {
					break
				}
				m, ok := item.(map[string]any)
				if !ok {
					continue
				}
				nAny, _ := m["n"]
				rAny, _ := m["r"]
				nNode, ok := nAny.(neo4j.Node)
				if !ok {
					continue
				}
				n := EntityRow{
					EntityID:      fmt.Sprintf("%v", nNode.Props["entity_id"]),
					CanonicalName: fmt.Sprintf("%v", nNode.Props["canonical_name"]),
					EntityType:    fmt.Sprintf("%v", nNode.Props["entity_type"]),
					ExtractorVer:  fmt.Sprintf("%v", nNode.Props["extractor_version"]),
				}
				out.Neighbors = append(out.Neighbors, n)
				if rel, ok := rAny.(neo4j.Relationship); ok {
					edge := NeighborEdge{
						FromEntityID: center.EntityID,
						ToEntityID:   n.EntityID,
						Predicate:    fmt.Sprintf("%v", rel.Props["predicate"]),
					}
					if c, ok := rel.Props["confidence"].(float64); ok {
						edge.Confidence = c
					}
					out.Edges = append(out.Edges, edge)
				}
			}
		}
		return out, nil
	})
	if err != nil {
		return nil, err
	}
	return res.(*NeighborsResult), nil
}
