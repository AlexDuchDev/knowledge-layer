package httpserver

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/knowledgelayer/api/internal/app"
	"github.com/knowledgelayer/api/internal/httpcontext"
	"github.com/knowledgelayer/api/internal/scenario_builder"
)

func mountScenarioBuilderRoutes(f *fiber.App, d *app.Deps) {
	sb := d.ScenarioBuilder
	if sb == nil {
		return
	}

	f.Get("/scenarios/presets", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if err := requireCanManageIdentity(c, d, principal); err != nil {
			return err
		}
		list, err := sb.Presets.List(c.Context())
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(list)
	})

	f.Post("/scenarios/from-preset", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if err := requireCanManageIdentity(c, d, principal); err != nil {
			return err
		}
		var body scenario_builder.FromPresetInput
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		id, err := sb.Presets.CreateFromPreset(c.Context(), body)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("scenario.created_from_preset", "user", &principal, "scenario", &id, nil))
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
	})

	f.Get("/scenarios", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if err := requireCanManageIdentity(c, d, principal); err != nil {
			return err
		}
		list, err := sb.Definitions.List(c.Context())
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(list)
	})

	f.Post("/scenarios", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if err := requireCanManageIdentity(c, d, principal); err != nil {
			return err
		}
		var body scenarioWriteBody
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		in := body.toScenarioWriteInput()
		id, err := sb.Definitions.Create(c.Context(), in)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("scenario.created", "user", &principal, "scenario", &id, nil))
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
	})

	f.Get("/scenarios/:id/preview", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if err := requireCanManageIdentity(c, d, principal); err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		pv, err := sb.Preview.Build(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "scenario not found")
		}
		return c.JSON(pv)
	})

	f.Get("/scenarios/:id", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if err := requireCanManageIdentity(c, d, principal); err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		full, err := sb.Definitions.Get(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "scenario not found")
		}
		return c.JSON(full)
	})

	f.Patch("/scenarios/:id", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if err := requireCanManageIdentity(c, d, principal); err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		var body patchScenarioBody
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		var pol *scenario_builder.OutputPolicyWrite
		if body.OutputPolicy != nil {
			pol = body.OutputPolicy.toOutputPolicyWrite()
		}
		if err := sb.Definitions.Patch(c.Context(), id,
			body.Name, body.Code, body.Description, body.ScenarioType, body.Active,
			body.TargetRoleScopeJSON, body.InputScopeJSON, body.TriggerConfigJSON,
			body.TriggerType, body.ProcessingMode, body.OutputMode, body.UISurface,
			body.ConfigJSON, body.PreviewConfig, body.Notes,
			body.OwnerUserID, body.OwnerTeamID, pol,
		); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("scenario.updated", "user", &principal, "scenario", &id, nil))
		full, err := sb.Definitions.Get(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(full)
	})

	f.Delete("/scenarios/:id", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if err := requireCanManageIdentity(c, d, principal); err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		if err := sb.Definitions.Delete(c.Context(), id); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("scenario.deleted", "user", &principal, "scenario", &id, nil))
		return c.SendStatus(fiber.StatusNoContent)
	})

	f.Post("/scenarios/:id/role-bindings", func(c *fiber.Ctx) error {
		return postScenarioBindings(c, d, "scenario.role_bindings_updated", func(ctx *fiber.Ctx, id uuid.UUID, principal uuid.UUID) error {
			var body struct {
				Bindings []scenario_builder.RoleBindingWrite `json:"bindings"`
			}
			if err := ctx.BodyParser(&body); err != nil {
				return fiber.NewError(fiber.StatusBadRequest, "invalid json")
			}
			if err := d.ScenarioBuilder.Bindings.ReplaceRoleBindings(ctx.Context(), id, body.Bindings); err != nil {
				return fiber.NewError(fiber.StatusBadRequest, err.Error())
			}
			return nil
		})
	})

	f.Post("/scenarios/:id/source-bindings", func(c *fiber.Ctx) error {
		return postScenarioBindings(c, d, "scenario.source_bindings_updated", func(ctx *fiber.Ctx, id uuid.UUID, principal uuid.UUID) error {
			var body struct {
				Bindings []scenario_builder.SourceBindingRow `json:"bindings"`
			}
			if err := ctx.BodyParser(&body); err != nil {
				return fiber.NewError(fiber.StatusBadRequest, "invalid json")
			}
			if err := d.ScenarioBuilder.Bindings.ReplaceSourceBindings(ctx.Context(), id, body.Bindings); err != nil {
				return fiber.NewError(fiber.StatusBadRequest, err.Error())
			}
			return nil
		})
	})

	f.Post("/scenarios/:id/job-bindings", func(c *fiber.Ctx) error {
		return postScenarioBindings(c, d, "scenario.job_bindings_updated", func(ctx *fiber.Ctx, id uuid.UUID, principal uuid.UUID) error {
			var body struct {
				Bindings []scenario_builder.JobBindingRow `json:"bindings"`
			}
			if err := ctx.BodyParser(&body); err != nil {
				return fiber.NewError(fiber.StatusBadRequest, "invalid json")
			}
			if err := d.ScenarioBuilder.Bindings.ReplaceJobBindings(ctx.Context(), id, body.Bindings); err != nil {
				return fiber.NewError(fiber.StatusBadRequest, err.Error())
			}
			return nil
		})
	})

	f.Post("/scenarios/:id/ui-bindings", func(c *fiber.Ctx) error {
		return postScenarioBindings(c, d, "scenario.ui_bindings_updated", func(ctx *fiber.Ctx, id uuid.UUID, principal uuid.UUID) error {
			var body struct {
				Bindings []scenario_builder.UIBindingRow `json:"bindings"`
			}
			if err := ctx.BodyParser(&body); err != nil {
				return fiber.NewError(fiber.StatusBadRequest, "invalid json")
			}
			if err := d.ScenarioBuilder.Bindings.ReplaceUIBindings(ctx.Context(), id, body.Bindings); err != nil {
				return fiber.NewError(fiber.StatusBadRequest, err.Error())
			}
			return nil
		})
	})
}

func postScenarioBindings(c *fiber.Ctx, d *app.Deps, auditEvent string, fn func(*fiber.Ctx, uuid.UUID, uuid.UUID) error) error {
	principal, err := httpcontext.RequirePrincipal(c)
	if err != nil {
		return err
	}
	if err := requireCanManageIdentity(c, d, principal); err != nil {
		return err
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}
	if err := fn(c, id, principal); err != nil {
		return err
	}
	_ = d.AuditOps.Write(c.Context(), auditInput(auditEvent, "user", &principal, "scenario", &id, nil))
	return c.SendStatus(fiber.StatusNoContent)
}

type scenarioWriteBody struct {
	Code                string            `json:"code"`
	Name                string            `json:"name"`
	Description         *string           `json:"description"`
	ScenarioType        string            `json:"scenario_type"`
	Active              *bool             `json:"active"`
	TargetRoleScopeJSON json.RawMessage   `json:"target_role_scope_json"`
	InputScopeJSON      json.RawMessage   `json:"input_scope_json"`
	TriggerType         string            `json:"trigger_type"`
	TriggerConfigJSON   json.RawMessage   `json:"trigger_config_json"`
	ProcessingMode      string            `json:"processing_mode"`
	OutputMode          string            `json:"output_mode"`
	UISurface           string            `json:"ui_surface"`
	ConfigJSON          json.RawMessage   `json:"config_json"`
	PreviewConfig       json.RawMessage   `json:"preview_config"`
	Notes               *string           `json:"notes"`
	OwnerUserID         *uuid.UUID        `json:"owner_user_id"`
	OwnerTeamID         *uuid.UUID        `json:"owner_team_id"`
	IsPreset            bool              `json:"is_preset"`
	PresetKey           *string           `json:"preset_key"`
	OutputPolicy        *outputPolicyBody `json:"output_policy"`
}

func (b scenarioWriteBody) toScenarioWriteInput() scenario_builder.ScenarioWriteInput {
	in := scenario_builder.ScenarioWriteInput{
		Code:                b.Code,
		Name:                b.Name,
		Description:         b.Description,
		ScenarioType:        b.ScenarioType,
		Active:              b.Active,
		TargetRoleScopeJSON: b.TargetRoleScopeJSON,
		InputScopeJSON:      b.InputScopeJSON,
		TriggerType:         b.TriggerType,
		TriggerConfigJSON:   b.TriggerConfigJSON,
		ProcessingMode:      b.ProcessingMode,
		OutputMode:          b.OutputMode,
		UISurface:           b.UISurface,
		ConfigJSON:          b.ConfigJSON,
		PreviewConfig:       b.PreviewConfig,
		Notes:               b.Notes,
		OwnerUserID:         b.OwnerUserID,
		OwnerTeamID:         b.OwnerTeamID,
		IsPreset:            b.IsPreset,
		PresetKey:           b.PresetKey,
	}
	if b.OutputPolicy != nil {
		in.OutputPolicy = b.OutputPolicy.toOutputPolicyWrite()
	}
	return in
}

type outputPolicyBody struct {
	OutputDomainID     *uuid.UUID      `json:"output_domain_id"`
	OutputSensitivity  int             `json:"output_sensitivity_level"`
	ReviewRequired     bool            `json:"review_required"`
	PublicationMode    string          `json:"publication_mode"`
	CitationsRequired  bool            `json:"citations_required"`
	ProvenanceRequired bool            `json:"provenance_required"`
	ExtraJSON          json.RawMessage `json:"extra_json"`
}

func (b *outputPolicyBody) toOutputPolicyWrite() *scenario_builder.OutputPolicyWrite {
	if b == nil {
		return nil
	}
	return &scenario_builder.OutputPolicyWrite{
		OutputDomainID:     b.OutputDomainID,
		OutputSensitivity:  b.OutputSensitivity,
		ReviewRequired:     b.ReviewRequired,
		PublicationMode:    b.PublicationMode,
		CitationsRequired:  b.CitationsRequired,
		ProvenanceRequired: b.ProvenanceRequired,
		ExtraJSON:          b.ExtraJSON,
	}
}

type patchScenarioBody struct {
	Name                *string           `json:"name"`
	Code                *string           `json:"code"`
	Description         *string           `json:"description"`
	ScenarioType        *string           `json:"scenario_type"`
	Active              *bool             `json:"active"`
	TargetRoleScopeJSON *json.RawMessage  `json:"target_role_scope_json"`
	InputScopeJSON      *json.RawMessage  `json:"input_scope_json"`
	TriggerType         *string           `json:"trigger_type"`
	TriggerConfigJSON   *json.RawMessage  `json:"trigger_config_json"`
	ProcessingMode      *string           `json:"processing_mode"`
	OutputMode          *string           `json:"output_mode"`
	UISurface           *string           `json:"ui_surface"`
	ConfigJSON          *json.RawMessage  `json:"config_json"`
	PreviewConfig       *json.RawMessage  `json:"preview_config"`
	Notes               *string           `json:"notes"`
	OwnerUserID         *uuid.UUID        `json:"owner_user_id"`
	OwnerTeamID         *uuid.UUID        `json:"owner_team_id"`
	OutputPolicy        *outputPolicyBody `json:"output_policy"`
}
