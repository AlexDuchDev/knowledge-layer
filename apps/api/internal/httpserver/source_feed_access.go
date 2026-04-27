package httpserver

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/knowledgelayer/api/internal/app"
	"github.com/knowledgelayer/api/internal/identity_access"
	"github.com/knowledgelayer/api/internal/ingestion_connectors"
)

func requireManageSourceFeed(c *fiber.Ctx, d *app.Deps, principal uuid.UUID, feed *ingestion_connectors.SourceFeed) error {
	domainID := feed.DomainID
	sens := feed.SensitivityLevel
	fid := feed.ID
	dec, err := d.Access.Evaluate(c.Context(), identity_access.EvaluateInput{
		PrincipalID:      principal,
		Action:           "manage_source_feed",
		ResourceType:     "source_feed",
		ResourceID:       &fid,
		DomainID:         &domainID,
		SensitivityLevel: &sens,
	})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if !dec.Allow || !dec.SensitivityOK {
		return fiber.NewError(fiber.StatusForbidden, "access denied")
	}
	return nil
}

func requireManageSourceFeedNewInDomain(c *fiber.Ctx, d *app.Deps, principal uuid.UUID, domainID uuid.UUID, sensitivity int) error {
	dec, err := d.Access.Evaluate(c.Context(), identity_access.EvaluateInput{
		PrincipalID:      principal,
		Action:           "manage_source_feed",
		ResourceType:     "source_feed",
		DomainID:         &domainID,
		SensitivityLevel: &sensitivity,
	})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if !dec.Allow || !dec.SensitivityOK {
		return fiber.NewError(fiber.StatusForbidden, "access denied")
	}
	return nil
}
