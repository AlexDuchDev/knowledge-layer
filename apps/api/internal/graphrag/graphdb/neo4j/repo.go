package neo4j

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type Repo struct {
	Driver neo4j.DriverWithContext
}

type UpsertEntityInput struct {
	DomainID       uuid.UUID
	EntityID       uuid.UUID
	CanonicalName  string
	EntityType     string
	Aliases        []string
	Confidence     float64
	ExtractorVer   string
	SourceChunkIDs []uuid.UUID
}

func (r Repo) UpsertEntity(ctx context.Context, in UpsertEntityInput) error {
	if r.Driver == nil {
		return fmt.Errorf("neo4j repo: missing driver")
	}
	if in.DomainID == uuid.Nil {
		return fmt.Errorf("neo4j repo: domain_id is required")
	}
	if in.EntityID == uuid.Nil {
		return fmt.Errorf("neo4j repo: entity_id is required")
	}
	if strings.TrimSpace(in.CanonicalName) == "" {
		return fmt.Errorf("neo4j repo: canonical_name is required")
	}

	session := r.Driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		// Upsert entity
		_, err := tx.Run(ctx, `
MERGE (e:Entity {entity_id: $entity_id})
SET e.domain_id = $domain_id,
    e.canonical_name = $canonical_name,
    e.entity_type = $entity_type,
    e.aliases = $aliases,
    e.confidence = $confidence,
    e.extractor_version = $extractor_version,
    e.updated_at = datetime()
`, map[string]any{
			"domain_id":         in.DomainID.String(),
			"entity_id":         in.EntityID.String(),
			"canonical_name":    in.CanonicalName,
			"entity_type":       in.EntityType,
			"aliases":           in.Aliases,
			"confidence":        in.Confidence,
			"extractor_version": in.ExtractorVer,
		})
		if err != nil {
			return nil, fmt.Errorf("neo4j upsert entity: %w", err)
		}

		// Optional: connect evidence chunks (Chunk nodes are upserted elsewhere)
		for _, chID := range in.SourceChunkIDs {
			if chID == uuid.Nil {
				continue
			}
			if _, err := tx.Run(ctx, `
MATCH (e:Entity {entity_id: $entity_id})
MATCH (c:Chunk {chunk_id: $chunk_id})
WHERE e.domain_id = $domain_id AND c.domain_id = $domain_id
MERGE (c)-[m:MENTIONS]->(e)
SET m.extractor_version = $extractor_version,
    m.updated_at = datetime()
`, map[string]any{
				"domain_id":         in.DomainID.String(),
				"entity_id":         in.EntityID.String(),
				"chunk_id":          chID.String(),
				"extractor_version": in.ExtractorVer,
			}); err != nil {
				return nil, fmt.Errorf("neo4j mentions edge: %w", err)
			}
		}
		return nil, nil
	})
	return err
}

type UpsertChunkInput struct {
	DomainID   uuid.UUID
	ChunkID    uuid.UUID
	EntityID   uuid.UUID
	TextHash   string
	Ordinal    int
	TokenCount int
}

func (r Repo) UpsertChunk(ctx context.Context, in UpsertChunkInput) error {
	if r.Driver == nil {
		return fmt.Errorf("neo4j repo: missing driver")
	}
	if in.DomainID == uuid.Nil {
		return fmt.Errorf("neo4j repo: domain_id is required")
	}
	if in.ChunkID == uuid.Nil {
		return fmt.Errorf("neo4j repo: chunk_id is required")
	}
	if in.EntityID == uuid.Nil {
		return fmt.Errorf("neo4j repo: entity_id is required")
	}

	session := r.Driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		_, err := tx.Run(ctx, `
MERGE (c:Chunk {chunk_id: $chunk_id})
SET c.domain_id = $domain_id,
    c.entity_id = $entity_id,
    c.text_hash = $text_hash,
    c.ordinal = $ordinal,
    c.token_count = $token_count,
    c.updated_at = datetime()
`, map[string]any{
			"domain_id":   in.DomainID.String(),
			"chunk_id":    in.ChunkID.String(),
			"entity_id":   in.EntityID.String(),
			"text_hash":   in.TextHash,
			"ordinal":     in.Ordinal,
			"token_count": in.TokenCount,
		})
		if err != nil {
			return nil, fmt.Errorf("neo4j upsert chunk: %w", err)
		}
		return nil, nil
	})
	return err
}

type UpsertRelatedInput struct {
	DomainID     uuid.UUID
	FromEntityID uuid.UUID
	ToEntityID   uuid.UUID
	Predicate    string
	Confidence   float64
	ExtractorVer string
}

func (r Repo) UpsertRelated(ctx context.Context, in UpsertRelatedInput) error {
	if r.Driver == nil {
		return fmt.Errorf("neo4j repo: missing driver")
	}
	if in.DomainID == uuid.Nil {
		return fmt.Errorf("neo4j repo: domain_id is required")
	}
	if in.FromEntityID == uuid.Nil || in.ToEntityID == uuid.Nil {
		return fmt.Errorf("neo4j repo: from/to entity_id are required")
	}
	if strings.TrimSpace(in.Predicate) == "" {
		return fmt.Errorf("neo4j repo: predicate is required")
	}

	session := r.Driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		_, err := tx.Run(ctx, `
MATCH (a:Entity {entity_id: $from_id})
MATCH (b:Entity {entity_id: $to_id})
WHERE a.domain_id = $domain_id AND b.domain_id = $domain_id
MERGE (a)-[r:RELATED {predicate: $predicate}]->(b)
SET r.confidence = $confidence,
    r.extractor_version = $extractor_version,
    r.updated_at = datetime()
`, map[string]any{
			"domain_id":         in.DomainID.String(),
			"from_id":           in.FromEntityID.String(),
			"to_id":             in.ToEntityID.String(),
			"predicate":         in.Predicate,
			"confidence":        in.Confidence,
			"extractor_version": in.ExtractorVer,
		})
		if err != nil {
			return nil, fmt.Errorf("neo4j upsert related: %w", err)
		}
		return nil, nil
	})
	return err
}
