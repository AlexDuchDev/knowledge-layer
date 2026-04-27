package graphrag

import (
	"context"
	"fmt"

	neo4jdriver "github.com/neo4j/neo4j-go-driver/v5/neo4j"

	neo4jgraph "github.com/knowledgelayer/api/internal/graphrag/graphdb/neo4j"
)

type GraphStore struct {
	Driver neo4jdriver.DriverWithContext
	Repo   *neo4jgraph.Repo
	Schema neo4jgraph.SchemaEnsurer
}

func NewGraphStore(driver neo4jdriver.DriverWithContext) *GraphStore {
	return &GraphStore{
		Driver: driver,
		Repo:   &neo4jgraph.Repo{Driver: driver},
		Schema: neo4jgraph.SchemaEnsurer{Driver: driver},
	}
}

func (s *GraphStore) Close(ctx context.Context) error {
	if s == nil || s.Driver == nil {
		return nil
	}
	return s.Driver.Close(ctx)
}

func (s *GraphStore) EnsureSchema(ctx context.Context) error {
	if s == nil {
		return fmt.Errorf("graphrag: missing store")
	}
	return s.Schema.Ensure(ctx)
}
