package domain

import (
	"context"

	"github.com/google/uuid"
)

// UserReader is satisfied by infra repositories (today: identity_access.Repo).
type UserReader interface {
	Find(ctx context.Context, id uuid.UUID) error
}
