package presetcatalog

import (
	"context"

	"github.com/google/uuid"
)

// RelationshipService resolves related catalog entries.
type RelationshipService struct {
	repo *RelationshipRepository
}

func NewRelationshipService(repo *RelationshipRepository) *RelationshipService {
	return &RelationshipService{repo: repo}
}

// ListRelated returns outgoing relationships from a catalog entry.
func (s *RelationshipService) ListRelated(ctx context.Context, fromID uuid.UUID) ([]RelatedEntry, error) {
	return s.repo.ListFrom(ctx, fromID)
}
