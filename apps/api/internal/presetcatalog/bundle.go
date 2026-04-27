package presetcatalog

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/knowledgelayer/api/internal/knowledge_jobs"
	"github.com/knowledgelayer/api/internal/role_builder"
	"github.com/knowledgelayer/api/internal/scenario_builder"
)

// Bundle wires preset catalog repositories and services.
type Bundle struct {
	Catalog          *CatalogService
	Relationships    *RelationshipService
	Instantiation    *InstantiationService
	CatalogRepo      *CatalogRepository
	RelationshipRepo *RelationshipRepository
}

// NewBundle constructs the preset catalog subsystem.
func NewBundle(pool *pgxpool.Pool, rb *role_builder.Services, sb *scenario_builder.Services, jobs *knowledge_jobs.JobService) *Bundle {
	catRepo := NewCatalogRepository(pool)
	relRepo := NewRelationshipRepository(pool)
	logRepo := NewLogRepository(pool)
	return &Bundle{
		Catalog:          NewCatalogService(catRepo, rb),
		Relationships:    NewRelationshipService(relRepo),
		Instantiation:    NewInstantiationService(catRepo, logRepo, rb, sb, jobs),
		CatalogRepo:      catRepo,
		RelationshipRepo: relRepo,
	}
}
