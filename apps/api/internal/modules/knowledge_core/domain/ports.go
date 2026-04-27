package domain

import (
	"context"

	"github.com/google/uuid"
)

// EntityReader is satisfied by knowledge_core.EntityRepo (infra extraction pending).
type EntityReader interface {
	Get(ctx context.Context, id uuid.UUID) error
}
