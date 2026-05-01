package httpserver

import (
	"errors"
	"io"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/knowledgelayer/api/internal/app"
	"github.com/knowledgelayer/api/internal/httpcontext"
	"github.com/knowledgelayer/api/internal/ingestion_connectors"
)

// mountManualUploadRoutes wires the /api/manual/* surface that powers the
// "Collections" UI. A manual collection is a thin wrapper around source_feed:
//
//   POST   /api/manual/collections              create a collection
//   GET    /api/manual/collections              list user-visible collections
//   GET    /api/manual/collections/:id          collection detail + counts
//   GET    /api/manual/collections/:id/artifacts  recent uploads
//   DELETE /api/manual/collections/:id          archive (soft delete)
//   POST   /api/manual/collections/:id/text     paste raw text
//   POST   /api/manual/collections/:id/file     multipart file upload
//   POST   /api/manual/collections/:id/url      fetch a single web page
//   POST   /api/manual/collections/:id/youtube  fetch transcript by URL/ID
//
// All upload routes go through the existing ingestion service, which writes
// raw_artifacts + normalized_records and triggers chunks/embeddings via the
// PersistNormalizedRecord hook. Permission checks reuse the source_feed
// "manage" gate — uploading content into a collection is a write into the
// underlying feed.
func mountManualUploadRoutes(f *fiber.App, d *app.Deps) {
	api := f.Group("/api/manual")

	api.Post("/collections", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		var body struct {
			Label            string    `json:"label"`
			Description      string    `json:"description"`
			DomainID         uuid.UUID `json:"domain_id"`
			SensitivityLevel int       `json:"sensitivity_level"`
		}
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		if strings.TrimSpace(body.Label) == "" {
			return fiber.NewError(fiber.StatusBadRequest, "label is required")
		}
		if body.DomainID == uuid.Nil {
			return fiber.NewError(fiber.StatusBadRequest, "domain_id is required")
		}
		if err := requireManageSourceFeedNewInDomain(c, d, principal, body.DomainID, body.SensitivityLevel); err != nil {
			return err
		}
		feed, err := d.Ingestion.CreateManualCollection(c.Context(), ingestion_connectors.CreateManualCollectionInput{
			Label:            body.Label,
			Description:      body.Description,
			DomainID:         body.DomainID,
			SensitivityLevel: body.SensitivityLevel,
			OwnerID:          principal,
		})
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("manual_collection.created", "user", &principal, "source_feed", &feed.ID, nil))
		view, err := d.Ingestion.GetManualCollection(c.Context(), feed.ID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.Status(fiber.StatusCreated).JSON(view)
	})

	api.Get("/collections", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		domains, err := d.Access.DomainIDsWithGrant(c.Context(), principal)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		limit := clampLimit(c.Query("limit"), 100, 500)
		list, err := d.Ingestion.ListManualCollections(c.Context(), domains, limit)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(list)
	})

	api.Get("/collections/:id", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		feedID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		feed, err := d.Ingestion.GetSourceFeed(c.Context(), feedID)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		if err := requireManageSourceFeed(c, d, principal, feed); err != nil {
			return err
		}
		view, err := d.Ingestion.GetManualCollection(c.Context(), feedID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(view)
	})

	api.Get("/collections/:id/artifacts", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		feedID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		feed, err := d.Ingestion.GetSourceFeed(c.Context(), feedID)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		if err := requireManageSourceFeed(c, d, principal, feed); err != nil {
			return err
		}
		limit := clampLimit(c.Query("limit"), 100, 500)
		list, err := d.Ingestion.ListManualArtifacts(c.Context(), feedID, limit)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(list)
	})

	api.Patch("/collections/:id", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		feedID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		feed, err := d.Ingestion.GetSourceFeed(c.Context(), feedID)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		if err := requireManageSourceFeed(c, d, principal, feed); err != nil {
			return err
		}
		var body struct {
			Label       *string `json:"label"`
			Description *string `json:"description"`
		}
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		view, err := d.Ingestion.PatchManualCollection(c.Context(), feedID, ingestion_connectors.PatchManualCollectionInput{
			Label:       body.Label,
			Description: body.Description,
		})
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("manual_collection.updated", "user", &principal, "source_feed", &feedID, nil))
		return c.JSON(view)
	})

	api.Delete("/collections/:id", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		feedID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		feed, err := d.Ingestion.GetSourceFeed(c.Context(), feedID)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		if err := requireManageSourceFeed(c, d, principal, feed); err != nil {
			return err
		}
		if err := d.Ingestion.ArchiveSourceFeed(c.Context(), feedID); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("manual_collection.archived", "user", &principal, "source_feed", &feedID, nil))
		return c.SendStatus(fiber.StatusNoContent)
	})

	api.Post("/collections/:id/text", func(c *fiber.Ctx) error {
		principal, feed, err := requireManualUpload(c, d)
		if err != nil {
			return err
		}
		var body struct {
			Title             string `json:"title"`
			Body              string `json:"body"`
			SourceAttribution string `json:"source_attribution"`
		}
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		if strings.TrimSpace(body.Body) == "" {
			return fiber.NewError(fiber.StatusBadRequest, "body is required")
		}
		raw, err := d.Ingestion.IngestManualText(c.Context(), feed.ID, ingestion_connectors.IngestManualTextInput{
			Title:             body.Title,
			Body:              body.Body,
			SourceAttribution: body.SourceAttribution,
			UploaderID:        principal,
		})
		return respondManualUpload(c, d, principal, "manual_artifact.uploaded.text", raw, err)
	})

	api.Post("/collections/:id/file", func(c *fiber.Ctx) error {
		principal, feed, err := requireManualUpload(c, d)
		if err != nil {
			return err
		}
		fh, err := c.FormFile("file")
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "file required (multipart field 'file')")
		}
		if fh.Size > ingestion_connectors.MaxManualUploadSize {
			return fiber.NewError(fiber.StatusRequestEntityTooLarge, "file exceeds max size")
		}
		src, err := fh.Open()
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		defer src.Close()
		data, err := io.ReadAll(io.LimitReader(src, ingestion_connectors.MaxManualUploadSize+1))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "read upload: "+err.Error())
		}
		if int64(len(data)) > ingestion_connectors.MaxManualUploadSize {
			return fiber.NewError(fiber.StatusRequestEntityTooLarge, "file exceeds max size")
		}
		mime := fh.Header.Get("Content-Type")
		raw, err := d.Ingestion.IngestManualFile(c.Context(), feed.ID, ingestion_connectors.IngestManualFileInput{
			Filename:   fh.Filename,
			MimeType:   mime,
			Data:       data,
			UploaderID: principal,
		})
		return respondManualUpload(c, d, principal, "manual_artifact.uploaded.file", raw, err)
	})

	api.Post("/collections/:id/url", func(c *fiber.Ctx) error {
		principal, feed, err := requireManualUpload(c, d)
		if err != nil {
			return err
		}
		var body struct {
			URL string `json:"url"`
		}
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		if strings.TrimSpace(body.URL) == "" {
			return fiber.NewError(fiber.StatusBadRequest, "url is required")
		}
		raw, err := d.Ingestion.IngestManualURL(c.Context(), feed.ID, body.URL, principal)
		return respondManualUpload(c, d, principal, "manual_artifact.uploaded.url", raw, err)
	})

	api.Delete("/collections/:id/artifacts/:artifactId", func(c *fiber.Ctx) error {
		principal, feed, err := requireManualUpload(c, d)
		if err != nil {
			return err
		}
		artifactID, err := uuid.Parse(c.Params("artifactId"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid artifact id")
		}
		if err := d.Ingestion.DeleteManualArtifact(c.Context(), feed.ID, artifactID); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("manual_artifact.deleted", "user", &principal, "raw_artifact", &artifactID, nil))
		return c.SendStatus(fiber.StatusNoContent)
	})

	api.Post("/collections/:id/artifacts/:artifactId/renormalize", func(c *fiber.Ctx) error {
		principal, feed, err := requireManualUpload(c, d)
		if err != nil {
			return err
		}
		artifactID, err := uuid.Parse(c.Params("artifactId"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid artifact id")
		}
		if err := d.Ingestion.RenormalizeManualArtifact(c.Context(), feed.ID, artifactID); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("manual_artifact.renormalized", "user", &principal, "raw_artifact", &artifactID, nil))
		return c.JSON(fiber.Map{"status": "renormalized"})
	})

	api.Post("/collections/:id/search", func(c *fiber.Ctx) error {
		_, feed, err := requireManualUpload(c, d)
		if err != nil {
			return err
		}
		var body struct {
			Q     string `json:"q"`
			Limit int    `json:"limit"`
		}
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		hits, err := d.Ingestion.SearchManualCollection(c.Context(), feed.ID, body.Q, body.Limit)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(fiber.Map{"hits": hits, "query": body.Q})
	})

	api.Post("/collections/:id/youtube", func(c *fiber.Ctx) error {
		principal, feed, err := requireManualUpload(c, d)
		if err != nil {
			return err
		}
		var body struct {
			URL string `json:"url"`
		}
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		if strings.TrimSpace(body.URL) == "" {
			return fiber.NewError(fiber.StatusBadRequest, "url is required")
		}
		raw, err := d.Ingestion.IngestManualYouTube(c.Context(), feed.ID, body.URL, principal)
		return respondManualUpload(c, d, principal, "manual_artifact.uploaded.youtube", raw, err)
	})
}

// requireManualUpload resolves the feed from :id and runs the manage-feed
// permission gate. Returned values are the principal and the loaded feed.
func requireManualUpload(c *fiber.Ctx, d *app.Deps) (uuid.UUID, *ingestion_connectors.SourceFeed, error) {
	principal, err := httpcontext.RequirePrincipal(c)
	if err != nil {
		return uuid.Nil, nil, err
	}
	feedID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return uuid.Nil, nil, fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}
	feed, err := d.Ingestion.GetSourceFeed(c.Context(), feedID)
	if err != nil {
		return uuid.Nil, nil, fiber.NewError(fiber.StatusNotFound, "not found")
	}
	if err := requireManageSourceFeed(c, d, principal, feed); err != nil {
		return uuid.Nil, nil, err
	}
	return principal, feed, nil
}

// respondManualUpload converts the (raw, err) tuple from any of the four
// Ingest* methods into a clean HTTP response. Duplicate uploads (content_hash
// collision) return 200 with deduped=true rather than an error — the upload
// was valid, just redundant.
func respondManualUpload(c *fiber.Ctx, d *app.Deps, principal uuid.UUID, eventType string, raw *ingestion_connectors.RawArtifact, err error) error {
	if errors.Is(err, ingestion_connectors.ErrManualArtifactDuplicate) {
		return c.JSON(fiber.Map{
			"raw_artifact": raw,
			"deduped":      true,
		})
	}
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	if raw != nil {
		_ = d.AuditOps.Write(c.Context(), auditInput(eventType, "user", &principal, "raw_artifact", &raw.ID, nil))
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"raw_artifact": raw,
		"deduped":      false,
	})
}
