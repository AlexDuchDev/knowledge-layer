package scenario_builder

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

// Services bundles Scenario Builder product-layer services.
type Services struct {
	Definitions *DefinitionService
	Presets     *PresetService
	Bindings    *BindingService
	Preview     *PreviewService
}

// NewServices wires repositories and services.
func NewServices(pool *pgxpool.Pool) *Services {
	defRepo := NewDefinitionRepository(pool)
	bindRepo := NewBindingsRepository(pool)
	defSvc := NewDefinitionService(defRepo, bindRepo)
	return &Services{
		Definitions: defSvc,
		Presets:     NewPresetService(defRepo, defSvc),
		Bindings:    NewBindingService(defRepo, bindRepo),
		Preview:     NewPreviewService(defRepo),
	}
}
