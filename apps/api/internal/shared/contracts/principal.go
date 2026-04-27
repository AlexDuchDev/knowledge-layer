package contracts

import "github.com/google/uuid"

// PrincipalID is the authenticated user identifier carried through transport and app layers.
type PrincipalID uuid.UUID
