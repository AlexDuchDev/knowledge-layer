package permissions

import (
	"github.com/google/uuid"

	"github.com/knowledgelayer/api/internal/identity_access"
)

// EntityEvaluateInput builds EvaluateInput for a canonical entity target.
func EntityEvaluateInput(principal uuid.UUID, action string, entityID, domainID uuid.UUID, sensitivity int, entityType string) identity_access.EvaluateInput {
	et := entityType
	s := sensitivity
	return identity_access.EvaluateInput{
		PrincipalID:      principal,
		Action:           action,
		ResourceType:     string(TargetEntity),
		ResourceID:       &entityID,
		DomainID:         &domainID,
		SensitivityLevel: &s,
		EntityType:       &et,
	}
}

// SourceFeedEvaluateInput builds EvaluateInput for connector/source feed administration.
func SourceFeedEvaluateInput(principal uuid.UUID, action string, feedID *uuid.UUID, domainID uuid.UUID, sensitivity int) identity_access.EvaluateInput {
	s := sensitivity
	return identity_access.EvaluateInput{
		PrincipalID:      principal,
		Action:           action,
		ResourceType:     string(TargetSourceFeed),
		ResourceID:       feedID,
		DomainID:         &domainID,
		SensitivityLevel: &s,
	}
}
