package role_builder

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/knowledgelayer/api/internal/identity_access"
)

// Services bundles Role Builder product-layer services.
type Services struct {
	Definitions *DefinitionService
	Presets     *PresetService
	Assignments *AssignmentService
	Preview     *PreviewService
}

// NewServices wires repositories and services.
func NewServices(pool *pgxpool.Pool, access *identity_access.AccessEvaluator) *Services {
	defRepo := NewDefinitionRepository(pool)
	assignRepo := NewAssignmentRepository(pool)
	defSvc := NewDefinitionService(defRepo)
	return &Services{
		Definitions: defSvc,
		Presets:     NewPresetService(defRepo, defSvc),
		Assignments: NewAssignmentService(assignRepo, defRepo, access),
		Preview:     NewPreviewService(defRepo),
	}
}
