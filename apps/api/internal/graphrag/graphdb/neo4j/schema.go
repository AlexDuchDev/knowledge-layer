package neo4j

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type SchemaEnsurer struct {
	Driver neo4j.DriverWithContext
}

func (s SchemaEnsurer) Ensure(ctx context.Context) error {
	if s.Driver == nil {
		return fmt.Errorf("neo4j schema: missing driver")
	}

	stmts := []string{
		`CREATE CONSTRAINT chunk_id_unique IF NOT EXISTS FOR (c:Chunk) REQUIRE c.chunk_id IS UNIQUE`,
		`CREATE CONSTRAINT entity_id_unique IF NOT EXISTS FOR (e:Entity) REQUIRE e.entity_id IS UNIQUE`,
		`CREATE INDEX entity_domain_name IF NOT EXISTS FOR (e:Entity) ON (e.domain_id, e.canonical_name)`,
		`CREATE INDEX chunk_domain_id IF NOT EXISTS FOR (c:Chunk) ON (c.domain_id)`,
		`CREATE INDEX chunk_entity_id IF NOT EXISTS FOR (c:Chunk) ON (c.entity_id)`,
	}

	session := s.Driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		for _, q := range stmts {
			if _, err := tx.Run(ctx, q, nil); err != nil {
				return nil, fmt.Errorf("neo4j schema: run %q: %w", q, err)
			}
		}
		return nil, nil
	})
	return err
}
