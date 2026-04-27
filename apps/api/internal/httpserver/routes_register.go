package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/knowledgelayer/api/internal/ai/privacy"
	"github.com/knowledgelayer/api/internal/ai/prompts"
	"github.com/knowledgelayer/api/internal/answertrace"
	"github.com/knowledgelayer/api/internal/app"
	"github.com/knowledgelayer/api/internal/config"
	"github.com/knowledgelayer/api/internal/extracted_meeting_tasks"
	"github.com/knowledgelayer/api/internal/governance"
	"github.com/knowledgelayer/api/internal/httpcontext"
	"github.com/knowledgelayer/api/internal/identity_access"
	"github.com/knowledgelayer/api/internal/ingestion_connectors"
	"github.com/knowledgelayer/api/internal/ingestion_connectors/adapters/mattermost"
	"github.com/knowledgelayer/api/internal/ingestion_connectors/adapters/slack"
	"github.com/knowledgelayer/api/internal/knowledge_core"
	"github.com/knowledgelayer/api/internal/knowledge_jobs"
	"github.com/knowledgelayer/api/internal/llm"
	audit_opstransport "github.com/knowledgelayer/api/internal/modules/audit_ops/transport"
	"github.com/knowledgelayer/api/internal/onboarding"
	"github.com/knowledgelayer/api/internal/qa"
	"github.com/knowledgelayer/api/internal/recommendations"
	"github.com/knowledgelayer/api/internal/retrieval_intelligence"
	"github.com/knowledgelayer/api/internal/secondbrain"
)

func mountAPIRoutes(f *fiber.App, d *app.Deps, cfg config.Config) {
	registerConnectorIntegrationRoutes(f, d)
	registerConnectorOAuthRoutes(f, d, cfg)

	f.Get("/users", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if err := requireCanManageIdentity(c, d, principal); err != nil {
			return err
		}
		list, err := d.Identity.ListUsers(c.Context())
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(list)
	})

	f.Get("/users/:id", func(c *fiber.Ctx) error {
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
		u, err := d.Identity.GetUser(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "user not found")
		}
		return c.JSON(u)
	})

	// GET /users/:id/effective-access — surfaces the 9-step access pipeline trace
	// for a target user against an optional resource (Phase 2.1.6 effective-access UI).
	// Auth: identity admin OR self (a user can always introspect their own access).
	// Required query: action, resource_type. Optional: resource_id, domain_id,
	// sensitivity_level, entity_type. The Trace field — normally internal — is
	// included in the response so operators can see exactly which gate denied or
	// allowed; never expose this endpoint to anonymous callers.
	f.Get("/users/:id/effective-access", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		targetID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid user id")
		}
		if targetID != principal {
			if err := requireCanManageIdentity(c, d, principal); err != nil {
				return err
			}
		}
		action := strings.TrimSpace(c.Query("action"))
		resourceType := strings.TrimSpace(c.Query("resource_type"))
		if action == "" || resourceType == "" {
			return fiber.NewError(fiber.StatusBadRequest, "action and resource_type query params required")
		}
		in := identity_access.EvaluateInput{
			PrincipalID:  targetID,
			Action:       action,
			ResourceType: resourceType,
		}
		if v := strings.TrimSpace(c.Query("resource_id")); v != "" {
			rid, perr := uuid.Parse(v)
			if perr != nil {
				return fiber.NewError(fiber.StatusBadRequest, "invalid resource_id")
			}
			in.ResourceID = &rid
		}
		if v := strings.TrimSpace(c.Query("domain_id")); v != "" {
			did, perr := uuid.Parse(v)
			if perr != nil {
				return fiber.NewError(fiber.StatusBadRequest, "invalid domain_id")
			}
			in.DomainID = &did
		}
		if v := strings.TrimSpace(c.Query("sensitivity_level")); v != "" {
			lvl, perr := strconv.Atoi(v)
			if perr != nil {
				return fiber.NewError(fiber.StatusBadRequest, "invalid sensitivity_level")
			}
			in.SensitivityLevel = &lvl
		}
		if v := strings.TrimSpace(c.Query("entity_type")); v != "" {
			in.EntityType = &v
		}
		dec, err := d.Access.Evaluate(c.Context(), in)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		// AccessDecision.Trace is `json:"-"` for safety; rebuild the response so
		// the trace becomes visible only on this introspection endpoint.
		return c.JSON(fiber.Map{
			"allow":                 dec.Allow,
			"reason_code":           dec.ReasonCode,
			"reasons":               dec.Reasons,
			"trace":                 dec.Trace,
			"sensitivity_ok":        dec.SensitivityOK,
			"effective_sensitivity": dec.EffectiveSensitivity,
			"resolved_domain_id":    dec.ResolvedDomainID,
			"sensitivity_result":    dec.SensitivityResult,
			"matched_rule_code":     dec.MatchedRuleCode,
			"matched_policies":      dec.MatchedPolicies,
			"matched_overrides":     dec.MatchedOverrides,
			"input": fiber.Map{
				"action":           in.Action,
				"resource_type":    in.ResourceType,
				"resource_id":      in.ResourceID,
				"domain_id":        in.DomainID,
				"sensitivity_level": in.SensitivityLevel,
				"entity_type":      in.EntityType,
			},
		})
	})

	f.Get("/domains", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		list, err := d.Identity.ListDomainsForUser(c.Context(), principal)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(list)
	})

	f.Get("/onboarding/domain-kits", func(c *fiber.Ctx) error {
		if _, err := httpcontext.RequirePrincipal(c); err != nil {
			return err
		}
		return c.JSON(onboarding.BuiltinKits())
	})

	f.Post("/domains/:id/apply-setup-kit", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		domainID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		var body struct {
			KitID string `json:"kit_id"`
		}
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		kit, err := onboarding.GetKit(strings.TrimSpace(body.KitID))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		dec, err := d.Access.Evaluate(c.Context(), identity_access.EvaluateInput{
			PrincipalID:  principal,
			Action:       "publish",
			ResourceType: "domain",
			DomainID:     &domainID,
		})
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		if !dec.Allow || !dec.SensitivityOK {
			return fiber.NewError(fiber.StatusForbidden, "publish permission required on domain")
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("domain.setup_kit_applied", "user", &principal, "domain", &domainID, ptr(kit.ID)))
		// Kit metadata is documentation-oriented; no DB entities are created here (see onboarding.DomainSetupKit).
		return c.JSON(fiber.Map{
			"recorded": true,
			"applied":  false,
			"kit":      kit,
		})
	})

	f.Get("/home/feed", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		granted, err := d.Access.DomainIDsWithGrant(c.Context(), principal)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		feed, err := d.Home.Build(c.Context(), principal, granted)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(feed)
	})

	f.Get("/me/follows", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if d.Follows == nil {
			return fiber.NewError(fiber.StatusInternalServerError, "follows not configured")
		}
		list, err := d.Follows.ListByUser(c.Context(), principal)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(list)
	})

	f.Post("/me/follows", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if d.Follows == nil {
			return fiber.NewError(fiber.StatusInternalServerError, "follows not configured")
		}
		var body struct {
			ScopeType  string    `json:"scope_type"`
			RefID      uuid.UUID `json:"ref_id"`
			EntityType string    `json:"entity_type"`
		}
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		st := strings.TrimSpace(body.ScopeType)
		switch st {
		case "domain", "content_hub", "knowledge_topic", "digest_stream":
		default:
			return fiber.NewError(fiber.StatusBadRequest, "invalid scope_type")
		}
		granted, err := d.Access.DomainIDsWithGrant(c.Context(), principal)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		domainOK := func(did uuid.UUID) bool {
			for _, g := range granted {
				if g == did {
					return true
				}
			}
			return false
		}
		switch st {
		case "domain":
			if !domainOK(body.RefID) {
				return fiber.NewError(fiber.StatusForbidden, "domain not granted")
			}
		case "content_hub":
			hub, herr := d.ContentHub.GetByID(c.Context(), body.RefID)
			if herr != nil {
				return fiber.NewError(fiber.StatusNotFound, "hub not found")
			}
			if !domainOK(hub.DomainID) {
				return fiber.NewError(fiber.StatusForbidden, "access denied")
			}
		case "knowledge_topic":
			if strings.TrimSpace(body.EntityType) == "" {
				return fiber.NewError(fiber.StatusBadRequest, "entity_type required for knowledge_topic")
			}
			if !domainOK(body.RefID) {
				return fiber.NewError(fiber.StatusForbidden, "domain not granted")
			}
		case "digest_stream":
			if !domainOK(body.RefID) {
				return fiber.NewError(fiber.StatusForbidden, "domain not granted")
			}
		}
		out, err := d.Follows.Add(c.Context(), principal, st, body.RefID, strings.TrimSpace(body.EntityType))
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("follow.created", "user", &principal, "user_scope_follow", &out.ID, nil))
		return c.JSON(out)
	})

	f.Delete("/me/follows", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if d.Follows == nil {
			return fiber.NewError(fiber.StatusInternalServerError, "follows not configured")
		}
		st := strings.TrimSpace(c.Query("scope_type"))
		refStr := strings.TrimSpace(c.Query("ref_id"))
		et := strings.TrimSpace(c.Query("entity_type"))
		if st == "" || refStr == "" {
			return fiber.NewError(fiber.StatusBadRequest, "scope_type and ref_id required")
		}
		refID, perr := uuid.Parse(refStr)
		if perr != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid ref_id")
		}
		switch st {
		case "domain", "content_hub", "knowledge_topic", "digest_stream":
		default:
			return fiber.NewError(fiber.StatusBadRequest, "invalid scope_type")
		}
		if err := d.Follows.Remove(c.Context(), principal, st, refID, et); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("follow.removed", "user", &principal, "user", &principal, nil))
		return c.SendStatus(fiber.StatusNoContent)
	})

	f.Post("/access/evaluate", func(c *fiber.Ctx) error {
		caller, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		var body identity_access.EvaluateInput
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		if body.PrincipalID == uuid.Nil {
			return fiber.NewError(fiber.StatusBadRequest, "principal_id required")
		}
		if body.PrincipalID != caller {
			if err := requireCanManageIdentity(c, d, caller); err != nil {
				return err
			}
		}
		out, err := d.Access.Evaluate(c.Context(), body)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(out)
	})

	f.Get("/entities", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		granted, err := d.Access.DomainIDsWithGrant(c.Context(), principal)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		if len(granted) == 0 {
			return c.JSON([]knowledge_core.Entity{})
		}
		filters := map[string]string{
			"type":            c.Query("type"),
			"domain_id":       c.Query("domain_id"),
			"owner_id":        c.Query("owner_id"),
			"truth_mode":      c.Query("truth_mode"),
			"lifecycle_state": c.Query("lifecycle_state"),
			"approval_status": c.Query("approval_status"),
			"sort":            c.Query("sort"),
		}
		if fd := filters["domain_id"]; fd != "" {
			want, perr := uuid.Parse(fd)
			if perr != nil {
				return c.JSON([]knowledge_core.Entity{})
			}
			ok := false
			for _, g := range granted {
				if g == want {
					ok = true
					break
				}
			}
			if !ok {
				return c.JSON([]knowledge_core.Entity{})
			}
			granted = []uuid.UUID{want}
		}
		limit := 50
		if v := c.Query("limit"); v != "" {
			if n, e := strconv.Atoi(v); e == nil && n > 0 && n <= 500 {
				limit = n
			}
		}
		offset := 0
		if v := c.Query("offset"); v != "" {
			if n, e := strconv.Atoi(v); e == nil && n >= 0 {
				offset = n
			}
		}
		list, err := d.Entities.ListInDomains(c.Context(), filters, granted, limit, offset)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(list)
	})

	f.Get("/entities/:id/versions", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		ent, err := d.Entities.Get(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		if err := requireEntityAction(c, d, principal, ent, "view"); err != nil {
			return err
		}
		list, err := d.Entities.ListEntityVersions(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(list)
	})

	f.Get("/entities/:id/compiled-truth", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		ent, err := d.Entities.Get(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		if err := requireEntityAction(c, d, principal, ent, "view"); err != nil {
			return err
		}
		ct, err := d.Entities.GetCompiledTruth(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		return c.JSON(ct)
	})

	f.Put("/entities/:id/compiled-truth", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		ent, err := d.Entities.Get(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		if err := requireEntityAction(c, d, principal, ent, "edit"); err != nil {
			return err
		}
		var body struct {
			CompiledSummary *string `json:"compiled_summary"`
			CompiledBody    *string `json:"compiled_body"`
		}
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		var basedOn *int
		if v, verr := d.Entities.ListEntityVersions(c.Context(), id); verr == nil && len(v) > 0 {
			n := v[len(v)-1].VersionNumber
			basedOn = &n
		}
		bt := "user"
		_ = d.Entities.UpsertCompiledTruth(c.Context(), knowledge_core.EntityCompiledTruth{
			EntityID:             id,
			CompiledSummary:      body.CompiledSummary,
			CompiledBody:         body.CompiledBody,
			BasedOnVersionNumber: basedOn,
			CompiledByType:       &bt,
			CompiledByID:         &principal,
		})
		_ = d.AuditOps.Write(c.Context(), auditInput("entity.compiled_truth.updated", "user", &principal, "entity", &id, nil))
		return c.SendStatus(fiber.StatusNoContent)
	})

	f.Get("/entities/:id", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		e, err := d.Entities.Get(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		if err := requireEntityAction(c, d, principal, e, "view"); err != nil {
			return err
		}
		return c.JSON(e)
	})

	f.Get("/entities/:id/chunks", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		e, err := d.Entities.Get(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		if err := requireEntityAction(c, d, principal, e, "view"); err != nil {
			return err
		}
		list, err := d.Chunks.ListByEntity(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(fiber.Map{"chunks": list})
	})

	f.Get("/entities/:id/detail", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		e, err := d.Entities.Get(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		if err := requireEntityAction(c, d, principal, e, "view"); err != nil {
			return err
		}

		var payload any = nil
		p, perr := d.Entities.GetPayload(c.Context(), id)
		if perr != nil && !errors.Is(perr, pgx.ErrNoRows) {
			return fiber.NewError(fiber.StatusInternalServerError, perr.Error())
		}
		if perr == nil {
			payload = p
		}

		prov, err := d.Entities.ListProvenance(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}

		var source *string
		var externalURL *string
		if perr == nil {
			var pj map[string]any
			if jerr := json.Unmarshal(p.PayloadJSON, &pj); jerr == nil {
				if s, ok := pj["source"].(string); ok && s != "" {
					source = &s
				}
				if s, ok := pj["web_view_link"].(string); ok && s != "" {
					if u, uerr := url.Parse(s); uerr == nil && (u.Scheme == "https" || u.Scheme == "http") && u.Host != "" {
						// Avoid implying any bypass: this is a UI convenience link only.
						externalURL = &s
					}
				}
			}
		}

		snapshotAt := e.UpdatedAt
		if perr == nil {
			snapshotAt = p.UpdatedAt
		}

		type preview struct {
			Text      string `json:"text"`
			Truncated bool   `json:"truncated"`
		}
		var bodyPreview *preview
		if e.Body != nil && *e.Body != "" {
			const max = 2000
			txt := strings.TrimSpace(*e.Body)
			if len(txt) > max {
				bodyPreview = &preview{Text: txt[:max], Truncated: true}
			} else {
				bodyPreview = &preview{Text: txt, Truncated: false}
			}
		}

		return c.JSON(fiber.Map{
			"entity":             e,
			"payload":            payload,
			"provenance":         prov,
			"snapshot_at":        snapshotAt,
			"source":             source,
			"open_in_source_url": externalURL,
			"content_preview":    bodyPreview,
			"freshness_status":   e.FreshnessStatus,
			"truth_mode":         e.TruthMode,
			"external_ref":       e.ExternalRef,
			"owner_id":           e.OwnerID,
			"domain_id":          e.DomainID,
			"sensitivity_level":  e.SensitivityLevel,
			"lifecycle_state":    e.LifecycleState,
			"canonical_status":   e.CanonicalStatus,
			"approval_status":    e.ApprovalStatus,
			"updated_at":         e.UpdatedAt,
			"created_at":         e.CreatedAt,
			"access_policy_id":   e.AccessPolicyID,
		})
	})

	f.Post("/entities/:id/ask", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		ent, err := d.Entities.Get(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		if err := requireEntityAction(c, d, principal, ent, "view"); err != nil {
			return err
		}
		var body qa.AskEntityInput
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		if ok, err := d.RoleBuilder.Assignments.PrincipalAllowsScenario(c.Context(), principal, body.ScenarioCode); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		} else if !ok {
			return fiber.NewError(fiber.StatusForbidden, "scenario not permitted for this principal")
		}
		if err := d.Retrieval.PreprocessAskMultimodal(c.Context(), &body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		out, err := d.Retrieval.AskEntity(c.Context(), principal, id, body)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		if tid, perr := uuid.Parse(out.TraceID); perr == nil {
			cit, _ := json.Marshal(out.Citations)
			sup, _ := json.Marshal(out.SupportingEntities)
			scp, _ := json.Marshal(out.Scope)
			priv := json.RawMessage(`{}`)
			if len(out.PrivacyTraceJSON) > 0 {
				priv = out.PrivacyTraceJSON
			}
			_ = d.Retrieval.PersistAnswerTrace(c.Context(), answertrace.Row{
				ID:                     tid,
				PrincipalID:            principal,
				EntityID:               id,
				Question:               body.Question,
				Answer:                 out.Answer,
				CitationsJSON:          cit,
				SupportingEntitiesJSON: sup,
				ScopeJSON:              scp,
				Model:                  retrieval_intelligence.OpenAIModelFromEnv(),
				RetrievalMode:          "entity_scoped",
				SupportingChunksJSON:   []byte("[]"),
				MetricsJSON:            []byte("{}"),
				PromptVersion:          "qa-synth-v1",
				PrivacyJSON:            priv,
			})
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("entity.asked", "user", &principal, "entity", &id, nil))
		return c.JSON(out)
	})

	// Global Ask: discovery via permission-scoped search (same filters as GET /search), then synthesis over top hits.
	f.Post("/ask", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		var body struct {
			qa.AskEntityInput
			DomainID        string `json:"domain_id"`
			Type            string `json:"type"`
			TruthMode       string `json:"truth_mode"`
			LifecycleState  string `json:"lifecycle_state"`
			FreshnessStatus string `json:"freshness_status"`
			ApprovalStatus  string `json:"approval_status"`
			RetrievalMode   string `json:"retrieval_mode"`
		}
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		askIn := body.AskEntityInput
		if ok, err := d.RoleBuilder.Assignments.PrincipalAllowsScenario(c.Context(), principal, askIn.ScenarioCode); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		} else if !ok {
			return fiber.NewError(fiber.StatusForbidden, "scenario not permitted for this principal")
		}
		if err := d.Retrieval.PreprocessAskMultimodal(c.Context(), &askIn); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		filters := retrieval_intelligence.BuildGlobalAskSearchFilters(askIn.Question, body.DomainID, body.Type, body.TruthMode, body.LifecycleState, body.FreshnessStatus, body.ApprovalStatus)
		out, gtrace, err := d.Retrieval.AskGlobal(c.Context(), principal, askIn, filters, body.RetrievalMode)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		anchorID := uuid.Nil
		if aid, ok := out.Scope["anchor_entity_id"].(string); ok {
			if parsed, perr := uuid.Parse(aid); perr == nil {
				anchorID = parsed
			}
		}
		if anchorID == uuid.Nil && len(out.SupportingEntities) > 0 {
			anchorID = out.SupportingEntities[0].EntityID
		}
		if tid, perr := uuid.Parse(out.TraceID); perr == nil {
			cit, _ := json.Marshal(out.Citations)
			sup, _ := json.Marshal(out.SupportingEntities)
			scp, _ := json.Marshal(out.Scope)
			metrics := []byte("{}")
			chunks := []byte("[]")
			rm := ""
			if gtrace != nil {
				if gtrace.Metrics != nil {
					metrics, _ = json.Marshal(gtrace.Metrics)
				}
				if len(gtrace.SupportingChunksJSON) > 0 {
					chunks = gtrace.SupportingChunksJSON
				}
				rm = gtrace.RetrievalMode
			}
			priv := json.RawMessage(`{}`)
			if len(out.PrivacyTraceJSON) > 0 {
				priv = out.PrivacyTraceJSON
			}
			_ = d.Retrieval.PersistAnswerTrace(c.Context(), answertrace.Row{
				ID:                     tid,
				PrincipalID:            principal,
				EntityID:               anchorID,
				Question:               askIn.Question,
				Answer:                 out.Answer,
				CitationsJSON:          cit,
				SupportingEntitiesJSON: sup,
				ScopeJSON:              scp,
				Model:                  retrieval_intelligence.OpenAIModelFromEnv(),
				RetrievalMode:          rm,
				SupportingChunksJSON:   chunks,
				MetricsJSON:            metrics,
				PromptVersion:          "qa-synth-v1",
				PrivacyJSON:            priv,
			})
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("ask.global", "user", &principal, "entity", &anchorID, nil))
		return c.JSON(out)
	})

	// GraphRAG entity browser (Neo4j-backed). Citations remain chunk-first in Ask; these endpoints expose the extracted graph.
	f.Get("/graphrag/entities", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if d.GraphRAG == nil || d.GraphRAG.Repo == nil {
			return fiber.NewError(fiber.StatusServiceUnavailable, "GraphRAG graph store not configured")
		}
		domainIDRaw := strings.TrimSpace(c.Query("domain_id"))
		if domainIDRaw == "" {
			return fiber.NewError(fiber.StatusBadRequest, "domain_id required")
		}
		domainID, err := uuid.Parse(domainIDRaw)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid domain_id")
		}
		granted, err := d.Search.GrantedDomainsForPrincipal(c.Context(), principal)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		if !containsUUID(granted, domainID) {
			return fiber.NewError(fiber.StatusForbidden, "domain not permitted")
		}
		q := strings.TrimSpace(c.Query("q"))
		limit := 25
		if v := strings.TrimSpace(c.Query("limit")); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 50 {
				limit = n
			}
		}
		list, err := d.GraphRAG.Repo.SearchEntities(c.Context(), domainID.String(), q, limit)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(fiber.Map{"entities": list})
	})
	f.Get("/graphrag/entities/:id/neighbors", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if d.GraphRAG == nil || d.GraphRAG.Repo == nil {
			return fiber.NewError(fiber.StatusServiceUnavailable, "GraphRAG graph store not configured")
		}
		domainIDRaw := strings.TrimSpace(c.Query("domain_id"))
		if domainIDRaw == "" {
			return fiber.NewError(fiber.StatusBadRequest, "domain_id required")
		}
		domainID, err := uuid.Parse(domainIDRaw)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid domain_id")
		}
		granted, err := d.Search.GrantedDomainsForPrincipal(c.Context(), principal)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		if !containsUUID(granted, domainID) {
			return fiber.NewError(fiber.StatusForbidden, "domain not permitted")
		}
		id := strings.TrimSpace(c.Params("id"))
		out, err := d.GraphRAG.Repo.Neighbors(c.Context(), domainID.String(), id, 30)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		return c.JSON(out)
	})

	f.Get("/answer-traces/:id", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		row, err := d.AnswerTrace.Get(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		if row.PrincipalID != principal {
			if err := requireCanManageIdentity(c, d, principal); err != nil {
				return err
			}
		}
		return c.JSON(row)
	})

	f.Post("/ai/summarize", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if err := requireCanManageIdentity(c, d, principal); err != nil {
			return err
		}
		var body struct {
			Text string `json:"text"`
		}
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		txt := strings.TrimSpace(body.Text)
		if txt == "" {
			return fiber.NewError(fiber.StatusBadRequest, "text required")
		}
		if len(txt) > 120_000 {
			return fiber.NewError(fiber.StatusBadRequest, "text too large")
		}
		client, err := llm.NewOpenAIFromEnv()
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		// Phase 4.1.1 follow-up: prompt loaded from the central registry so
		// the template id is recorded in answer_traces.privacy_json.
		tpl, err := prompts.Get("ai_summarize.v1")
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		cid := uuid.New().String()
		segs := []privacy.TextSegment{{Field: "free_text", Text: txt}}
		inv := privacy.InvokeInput{
			System:            tpl.SystemPrompt,
			Segments:          segs,
			PolicyCtx:         privacy.PolicyContext{Scenario: "ai_summarize", OutputType: "summary"},
			PromptTemplateID:  tpl.ID,
			CorrelationID:     cid,
			Principal:         principal,
			RehydrationMode:   privacy.RehydrationPartial,
			PublicationMode:   "reviewed_publish",
			OutputSensitivity: 0,
		}
		res, err := d.PrivacyGateway.InvokeOpenAI(c.Context(), client, inv)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("ai.summarize", "user", &principal, "ai", nil, nil))
		return c.JSON(fiber.Map{"summary": res.Answer})
	})

	f.Post("/ai/draft-suggestions", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		var body struct {
			EntityID uuid.UUID `json:"entity_id"`
		}
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		if body.EntityID == uuid.Nil {
			return fiber.NewError(fiber.StatusBadRequest, "entity_id required")
		}
		ent, err := d.Entities.Get(c.Context(), body.EntityID)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		if err := requireEntityAction(c, d, principal, ent, "view"); err != nil {
			return err
		}
		if err := requireEntityAction(c, d, principal, ent, "edit"); err != nil {
			return err
		}
		if ent.LifecycleState != "draft" {
			return fiber.NewError(fiber.StatusBadRequest, "draft suggestions are only available for draft lifecycle entities")
		}
		client, err := llm.NewOpenAIFromEnv()
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		var sb strings.Builder
		sb.WriteString("title: ")
		sb.WriteString(ent.Title)
		sb.WriteString("\ntype: ")
		sb.WriteString(ent.Type)
		sb.WriteString("\ntruth_mode: ")
		sb.WriteString(ent.TruthMode)
		sb.WriteString("\n")
		if ent.Summary != nil {
			sb.WriteString("summary:\n")
			sb.WriteString(*ent.Summary)
			sb.WriteString("\n")
		}
		if ent.Body != nil {
			sb.WriteString("body:\n")
			sb.WriteString(*ent.Body)
		}
		// Phase 4.1.1 follow-up: prompt loaded from the central registry.
		tpl, err := prompts.Get("ai_draft_suggestions.v1")
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		dom := ent.DomainID
		cid := uuid.New().String()
		segs := []privacy.TextSegment{{Field: "free_text", Text: sb.String()}}
		inv := privacy.InvokeInput{
			System:            tpl.SystemPrompt,
			Segments:          segs,
			PolicyCtx:         privacy.PolicyContext{DomainID: &dom, Scenario: "ai_draft_suggestions", OutputType: "draft_suggestions"},
			PromptTemplateID:  tpl.ID,
			CorrelationID:     cid,
			Principal:         principal,
			RehydrationMode:   privacy.RehydrationPartial,
			EvidenceEntities:  []*knowledge_core.Entity{ent},
			PublicationMode:   "reviewed_publish",
			OutputSensitivity: ent.SensitivityLevel,
		}
		res, err := d.PrivacyGateway.InvokeOpenAI(c.Context(), client, inv)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("ai.draft_suggestions", "user", &principal, "entity", &ent.ID, nil))
		return c.JSON(fiber.Map{
			"suggestions_markdown": res.Answer,
			"disclaimer":           "AI-generated suggestions; review before applying. Does not change workflow or publication state.",
		})
	})

	f.Post("/entities", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		var in knowledge_core.CreateEntityInput
		if err := c.BodyParser(&in); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		if in.OwnerID == nil {
			in.OwnerID = &principal
		}
		domainID := in.DomainID
		sens := in.SensitivityLevel
		et := in.Type
		dec, err := d.Access.Evaluate(c.Context(), identity_access.EvaluateInput{
			PrincipalID:      principal,
			Action:           "create",
			ResourceType:     "entity",
			DomainID:         &domainID,
			SensitivityLevel: &sens,
			EntityType:       &et,
		})
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		if !dec.Allow || !dec.SensitivityOK {
			return fiber.NewError(fiber.StatusForbidden, "access denied")
		}
		e, err := d.Entities.Create(c.Context(), in)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		_ = d.Search.ReindexEntity(c.Context(), e.ID)
		_ = d.AuditOps.Write(c.Context(), auditInput("entity.created", "user", &principal, "entity", &e.ID, nil))
		return c.Status(fiber.StatusCreated).JSON(e)
	})

	f.Patch("/entities/:id", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		before, err := d.Entities.Get(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		if err := requireEntityAction(c, d, principal, before, "edit"); err != nil {
			return err
		}
		var in knowledge_core.PatchEntityInput
		if err := c.BodyParser(&in); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		e, err := d.Entities.Patch(c.Context(), id, in)
		if err != nil {
			if errors.Is(err, knowledge_core.ErrPatchPublishForbidden) {
				return fiber.NewError(fiber.StatusBadRequest, "use POST /entities/:id/publish to set lifecycle_state=published — PATCH cannot publish (Phase 4.2.1).")
			}
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		_ = d.Search.ReindexEntity(c.Context(), e.ID)
		_ = d.AuditOps.Write(c.Context(), auditInput("entity.updated", "user", &principal, "entity", &e.ID, nil))
		return c.JSON(e)
	})

	// POST /entities/:id/publish — single canonical path to move an entity to
	// lifecycle_state="published" (Phase 4.2.1). Requires the publish action
	// in the entity's domain (action="publish" in AccessEvaluator). PATCH
	// rejects setting lifecycle_state="published" directly so the version
	// snapshot, approval stamps, search projection update, and audit emission
	// happen exactly once via this route + EntityRepo.Publish.
	f.Post("/entities/:id/publish", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		ent, err := d.Entities.Get(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		if err := requireEntityAction(c, d, principal, ent, "publish"); err != nil {
			return err
		}
		pres, err := d.Entities.Publish(c.Context(), id, principal)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		if pres.WasPublished {
			_ = d.Search.ReindexEntity(c.Context(), id)
			_ = d.AuditOps.Write(c.Context(), auditInput("entity.published", "user", &principal, "entity", &id, ptr("direct")))
		}
		return c.JSON(fiber.Map{
			"entity":         pres.Entity,
			"was_published":  pres.WasPublished,
			"was_idempotent": pres.WasIdempotent,
		})
	})

	f.Get("/entities/:id/links", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		ent, err := d.Entities.Get(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		if err := requireEntityAction(c, d, principal, ent, "view"); err != nil {
			return err
		}
		list, err := d.Entities.ListLinks(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(list)
	})

	// GET /entities/:id/graph-explore — bounded one-hop entity neighbours via
	// GraphRAG co-mention (Phase 2.3.1). Permission filter is applied internally
	// by retrieval_intelligence (Phase 1.1.1 hardening): denied neighbours are
	// silently dropped and counted in `denied_count`. Requires NEO4J_URL.
	f.Get("/entities/:id/graph-explore", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		maxNodes := 24
		if q := c.Query("max_nodes"); q != "" {
			if v, e := strconv.Atoi(q); e == nil && v > 0 {
				maxNodes = v
			}
		}
		out, err := d.Retrieval.GraphExplore(c.Context(), principal, id, maxNodes)
		if err != nil {
			if errors.Is(err, retrieval_intelligence.ErrAccessDenied) {
				return fiber.NewError(fiber.StatusForbidden, "access denied")
			}
			if errors.Is(err, retrieval_intelligence.ErrGraphNotConfigured) {
				return fiber.NewError(fiber.StatusServiceUnavailable, "GraphRAG is not configured (set NEO4J_URL)")
			}
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(out)
	})

	f.Get("/entities/:id/related", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		root, err := d.Entities.Get(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		if err := requireEntityAction(c, d, principal, root, "view"); err != nil {
			return err
		}

		limit := 6
		if q := c.Query("limit"); q != "" {
			if v, e := strconv.Atoi(q); e == nil {
				if v < 1 {
					limit = 1
				} else if v > 12 {
					limit = 12
				} else {
					limit = v
				}
			}
		}

		depth := 1
		if c.Query("depth") == "2" {
			depth = 2
		}

		links, err := d.Entities.ListLinks(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}

		type relatedItem struct {
			Entity knowledge_core.Entity `json:"entity"`
			Reason string                `json:"reason"`
		}

		seen := map[uuid.UUID]struct{}{}
		out := make([]relatedItem, 0, limit)
		var oneHopForExpansion []uuid.UUID

		tryAdd := func(other uuid.UUID, reason string) bool {
			if other == uuid.Nil || other == id {
				return false
			}
			if _, ok := seen[other]; ok {
				return false
			}
			ent, gerr := d.Entities.Get(c.Context(), other)
			if gerr != nil {
				return false
			}
			if err := requireEntityAction(c, d, principal, ent, "view"); err != nil {
				return false
			}
			seen[other] = struct{}{}
			out = append(out, relatedItem{Entity: *ent, Reason: reason})
			return true
		}

		for _, l := range links {
			if len(out) >= limit {
				break
			}
			var other uuid.UUID
			if l.FromEntityID == id {
				other = l.ToEntityID
			} else {
				other = l.FromEntityID
			}
			if tryAdd(other, "linked:"+l.RelationType) {
				oneHopForExpansion = append(oneHopForExpansion, other)
			}
		}

		// Bounded 2-hop: expand from at most 4 direct neighbors; each target is view-checked (no relation bypass).
		if depth == 2 && len(out) < limit && len(oneHopForExpansion) > 0 {
			maxSeeds := 4
			if len(oneHopForExpansion) < maxSeeds {
				maxSeeds = len(oneHopForExpansion)
			}
			for s := 0; s < maxSeeds && len(out) < limit; s++ {
				seed := oneHopForExpansion[s]
				links2, err2 := d.Entities.ListLinks(c.Context(), seed)
				if err2 != nil {
					return fiber.NewError(fiber.StatusInternalServerError, err2.Error())
				}
				for _, l := range links2 {
					if len(out) >= limit {
						break
					}
					var other uuid.UUID
					if l.FromEntityID == seed {
						other = l.ToEntityID
					} else {
						other = l.FromEntityID
					}
					_ = tryAdd(other, "linked_2hop:"+l.RelationType+":via:"+seed.String())
				}
			}
		}

		return c.JSON(out)
	})

	f.Get("/entities/:id/recommendations", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		root, err := d.Entities.Get(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		if err := requireEntityAction(c, d, principal, root, "view"); err != nil {
			return err
		}
		limit := 8
		if q := c.Query("limit"); q != "" {
			if v, e := strconv.Atoi(q); e == nil {
				if v < 1 {
					limit = 1
				} else if v > 16 {
					limit = 16
				} else {
					limit = v
				}
			}
		}
		items, err := recommendations.ForEntity(c.Context(), d.Entities, d.ContentHub, root, limit,
			func(ctx context.Context, e *knowledge_core.Entity) bool {
				return entityViewOK(ctx, d, principal, e)
			})
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(items)
	})

	f.Get("/recommendations/browse", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		entityType := strings.TrimSpace(c.Query("type"))
		if entityType == "" {
			return fiber.NewError(fiber.StatusBadRequest, "type required")
		}
		granted, err := d.Access.DomainIDsWithGrant(c.Context(), principal)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		domainIDs := granted
		if q := strings.TrimSpace(c.Query("domain_id")); q != "" {
			did, perr := uuid.Parse(q)
			if perr != nil {
				return fiber.NewError(fiber.StatusBadRequest, "invalid domain_id")
			}
			allowed := false
			for _, g := range granted {
				if g == did {
					allowed = true
					break
				}
			}
			if !allowed {
				return fiber.NewError(fiber.StatusForbidden, "domain not granted")
			}
			domainIDs = []uuid.UUID{did}
		}
		limit := 6
		if q := c.Query("limit"); q != "" {
			if v, e := strconv.Atoi(q); e == nil {
				if v >= 1 && v <= 16 {
					limit = v
				}
			}
		}
		items, err := recommendations.ForBrowse(c.Context(), d.Entities, domainIDs, entityType, limit,
			func(ctx context.Context, e *knowledge_core.Entity) bool {
				return entityViewOK(ctx, d, principal, e)
			})
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(items)
	})

	f.Post("/entities/:id/links", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		fromID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		fromEnt, err := d.Entities.Get(c.Context(), fromID)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		if err := requireEntityAction(c, d, principal, fromEnt, "edit"); err != nil {
			return err
		}
		var body struct {
			ToEntityID   uuid.UUID `json:"to_entity_id"`
			RelationType string    `json:"relation_type"`
		}
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		toEnt, err := d.Entities.Get(c.Context(), body.ToEntityID)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "to_entity not found")
		}
		if err := requireEntityAction(c, d, principal, toEnt, "view"); err != nil {
			return err
		}
		link, err := d.Entities.AddLink(c.Context(), fromID, body.ToEntityID, body.RelationType, &principal)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.Status(fiber.StatusCreated).JSON(link)
	})

	f.Get("/entities/:id/provenance", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		ent, err := d.Entities.Get(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		if err := requireEntityAction(c, d, principal, ent, "view"); err != nil {
			return err
		}
		list, err := d.Entities.ListProvenance(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(list)
	})

	f.Get("/entities/:id/evidence", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		ent, err := d.Entities.Get(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		if err := requireEntityAction(c, d, principal, ent, "view"); err != nil {
			return err
		}

		evidence, err := d.Entities.ListProvenanceEvidence(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}

		allowRaw := true
		if err := requireViewRaw(c, d, principal, ent.DomainID, ent.SensitivityLevel, nil); err != nil {
			allowRaw = false
		}
		allowNorm := true
		if err := requireIngestionDerivedView(c, d, principal, ent.DomainID, ent.SensitivityLevel, nil); err != nil {
			allowNorm = false
		}

		type evidenceItem struct {
			Record              knowledge_core.ProvenanceRecord `json:"record"`
			RawArtifactIDs      []uuid.UUID                     `json:"raw_artifact_ids,omitempty"`
			NormalizedRecordIDs []uuid.UUID                     `json:"normalized_record_ids,omitempty"`
		}
		out := make([]evidenceItem, 0, len(evidence))
		for _, e := range evidence {
			item := evidenceItem{Record: e.Record}
			if allowRaw {
				item.RawArtifactIDs = e.RawArtifactIDs
			}
			if allowNorm {
				item.NormalizedRecordIDs = e.NormalizedRecordIDs
			}
			out = append(out, item)
		}

		return c.JSON(fiber.Map{
			"entity_id":           id,
			"can_view_raw":        allowRaw,
			"can_view_normalized": allowNorm,
			"evidence":            out,
		})
	})

	f.Get("/entity-versions/:id", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		ver, err := d.Entities.GetEntityVersion(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		ent, err := d.Entities.Get(c.Context(), ver.EntityID)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		if err := requireEntityAction(c, d, principal, ent, "view"); err != nil {
			return err
		}
		return c.JSON(ver)
	})

	f.Get("/connectors", func(c *fiber.Ctx) error {
		list, err := d.Ingestion.ListConnectors(c.Context())
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(list)
	})

	// POST /connectors/webhook/:adapter_kind/:source_feed_id — receives push
	// deliveries for event-driven connectors (Phase 2.2.3 pilot — Slack).
	//
	// Auth: NONE at the HTTP layer. The adapter is the source of truth for
	// "is this delivery authentic?" via per-feed signature verification (HMAC,
	// per the connector's own protocol). The route's job is transport: load the
	// feed, resolve the adapter, hand it the raw body + headers, and persist or
	// reply. This intentionally mirrors how Slack/GitHub/Stripe webhook
	// receivers are typically deployed (no session cookie, signed body).
	//
	// The route ALWAYS reads the body in full before invoking the adapter so
	// HMAC verification can hash the exact bytes Slack signed; we cap at 1 MiB
	// to avoid memory amplification from a misbehaving sender.
	f.Post("/connectors/webhook/:adapter_kind/:source_feed_id", func(c *fiber.Ctx) error {
		adapterKind := strings.TrimSpace(c.Params("adapter_kind"))
		feedID, err := uuid.Parse(c.Params("source_feed_id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid source_feed_id")
		}
		const maxBodyBytes = 1 << 20
		body := c.Body()
		if len(body) > maxBodyBytes {
			return fiber.NewError(fiber.StatusRequestEntityTooLarge, "webhook payload exceeds 1 MiB")
		}

		feed, err := d.Ingestion.GetSourceFeed(c.Context(), feedID)
		if err != nil || feed == nil {
			return fiber.NewError(fiber.StatusNotFound, "source feed not found")
		}
		conn, err := d.Ingestion.GetConnector(c.Context(), feed.ConnectorID)
		if err != nil || conn == nil {
			return fiber.NewError(fiber.StatusNotFound, "connector not found")
		}
		if conn.Type != adapterKind {
			return fiber.NewError(fiber.StatusBadRequest, "adapter_kind does not match feed connector")
		}

		handler, ok := d.Ingestion.Registry.WebhookHandlerForType(adapterKind)
		if !ok {
			return fiber.NewError(fiber.StatusNotImplemented, "connector does not support webhooks")
		}

		headers := map[string][]string{}
		c.Request().Header.VisitAll(func(k, v []byte) {
			key := string(k)
			headers[key] = append(headers[key], string(v))
		})

		// Copy body bytes — Fiber reuses its buffer across requests, so the
		// adapter's HMAC verification could otherwise hash a future request.
		bodyCopy := make([]byte, len(body))
		copy(bodyCopy, body)

		result, err := handler.HandleWebhook(c.Context(), ingestion_connectors.WebhookRequest{
			Connector: *conn,
			Feed:      *feed,
			Headers:   headers,
			Body:      bodyCopy,
		})
		if err != nil {
			// Map well-known sentinel errors so observability can distinguish
			// "we rejected the delivery" (4xx) from "we crashed" (5xx).
			// Each adapter exposes its own sentinels with consistent semantics:
			// auth failure → 401, missing config → 503, malformed body → 400.
			switch {
			case errors.Is(err, slack.ErrWebhookBadSignature),
				errors.Is(err, slack.ErrWebhookStaleTimestamp),
				errors.Is(err, slack.ErrWebhookMissingHeaders),
				errors.Is(err, mattermost.ErrWebhookBadToken):
				return fiber.NewError(fiber.StatusUnauthorized, err.Error())
			case errors.Is(err, slack.ErrWebhookSigningSecretMissing),
				errors.Is(err, mattermost.ErrWebhookTokenMissing):
				return fiber.NewError(fiber.StatusServiceUnavailable, err.Error())
			case errors.Is(err, slack.ErrWebhookMalformedBody),
				errors.Is(err, mattermost.ErrWebhookMalformedBody):
				return fiber.NewError(fiber.StatusBadRequest, err.Error())
			default:
				return fiber.NewError(fiber.StatusInternalServerError, err.Error())
			}
		}

		// URL-verification handshake: Slack sends `{type:"url_verification",challenge:"..."}`
		// and expects the challenge string back as text. Other providers may use
		// the same Challenge field for their own handshakes; the route does not
		// need to know.
		if result != nil && result.Challenge != "" {
			c.Set("Content-Type", "text/plain")
			return c.SendString(result.Challenge)
		}

		runID, ingested, deduped, err := d.Ingestion.IngestWebhookResult(c.Context(), *feed, *conn, result)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(fiber.Map{
			"run_id":   runID,
			"ingested": ingested,
			"deduped":  deduped,
		})
	})

	f.Get("/connectors/:id", func(c *fiber.Ctx) error {
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		co, err := d.Ingestion.GetConnector(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		return c.JSON(co)
	})

	f.Post("/connectors", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if err := requireCanManageIdentity(c, d, principal); err != nil {
			return err
		}
		var body struct {
			Type             string          `json:"type"`
			DisplayName      string          `json:"display_name"`
			AuthMode         string          `json:"auth_mode"`
			CapabilitiesJSON json.RawMessage `json:"capabilities_json"`
		}
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		co, err := d.Ingestion.CreateConnector(c.Context(), body.Type, body.DisplayName, body.AuthMode, body.CapabilitiesJSON)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("connector.created", "user", &principal, "connector", &co.ID, nil))
		return c.Status(fiber.StatusCreated).JSON(co)
	})

	f.Patch("/connectors/:id", func(c *fiber.Ctx) error {
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
		var body struct {
			DisplayName *string `json:"display_name"`
			Status      *string `json:"status"`
			AuthMode    *string `json:"auth_mode"`
		}
		_ = c.BodyParser(&body)
		co, err := d.Ingestion.PatchConnector(c.Context(), id, body.DisplayName, body.Status, body.AuthMode)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("connector.updated", "user", &principal, "connector", &id, nil))
		return c.JSON(co)
	})

	f.Get("/source-feeds", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		domains, err := d.Access.DomainIDsWithGrant(c.Context(), principal)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		limit := clampLimit(c.Query("limit"), 200, 500)
		list, err := d.Ingestion.ListSourceFeedsInDomains(c.Context(), domains, limit)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(list)
	})

	f.Get("/source-feeds/:id", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		sf, err := d.Ingestion.GetSourceFeed(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		domainID := sf.DomainID
		sens := sf.SensitivityLevel
		dec, err := d.Access.Evaluate(c.Context(), identity_access.EvaluateInput{
			PrincipalID:      principal,
			Action:           "manage_source_feed",
			ResourceType:     "source_feed",
			ResourceID:       &id,
			DomainID:         &domainID,
			SensitivityLevel: &sens,
		})
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		if !dec.Allow || !dec.SensitivityOK {
			return fiber.NewError(fiber.StatusForbidden, "access denied")
		}
		return c.JSON(sf)
	})

	f.Post("/source-feeds", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		var in ingestion_connectors.CreateSourceFeedInput
		if err := c.BodyParser(&in); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		if in.OwnerID == uuid.Nil {
			in.OwnerID = principal
		}
		if err := requireManageSourceFeedNewInDomain(c, d, principal, in.DomainID, in.SensitivityLevel); err != nil {
			return err
		}
		// Phase 4.2.2: adapter-level validation at config-save. Build a candidate
		// feed shape and reject the request if the adapter rejects the config —
		// surfaces issues immediately rather than at the first sync attempt.
		candidate := &ingestion_connectors.SourceFeed{
			ConnectorID:         in.ConnectorID,
			DomainID:            in.DomainID,
			SensitivityLevel:    in.SensitivityLevel,
			KnowledgeScope:      in.KnowledgeScope,
			ConnectorConfigJSON: in.ConnectorConfigJSON,
		}
		if err := d.Ingestion.ValidateSourceFeed(c.Context(), candidate); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		sf, err := d.Ingestion.CreateSourceFeed(c.Context(), in)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("source_feed.created", "user", &principal, "source_feed", &sf.ID, nil))
		return c.Status(fiber.StatusCreated).JSON(sf)
	})

	f.Patch("/source-feeds/:id", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		feed0, err := d.Ingestion.GetSourceFeed(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		if err := requireManageSourceFeed(c, d, principal, feed0); err != nil {
			return err
		}
		var body struct {
			DisplayName *string         `json:"display_name"`
			Config      json.RawMessage `json:"connector_config_json"`
		}
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		// Phase 4.2.2: adapter-level validation BEFORE persisting. We construct
		// a candidate feed by overlaying the new config onto the saved feed
		// (display_name does not affect adapter validation). If validation
		// fails, no DB write happens — operator sees the error immediately.
		candidate := *feed0
		if len(body.Config) > 0 {
			candidate.ConnectorConfigJSON = body.Config
		}
		if err := d.Ingestion.ValidateSourceFeed(c.Context(), &candidate); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		sf, err := d.Ingestion.PatchSourceFeed(c.Context(), id, body.DisplayName, body.Config)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("source_feed.updated", "user", &principal, "source_feed", &sf.ID, nil))
		return c.JSON(sf)
	})

	// POST /source-feeds/validate — pure dry-run validation against a candidate
	// feed (Phase 4.2.2). Used by the source-feed wizard to surface adapter
	// validation issues inline before the operator commits via POST/PATCH.
	// Auth: identity admin or any principal who could create the feed in the
	// target domain — same gate as POST /source-feeds.
	f.Post("/source-feeds/validate", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		var in ingestion_connectors.CreateSourceFeedInput
		if err := c.BodyParser(&in); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		if err := requireManageSourceFeedNewInDomain(c, d, principal, in.DomainID, in.SensitivityLevel); err != nil {
			return err
		}
		candidate := &ingestion_connectors.SourceFeed{
			ConnectorID:         in.ConnectorID,
			DomainID:            in.DomainID,
			SensitivityLevel:    in.SensitivityLevel,
			KnowledgeScope:      in.KnowledgeScope,
			ConnectorConfigJSON: in.ConnectorConfigJSON,
		}
		if err := d.Ingestion.ValidateSourceFeed(c.Context(), candidate); err != nil {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
				"valid": false,
				"error": err.Error(),
			})
		}
		return c.JSON(fiber.Map{"valid": true})
	})

	f.Delete("/source-feeds/:id", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		feed0, err := d.Ingestion.GetSourceFeed(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		if err := requireManageSourceFeed(c, d, principal, feed0); err != nil {
			return err
		}
		if err := d.Ingestion.ArchiveSourceFeed(c.Context(), id); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("source_feed.archived", "user", &principal, "source_feed", &id, nil))
		return c.SendStatus(fiber.StatusNoContent)
	})

	f.Post("/source-feeds/:id/preview", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		feed, err := d.Ingestion.GetSourceFeed(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		domainID := feed.DomainID
		sens := feed.SensitivityLevel
		dec, err := d.Access.Evaluate(c.Context(), identity_access.EvaluateInput{
			PrincipalID:      principal,
			Action:           "manage_source_feed",
			ResourceType:     "source_feed",
			DomainID:         &domainID,
			SensitivityLevel: &sens,
		})
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		if !dec.Allow || !dec.SensitivityOK {
			return fiber.NewError(fiber.StatusForbidden, "access denied")
		}
		conn, err := d.Ingestion.GetConnector(c.Context(), feed.ConnectorID)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "connector not found")
		}
		prev, err := d.Ingestion.PreviewSourceFeed(c.Context(), feed, conn)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("source_feed.preview", "user", &principal, "source_feed", &id, nil))
		return c.JSON(prev)
	})

	f.Post("/source-feeds/:id/activate", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		feed0, err := d.Ingestion.GetSourceFeed(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		if err := requireManageSourceFeed(c, d, principal, feed0); err != nil {
			return err
		}
		if err := d.Ingestion.Activate(c.Context(), id); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("source_feed.activated", "user", &principal, "source_feed", &id, nil))
		return c.SendStatus(fiber.StatusNoContent)
	})

	f.Post("/source-feeds/:id/pause", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		feed0, err := d.Ingestion.GetSourceFeed(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		if err := requireManageSourceFeed(c, d, principal, feed0); err != nil {
			return err
		}
		_ = d.Ingestion.Pause(c.Context(), id)
		_ = d.AuditOps.Write(c.Context(), auditInput("source_feed.paused", "user", &principal, "source_feed", &id, nil))
		return c.SendStatus(fiber.StatusNoContent)
	})

	f.Post("/source-feeds/:id/resume", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		feed0, err := d.Ingestion.GetSourceFeed(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		if err := requireManageSourceFeed(c, d, principal, feed0); err != nil {
			return err
		}
		_ = d.Ingestion.Resume(c.Context(), id)
		_ = d.AuditOps.Write(c.Context(), auditInput("source_feed.resumed", "user", &principal, "source_feed", &id, nil))
		return c.SendStatus(fiber.StatusNoContent)
	})

	f.Post("/source-feeds/:id/sync", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		feed0, err := d.Ingestion.GetSourceFeed(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		if err := requireManageSourceFeed(c, d, principal, feed0); err != nil {
			return err
		}
		if d.JobQueue != nil && d.JobQueue.Enabled() {
			if err := d.JobQueue.EnqueueConnectorSourceSync(c.Context(), id); err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, err.Error())
			}
			_ = d.AuditOps.Write(c.Context(), auditInput("source_feed.sync_queued", "user", &principal, "source_feed", &id, nil))
			return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
				"queued":         true,
				"source_feed_id": id.String(),
			})
		}
		run, err := d.Ingestion.SyncSourceFeed(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("source_feed.sync", "user", &principal, "ingestion_run", &run.ID, nil))
		return c.JSON(run)
	})

	f.Get("/source-feeds/:id/ingestion-runs", func(c *fiber.Ctx) error {
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		list, err := d.Ingestion.ListIngestionRuns(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(list)
	})

	f.Get("/ingestion-runs/:id", func(c *fiber.Ctx) error {
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		r, err := d.Ingestion.GetIngestionRun(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		return c.JSON(r)
	})

	f.Get("/source-feeds/:id/raw-artifacts", func(c *fiber.Ctx) error {
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
		if err := requireViewRaw(c, d, principal, feed.DomainID, feed.SensitivityLevel, nil); err != nil {
			return err
		}
		limit := 100
		if v := c.Query("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				limit = n
			}
		}
		list, err := d.Ingestion.ListRawArtifactsForFeed(c.Context(), feedID, limit)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(list)
	})

	f.Get("/raw-artifacts/:id", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		raw, err := d.Ingestion.GetRawArtifact(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		rid := raw.ID
		if err := requireViewRaw(c, d, principal, raw.DomainID, raw.FeedSensitivity, &rid); err != nil {
			return err
		}
		return c.JSON(raw)
	})

	f.Get("/source-feeds/:id/normalized-records", func(c *fiber.Ctx) error {
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
		if err := requireIngestionDerivedView(c, d, principal, feed.DomainID, feed.SensitivityLevel, nil); err != nil {
			return err
		}
		limit := 100
		if v := c.Query("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				limit = n
			}
		}
		list, err := d.Ingestion.ListNormalizedRecordsForFeed(c.Context(), feedID, limit)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(list)
	})

	f.Get("/normalized-records/:id", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		rec, err := d.Ingestion.GetNormalizedRecord(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		rid := rec.ID
		if err := requireIngestionDerivedView(c, d, principal, rec.DomainID, rec.FeedSensitivity, &rid); err != nil {
			return err
		}
		return c.JSON(rec)
	})

	f.Post("/normalized-records/:id/map-reference-document", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		rec, err := d.Ingestion.GetNormalizedRecord(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		rid := rec.ID
		if err := requireIngestionDerivedView(c, d, principal, rec.DomainID, rec.FeedSensitivity, &rid); err != nil {
			return err
		}
		var body struct {
			TruthMode *string `json:"truth_mode"`
		}
		_ = c.BodyParser(&body)
		tm := ""
		if body.TruthMode != nil {
			tm = *body.TruthMode
		}
		domainID := rec.DomainID
		sens := rec.FeedSensitivity
		dec, err := d.Access.Evaluate(c.Context(), identity_access.EvaluateInput{
			PrincipalID:      principal,
			Action:           "edit",
			ResourceType:     "entity",
			DomainID:         &domainID,
			SensitivityLevel: &sens,
		})
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		if !dec.Allow || !dec.SensitivityOK {
			return fiber.NewError(fiber.StatusForbidden, "access denied")
		}
		ent, err := ingestion_connectors.MapNormalizedRecordToReferenceDocument(c.Context(), d.Ingestion, d.Entities, id, principal, tm)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("normalized_record.mapped_reference_document", "user", &principal, "entity", &ent.ID, nil))
		return c.Status(fiber.StatusCreated).JSON(ent)
	})

	audit_opstransport.RegisterReadRoutes(f, d.AuditOps, func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		return requireCanManageIdentity(c, d, principal)
	})

	f.Get("/knowledge-job-templates", func(c *fiber.Ctx) error {
		return c.JSON(knowledge_jobs.ListJobTemplatesPublic())
	})

	f.Get("/knowledge-jobs/engine-metadata", func(c *fiber.Ctx) error {
		if _, err := httpcontext.RequirePrincipal(c); err != nil {
			return err
		}
		return c.JSON(knowledge_jobs.KnowledgeJobsEngineMetadataResponse())
	})

	f.Get("/job-builder/presets", func(c *fiber.Ctx) error {
		if _, err := httpcontext.RequirePrincipal(c); err != nil {
			return err
		}
		merged, err := d.Jobs.ListBuilderPresetsMerged(c.Context())
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(merged)
	})

	f.Get("/knowledge-jobs", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		expand := c.Query("expand") == "scenarios"
		if expand {
			items, err := d.Jobs.ListWithScenarioSummary(c.Context())
			if err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, err.Error())
			}
			visible := make([]knowledge_jobs.JobListItem, 0, len(items))
			for i := range items {
				if principalCanViewKnowledgeJob(c.Context(), d, principal, &items[i].KnowledgeJob) {
					visible = append(visible, items[i])
				}
			}
			return c.JSON(visible)
		}
		list, err := d.Jobs.List(c.Context())
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		visible := make([]knowledge_jobs.KnowledgeJob, 0, len(list))
		for i := range list {
			if principalCanViewKnowledgeJob(c.Context(), d, principal, &list[i]) {
				visible = append(visible, list[i])
			}
		}
		return c.JSON(visible)
	})

	f.Get("/knowledge-jobs/:id", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		j, err := d.Jobs.Get(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		if !principalCanViewKnowledgeJob(c.Context(), d, principal, j) {
			return fiber.NewError(fiber.StatusForbidden, "access denied")
		}
		return c.JSON(j)
	})

	f.Post("/knowledge-jobs", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		var in knowledge_jobs.CreateJobInput
		if err := c.BodyParser(&in); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		if in.OwnerID == uuid.Nil {
			in.OwnerID = principal
		}
		if err := requireCreateKnowledgeJobCapability(c, d, principal, in); err != nil {
			return err
		}
		j, err := d.Jobs.Create(c.Context(), in)
		if err != nil {
			if knowledge_jobs.IsClientJobDefinitionError(err) {
				return fiber.NewError(fiber.StatusBadRequest, err.Error())
			}
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.Status(fiber.StatusCreated).JSON(j)
	})

	f.Patch("/knowledge-jobs/:id", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		j0, err := d.Jobs.Get(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		if !principalCanManageKnowledgeJob(c.Context(), d, principal, j0) {
			return fiber.NewError(fiber.StatusForbidden, "access denied")
		}
		var patch knowledge_jobs.PatchJobInput
		if err := c.BodyParser(&patch); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		j, err := d.Jobs.Update(c.Context(), id, patch)
		if err != nil {
			if knowledge_jobs.IsClientJobDefinitionError(err) {
				return fiber.NewError(fiber.StatusBadRequest, err.Error())
			}
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("job.definition.updated", "user", &principal, "knowledge_job", &id, nil))
		return c.JSON(j)
	})

	f.Post("/knowledge-jobs/:id/clone", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		j0, err := d.Jobs.Get(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		if !principalCanManageKnowledgeJob(c.Context(), d, principal, j0) {
			return fiber.NewError(fiber.StatusForbidden, "access denied")
		}
		j, err := d.Jobs.Clone(c.Context(), id, principal)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("job.cloned", "user", &principal, "knowledge_job", &j.ID, nil))
		return c.Status(fiber.StatusCreated).JSON(j)
	})

	f.Get("/knowledge-jobs/:id/preview", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		j, err := d.Jobs.Get(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		if !principalCanViewKnowledgeJob(c.Context(), d, principal, j) {
			return fiber.NewError(fiber.StatusForbidden, "access denied")
		}
		p, err := d.Jobs.BuildJobPreview(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(p)
	})

	f.Post("/knowledge-jobs/:id/test-run", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		j, err := d.Jobs.Get(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		if !principalMayRunKnowledgeJob(c.Context(), d, principal, j) {
			return fiber.NewError(fiber.StatusForbidden, "operator cannot run job")
		}
		var body struct {
			DryRun bool `json:"dry_run"`
		}
		_ = c.BodyParser(&body)
		if body.DryRun {
			p, err := d.Jobs.TestRunDry(c.Context(), id)
			if err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, err.Error())
			}
			valid := len(p.ValidationErrors) == 0
			return c.JSON(fiber.Map{"valid": valid, "preview": p})
		}
		run, err := d.Jobs.Run(c.Context(), id, principal)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.Status(fiber.StatusAccepted).JSON(run)
	})

	f.Post("/knowledge-jobs/:id/scenario-bindings", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		if _, err := requireManageKnowledgeJobByID(c, d, principal, id); err != nil {
			return err
		}
		var rows []knowledge_jobs.ScenarioJobBindingWrite
		if err := c.BodyParser(&rows); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json: expected array of {scenario_id, relationship}")
		}
		if err := d.Jobs.ReplaceScenarioBindings(c.Context(), id, rows); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("job.scenario_bindings.replaced", "user", &principal, "knowledge_job", &id, nil))
		return c.SendStatus(fiber.StatusNoContent)
	})

	f.Get("/knowledge-jobs/:id/operators", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		j, err := d.Jobs.Get(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		if !principalCanViewKnowledgeJob(c.Context(), d, principal, j) {
			return fiber.NewError(fiber.StatusForbidden, "access denied")
		}
		list, err := d.Jobs.ListOperators(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(list)
	})

	f.Post("/knowledge-jobs/:id/operators", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		if _, err := requireManageKnowledgeJobByID(c, d, principal, id); err != nil {
			return err
		}
		var body struct {
			UserIDs []uuid.UUID `json:"user_ids"`
		}
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		if err := d.Jobs.ReplaceOperators(c.Context(), id, body.UserIDs); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("job.operators.replaced", "user", &principal, "knowledge_job", &id, nil))
		return c.SendStatus(fiber.StatusNoContent)
	})

	f.Post("/knowledge-jobs/:id/run", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		j, err := d.Jobs.Get(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		if !principalMayRunKnowledgeJob(c.Context(), d, principal, j) {
			return fiber.NewError(fiber.StatusForbidden, "operator cannot run job")
		}
		run, err := d.Jobs.Run(c.Context(), id, principal)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.Status(fiber.StatusAccepted).JSON(run)
	})

	f.Get("/job-runs/:id/outputs", func(c *fiber.Ctx) error {
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
		list, err := d.Jobs.ListOutputsForRun(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(list)
	})

	f.Get("/job-runs/:id", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		r, err := d.Jobs.GetRun(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		j, err := d.Jobs.Get(c.Context(), r.KnowledgeJobID)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		if !principalCanViewKnowledgeJob(c.Context(), d, principal, j) {
			return fiber.NewError(fiber.StatusForbidden, "access denied")
		}
		return c.JSON(r)
	})

	f.Get("/knowledge-jobs/:id/triggers", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		if _, err := requireManageKnowledgeJobByID(c, d, principal, id); err != nil {
			return err
		}
		list, err := d.Jobs.ListTriggers(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(list)
	})

	f.Post("/knowledge-jobs/:id/triggers", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		jobID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		if _, err := requireManageKnowledgeJobByID(c, d, principal, jobID); err != nil {
			return err
		}
		var body struct {
			TriggerType      string          `json:"trigger_type"`
			ScheduleExpr     *string         `json:"schedule_expr"`
			EventFilterJSON  json.RawMessage `json:"event_filter_json"`
			WindowConfigJSON json.RawMessage `json:"window_config_json"`
		}
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		t, err := d.Jobs.CreateTrigger(c.Context(), jobID, knowledge_jobs.CreateTriggerInput{
			TriggerType:      body.TriggerType,
			ScheduleExpr:     body.ScheduleExpr,
			EventFilterJSON:  body.EventFilterJSON,
			WindowConfigJSON: body.WindowConfigJSON,
		})
		if err != nil {
			if knowledge_jobs.IsClientTriggerDefinitionError(err) {
				return fiber.NewError(fiber.StatusBadRequest, err.Error())
			}
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("job_trigger.created", "user", &principal, "knowledge_job", &jobID, nil))
		return c.Status(fiber.StatusCreated).JSON(t)
	})

	f.Patch("/job-triggers/:id", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		tr, err := d.Jobs.GetTrigger(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		if _, err := requireManageKnowledgeJobByID(c, d, principal, tr.KnowledgeJobID); err != nil {
			return err
		}
		var body struct {
			Status       *string `json:"status"`
			ScheduleExpr *string `json:"schedule_expr"`
		}
		_ = c.BodyParser(&body)
		t, err := d.Jobs.PatchTrigger(c.Context(), id, body.Status, body.ScheduleExpr)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("job_trigger.updated", "user", &principal, "job_trigger", &id, nil))
		return c.JSON(t)
	})

	f.Delete("/job-triggers/:id", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		tr, err := d.Jobs.GetTrigger(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		if _, err := requireManageKnowledgeJobByID(c, d, principal, tr.KnowledgeJobID); err != nil {
			return err
		}
		if err := d.Jobs.DeleteTrigger(c.Context(), id); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return fiber.NewError(fiber.StatusNotFound, "not found")
			}
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("job_trigger.deleted", "user", &principal, "job_trigger", &id, nil))
		return c.SendStatus(fiber.StatusNoContent)
	})

	// GET /knowledge-jobs/runs — global recent runs across all jobs (Phase 2.1.3
	// CP listing). Query params: limit (default 50, max 200), status, job_type.
	// Auth: identity admin (operators). Per-job pagination remains available
	// at /knowledge-jobs/:id/runs.
	f.Get("/knowledge-jobs/runs", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if err := requireCanManageIdentity(c, d, principal); err != nil {
			return err
		}
		limit := 50
		if v := c.Query("limit"); v != "" {
			if n, e := strconv.Atoi(v); e == nil && n > 0 && n <= 200 {
				limit = n
			}
		}
		status := strings.TrimSpace(c.Query("status"))
		jobType := strings.TrimSpace(c.Query("job_type"))
		list, err := d.Jobs.ListRecentRuns(c.Context(), limit, status, jobType)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(fiber.Map{
			"items":     list,
			"count":     len(list),
			"limit":     limit,
			"status":    status,
			"job_type":  jobType,
			"truncated": len(list) >= limit,
		})
	})

	f.Get("/knowledge-jobs/:id/runs", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		j, err := d.Jobs.Get(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		if !principalCanViewKnowledgeJob(c.Context(), d, principal, j) {
			return fiber.NewError(fiber.StatusForbidden, "access denied")
		}
		limit := 50
		if v := c.Query("limit"); v != "" {
			if n, e := strconv.Atoi(v); e == nil && n > 0 && n <= 200 {
				limit = n
			}
		}
		list, err := d.Jobs.ListRunsForJob(c.Context(), id, limit)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(list)
	})

	f.Get("/job-outputs/:id", func(c *fiber.Ctx) error {
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
		o, err := d.Jobs.GetJobOutput(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		return c.JSON(o)
	})

	f.Get("/review-tasks", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		list, err := d.Review.List(c.Context())
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		visible := FilterReviewTasksForPrincipal(c.Context(), d, principal, list)
		return c.JSON(visible)
	})

	f.Get("/review-tasks/:id", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		t, err := requireReviewTaskEntityActions(c, d, principal, id, []string{"view"})
		if err != nil {
			return err
		}
		return c.JSON(t)
	})

	f.Post("/review-tasks/:id/start", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		if _, err := requireReviewTaskEntityActions(c, d, principal, id, []string{"view", "review"}); err != nil {
			return err
		}
		if err := d.Review.Start(c.Context(), id); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("review.started", "user", &principal, "review_task", &id, nil))
		return c.SendStatus(fiber.StatusNoContent)
	})

	f.Post("/review-tasks/:id/approve", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		var body struct {
			Note *string `json:"note"`
		}
		_ = c.BodyParser(&body)
		t, err := requireReviewTaskEntityActions(c, d, principal, id, []string{"view", "approve"})
		if err != nil {
			return err
		}
		if err := d.Review.Approve(c.Context(), id, body.Note); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		// Phase 4.2.1: route entity publish through EntityRepo.Publish so the
		// version snapshot, approval stamps, search projection update, and
		// audit emission happen via the single canonical path. Failures here
		// are non-fatal for the review approval itself — the review task is
		// already approved; we surface the publish error in logs and the
		// `review.approved` audit metadata so operators can investigate.
		if t.TargetType == "entity" {
			pres, perr := d.Entities.Publish(c.Context(), t.TargetID, principal)
			if perr != nil {
				log.Printf("review-tasks/%s/approve: publish entity %s: %v", id, t.TargetID, perr)
			} else if pres.WasPublished {
				_ = d.Search.ReindexEntity(c.Context(), t.TargetID)
				_ = d.AuditOps.Write(c.Context(), auditInput("entity.published", "user", &principal, "entity", &t.TargetID, ptr("review_approval")))
			}
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("review.approved", "user", &principal, "review_task", &id, ptr("approved")))
		return c.SendStatus(fiber.StatusNoContent)
	})

	f.Post("/review-tasks/:id/request-changes", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		if _, err := requireReviewTaskEntityActions(c, d, principal, id, []string{"view", "review"}); err != nil {
			return err
		}
		var body struct {
			Note string `json:"note"`
		}
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		note := body.Note
		if err := d.Review.RequestChanges(c.Context(), id, &note); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("review.changes_requested", "user", &principal, "review_task", &id, nil))
		return c.SendStatus(fiber.StatusNoContent)
	})

	f.Post("/review-tasks/:id/reject", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		if _, err := requireReviewTaskEntityActions(c, d, principal, id, []string{"view", "review"}); err != nil {
			return err
		}
		var body struct {
			Note *string `json:"note"`
		}
		_ = c.BodyParser(&body)
		if err := d.Review.Reject(c.Context(), id, body.Note); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("review.rejected", "user", &principal, "review_task", &id, ptr("rejected")))
		return c.SendStatus(fiber.StatusNoContent)
	})

	f.Get("/governance/reviews/overdue", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		doms, err := governancePublishDomainIDs(c, d, principal)
		if err != nil {
			return err
		}
		limit := clampLimit(c.Query("limit"), 200, 500)
		list, err := d.Review.ListOverdueInDomains(c.Context(), limit, doms)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(list)
	})

	f.Get("/governance/approval-queue", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		doms, err := governancePublishDomainIDs(c, d, principal)
		if err != nil {
			return err
		}
		limit := clampLimit(c.Query("limit"), 200, 500)
		list, err := d.ApprovalQueue.ListEntityReviewsInDomains(c.Context(), limit, doms)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(list)
	})

	f.Get("/governance/stale-content", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		doms, err := governancePublishDomainIDs(c, d, principal)
		if err != nil {
			return err
		}
		limit := clampLimit(c.Query("limit"), 200, 500)
		list, err := governance.ListStaleEntities(c.Context(), d.Pool, doms, limit)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(list)
	})

	f.Get("/governance/upkeep-suggestions", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		doms, err := governancePublishDomainIDs(c, d, principal)
		if err != nil {
			return err
		}
		limit := clampLimit(c.Query("limit"), 50, 200)
		list, err := governance.ListUpkeepSuggestions(c.Context(), d.Pool, doms, limit)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(list)
	})

	f.Get("/governance/publishing-queue", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if err := requireGovernancePublish(c, d, principal); err != nil {
			return err
		}
		doms, err := governancePublishDomainIDs(c, d, principal)
		if err != nil {
			return err
		}
		limit := clampLimit(c.Query("limit"), 50, 200)
		rows, err := d.Pool.Query(c.Context(), `
			SELECT id, type, title, domain_id, lifecycle_state, approval_status, truth_mode, freshness_status, updated_at
			FROM entities
			WHERE archived_at IS NULL AND domain_id = ANY($1)
			  AND lifecycle_state = 'published'
			  AND id NOT IN (SELECT entity_id FROM editorial_holdings)
			ORDER BY updated_at DESC
			LIMIT $2`, doms, limit)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		defer rows.Close()
		var out []fiber.Map
		for rows.Next() {
			var id, dom uuid.UUID
			var typ, title, life, appr, truth, fresh string
			var updated interface{}
			if err := rows.Scan(&id, &typ, &title, &dom, &life, &appr, &truth, &fresh, &updated); err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, err.Error())
			}
			out = append(out, fiber.Map{
				"id": id, "type": typ, "title": title, "domain_id": dom,
				"lifecycle_state": life, "approval_status": appr, "truth_mode": truth, "freshness_status": fresh, "updated_at": updated,
			})
		}
		return c.JSON(out)
	})

	f.Post("/governance/editorial/hold", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if err := requireGovernancePublish(c, d, principal); err != nil {
			return err
		}
		var body struct {
			EntityID uuid.UUID `json:"entity_id"`
			Reason   *string   `json:"reason"`
		}
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		ent, err := d.Entities.Get(c.Context(), body.EntityID)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		if err := requireEntityAction(c, d, principal, ent, "publish"); err != nil {
			return err
		}
		_, err = d.Pool.Exec(c.Context(), `
			INSERT INTO editorial_holdings (entity_id, held_by_id, reason) VALUES ($1,$2,$3)
			ON CONFLICT (entity_id) DO UPDATE SET held_by_id=$2, reason=$3, created_at=now()`,
			body.EntityID, principal, body.Reason)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.SendStatus(fiber.StatusNoContent)
	})

	f.Post("/governance/editorial/feature", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if err := requireGovernancePublish(c, d, principal); err != nil {
			return err
		}
		var body struct {
			HubID    uuid.UUID `json:"hub_id"`
			EntityID uuid.UUID `json:"entity_id"`
			Role     string    `json:"role"`
		}
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		if body.Role == "" {
			body.Role = "featured"
		}
		ent, err := d.Entities.Get(c.Context(), body.EntityID)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		if err := requireEntityAction(c, d, principal, ent, "view"); err != nil {
			return err
		}
		if err := d.ContentHub.AddItem(c.Context(), body.HubID, body.EntityID, body.Role, 0); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("hub.item_added", "user", &principal, "content_hub", &body.HubID, nil))
		return c.SendStatus(fiber.StatusNoContent)
	})

	f.Get("/governance/workflow-metrics", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if err := requireGovernancePublish(c, d, principal); err != nil {
			return err
		}
		doms, err := governancePublishDomainIDs(c, d, principal)
		if err != nil {
			return err
		}
		var open, overdue int64
		_ = d.Pool.QueryRow(c.Context(), `
			SELECT COUNT(*) FROM review_tasks rt
			JOIN entities e ON e.id = rt.target_id AND rt.target_type = 'entity' AND e.archived_at IS NULL
			WHERE rt.status IN ('pending','in_progress') AND e.domain_id = ANY($1)`, doms).Scan(&open)
		_ = d.Pool.QueryRow(c.Context(), `
			SELECT COUNT(*) FROM review_tasks rt
			JOIN entities e ON e.id = rt.target_id AND rt.target_type = 'entity' AND e.archived_at IS NULL
			WHERE rt.status IN ('pending','in_progress') AND rt.due_at IS NOT NULL AND rt.due_at < now() AND e.domain_id = ANY($1)`, doms).Scan(&overdue)
		return c.JSON(fiber.Map{"open_review_tasks": int(open), "overdue_review_tasks": int(overdue)})
	})

	f.Get("/governance/freshness-risk", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if err := requireGovernancePublish(c, d, principal); err != nil {
			return err
		}
		doms, err := governancePublishDomainIDs(c, d, principal)
		if err != nil {
			return err
		}
		limit := clampLimit(c.Query("limit"), 50, 200)
		rows, err := d.Pool.Query(c.Context(), `
			SELECT id, title, domain_id, type, sensitivity_level, freshness_status, lifecycle_state, truth_mode, updated_at,
				(CASE WHEN type IN ('policy','process_sop') THEN 40 ELSE 0 END
				 + CASE WHEN sensitivity_level >= 2 THEN 20 ELSE 0 END
				 + CASE WHEN freshness_status IN ('stale','expired','unknown') THEN 30 ELSE 10 END) AS risk_score
			FROM entities
			WHERE archived_at IS NULL AND domain_id = ANY($1)
			  AND freshness_status IS DISTINCT FROM 'fresh'
			ORDER BY risk_score DESC, updated_at ASC
			LIMIT $2`, doms, limit)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		defer rows.Close()
		var out []fiber.Map
		for rows.Next() {
			var id, dom uuid.UUID
			var title, typ, fresh, life, truth string
			var sens int
			var updated interface{}
			var score int
			if err := rows.Scan(&id, &title, &dom, &typ, &sens, &fresh, &life, &truth, &updated, &score); err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, err.Error())
			}
			out = append(out, fiber.Map{
				"id": id, "title": title, "domain_id": dom, "type": typ, "sensitivity_level": sens,
				"freshness_status": fresh, "lifecycle_state": life, "truth_mode": truth, "updated_at": updated, "risk_score": score,
			})
		}
		return c.JSON(out)
	})

	f.Get("/ops/answer-diagnostics", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if err := requireGovernancePublish(c, d, principal); err != nil {
			return err
		}
		limit := clampLimit(c.Query("limit"), 50, 200)
		rows, err := d.AnswerTrace.ListRecent(c.Context(), limit)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		var out []fiber.Map
		for _, row := range rows {
			var cites []any
			_ = json.Unmarshal(row.CitationsJSON, &cites)
			weak := len(cites) < 1 || len(row.Answer) < 40
			out = append(out, fiber.Map{
				"id": row.ID, "entity_id": row.EntityID, "question": row.Question, "answer_preview": truncateStr(row.Answer, 160),
				"citation_count": len(cites), "weak_evidence": weak, "model": row.Model, "created_at": row.CreatedAt,
			})
		}
		return c.JSON(out)
	})

	f.Get("/ops/search-insights", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if err := requireGovernancePublish(c, d, principal); err != nil {
			return err
		}
		limit := clampLimit(c.Query("limit"), 30, 200)
		rows, err := d.Pool.Query(c.Context(), `
			SELECT q, COUNT(*) AS c, AVG(hit_count)::float8 AS avg_hits
			FROM search_interaction_log
			WHERE created_at > now() - interval '30 days'
			GROUP BY q
			HAVING COUNT(*) >= 1
			ORDER BY c DESC
			LIMIT $1`, limit)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		defer rows.Close()
		var out []fiber.Map
		for rows.Next() {
			var q string
			var cnt int
			var avg float64
			if err := rows.Scan(&q, &cnt, &avg); err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, err.Error())
			}
			out = append(out, fiber.Map{"query": q, "count": cnt, "avg_hits": avg, "weak_pattern": avg < 1.5 && q != ""})
		}
		return c.JSON(out)
	})

	f.Get("/content-hubs", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		granted, err := d.Access.DomainIDsWithGrant(c.Context(), principal)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		list, err := d.ContentHub.ListInDomains(c.Context(), granted, 50)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(list)
	})

	f.Post("/content-hubs", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		var body struct {
			DomainID    uuid.UUID `json:"domain_id"`
			Slug        string    `json:"slug"`
			Title       string    `json:"title"`
			Description *string   `json:"description"`
		}
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		if body.Slug == "" || body.Title == "" {
			return fiber.NewError(fiber.StatusBadRequest, "slug and title required")
		}
		dec, err := d.Access.Evaluate(c.Context(), identity_access.EvaluateInput{
			PrincipalID:  principal,
			Action:       "publish",
			ResourceType: "domain",
			DomainID:     &body.DomainID,
		})
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		if !dec.Allow || !dec.SensitivityOK {
			return fiber.NewError(fiber.StatusForbidden, "access denied")
		}
		h, err := d.ContentHub.Create(c.Context(), body.DomainID, body.Slug, body.Title, body.Description, principal)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("content_hub.created", "user", &principal, "content_hub", &h.ID, nil))
		return c.Status(fiber.StatusCreated).JSON(h)
	})

	f.Get("/content-hubs/:id/view", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		h, err := d.ContentHub.GetByID(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		granted, err := d.Access.DomainIDsWithGrant(c.Context(), principal)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		ok := false
		for _, g := range granted {
			if g == h.DomainID {
				ok = true
				break
			}
		}
		if !ok {
			return fiber.NewError(fiber.StatusForbidden, "access denied")
		}
		items, err := d.ContentHub.ListItems(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		var entities []fiber.Map
		for _, it := range items {
			e, err := d.Entities.Get(c.Context(), it.EntityID)
			if err != nil {
				continue
			}
			if err := requireEntityAction(c, d, principal, e, "view"); err != nil {
				continue
			}
			entities = append(entities, fiber.Map{
				"id": e.ID, "type": e.Type, "title": e.Title, "truth_mode": e.TruthMode,
				"lifecycle_state": e.LifecycleState, "freshness_status": e.FreshnessStatus, "hub_role": it.Role,
			})
		}
		return c.JSON(fiber.Map{"hub": h, "entities": entities})
	})

	f.Post("/content-blocks", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		var body struct {
			DomainID uuid.UUID `json:"domain_id"`
			Title    string    `json:"title"`
			Body     string    `json:"body"`
		}
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		dec, err := d.Access.Evaluate(c.Context(), identity_access.EvaluateInput{
			PrincipalID:  principal,
			Action:       "publish",
			ResourceType: "domain",
			DomainID:     &body.DomainID,
		})
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		if !dec.Allow || !dec.SensitivityOK {
			return fiber.NewError(fiber.StatusForbidden, "access denied")
		}
		pid := principal
		b, err := d.ContentBlocks.Create(c.Context(), body.DomainID, &pid, body.Title, body.Body)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		return c.Status(fiber.StatusCreated).JSON(b)
	})

	f.Get("/entities/:id/content-blocks", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		e, err := d.Entities.Get(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		if err := requireEntityAction(c, d, principal, e, "view"); err != nil {
			return err
		}
		list, err := d.ContentBlocks.ListForEntity(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(list)
	})

	f.Post("/entities/:id/content-blocks", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		eid, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		ent, err := d.Entities.Get(c.Context(), eid)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		if err := requireEntityAction(c, d, principal, ent, "publish"); err != nil {
			return err
		}
		var body struct {
			BlockID   uuid.UUID `json:"block_id"`
			Placement string    `json:"placement"`
			SortOrder int       `json:"sort_order"`
		}
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		if err := d.ContentBlocks.AttachToEntity(c.Context(), eid, body.BlockID, body.Placement, body.SortOrder); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		return c.SendStatus(fiber.StatusNoContent)
	})

	f.Get("/governance/policy-exceptions", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if err := requireGovernancePublish(c, d, principal); err != nil {
			return err
		}
		limit := clampLimit(c.Query("limit"), 200, 500)
		list, err := d.PolicyExceptions.List(c.Context(), limit)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(list)
	})

	f.Get("/governance/policy-exceptions/:id", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if err := requireGovernancePublish(c, d, principal); err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		p, err := d.PolicyExceptions.Get(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		return c.JSON(p)
	})

	f.Post("/governance/policy-exceptions", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if err := requireGovernancePublish(c, d, principal); err != nil {
			return err
		}
		var in governance.CreatePolicyExceptionInput
		if err := c.BodyParser(&in); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		p, err := d.PolicyExceptions.Create(c.Context(), principal, in)
		if err != nil {
			if errors.Is(err, governance.ErrReasonRequired) {
				return fiber.NewError(fiber.StatusBadRequest, err.Error())
			}
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("policy_exception.created", "user", &principal, "policy_override", &p.ID, nil))
		return c.Status(fiber.StatusCreated).JSON(p)
	})

	f.Post("/governance/policy-exceptions/:id/review", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if err := requireGovernancePublish(c, d, principal); err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		if err := d.PolicyExceptions.MarkReviewed(c.Context(), id, principal); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("policy_exception.reviewed", "user", &principal, "policy_override", &id, nil))
		return c.SendStatus(fiber.StatusNoContent)
	})

	f.Post("/governance/policy-exceptions/:id/revoke", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if err := requireGovernancePublish(c, d, principal); err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		if err := d.PolicyExceptions.Revoke(c.Context(), id, principal); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("policy_exception.revoked", "user", &principal, "policy_override", &id, nil))
		return c.SendStatus(fiber.StatusNoContent)
	})

	f.Get("/governance/missing-owners", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		doms, err := governancePublishDomainIDs(c, d, principal)
		if err != nil {
			return err
		}
		list, err := d.OwnerRemediation.ListMissingInDomains(c.Context(), doms)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(list)
	})

	f.Post("/governance/missing-owners/assign", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if err := requireGovernancePublish(c, d, principal); err != nil {
			return err
		}
		var body struct {
			ResourceType string    `json:"resource_type"`
			ResourceID   uuid.UUID `json:"resource_id"`
			OwnerID      uuid.UUID `json:"owner_id"`
		}
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		switch body.ResourceType {
		case "entity":
			if err := d.OwnerRemediation.AssignEntityOwner(c.Context(), body.ResourceID, body.OwnerID); err != nil {
				return fiber.NewError(fiber.StatusBadRequest, err.Error())
			}
		case "source_feed":
			if err := d.OwnerRemediation.AssignSourceFeedOwner(c.Context(), body.ResourceID, body.OwnerID); err != nil {
				return fiber.NewError(fiber.StatusBadRequest, err.Error())
			}
		case "knowledge_job":
			if err := d.OwnerRemediation.AssignJobOwner(c.Context(), body.ResourceID, body.OwnerID); err != nil {
				return fiber.NewError(fiber.StatusBadRequest, err.Error())
			}
		default:
			return fiber.NewError(fiber.StatusBadRequest, "unsupported resource_type")
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("owner.assigned", "user", &principal, body.ResourceType, &body.ResourceID, nil))
		return c.SendStatus(fiber.StatusNoContent)
	})

	f.Post("/answer-feedback", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		var body struct {
			TraceID      string  `json:"trace_id"`
			FeedbackKind string  `json:"feedback_kind"`
			Comment      *string `json:"comment"`
		}
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		if body.TraceID == "" || body.FeedbackKind == "" {
			return fiber.NewError(fiber.StatusBadRequest, "trace_id and feedback_kind required")
		}
		fb, err := d.AnswerFeedback.Submit(c.Context(), principal, body.TraceID, body.FeedbackKind, body.Comment)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.Status(fiber.StatusCreated).JSON(fb)
	})

	f.Get("/answer-feedback", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if err := requireGovernancePublish(c, d, principal); err != nil {
			return err
		}
		limit := clampLimit(c.Query("limit"), 100, 500)
		list, err := d.AnswerFeedback.ListRecent(c.Context(), limit)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(list)
	})

	f.Post("/entities/:id/promote-canonical", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		ent, err := d.Entities.Get(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		domainID := ent.DomainID
		sens := ent.SensitivityLevel
		et := ent.Type
		dec, err := d.Access.Evaluate(c.Context(), identity_access.EvaluateInput{
			PrincipalID:      principal,
			Action:           "publish",
			ResourceType:     "entity",
			ResourceID:       &id,
			DomainID:         &domainID,
			SensitivityLevel: &sens,
			EntityType:       &et,
		})
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		if !dec.Allow || !dec.SensitivityOK {
			return fiber.NewError(fiber.StatusForbidden, "access denied")
		}
		var body struct {
			ChangeSummary *string `json:"change_summary"`
		}
		_ = c.BodyParser(&body)
		summary := "promote_to_canonical_platform"
		if body.ChangeSummary != nil && *body.ChangeSummary != "" {
			summary = *body.ChangeSummary
		}
		out, err := d.Entities.PromoteDerivedToCanonicalPlatform(c.Context(), id, principal, summary)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("entity.promoted_canonical", "user", &principal, "entity", &id, nil))
		return c.JSON(out)
	})

	// --- Extracted meeting tasks (Second Brain) + overlay (chat links, metrics) ---
	f.Get("/domains/:domainId/extracted-meeting-tasks", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		domainID, err := uuid.Parse(c.Params("domainId"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid domain id")
		}
		granted, err := d.Access.DomainIDsWithGrant(c.Context(), principal)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		if !domainInGrants(granted, domainID) {
			return fiber.NewError(fiber.StatusForbidden, "access denied")
		}
		st := strings.TrimSpace(c.Query("review_status"))
		var stPtr *string
		if st != "" {
			stPtr = &st
		}
		limit := clampLimit(c.Query("limit"), 50, 200)
		list, err := d.ExtractedMeetingTasks.ListByDomain(c.Context(), domainID, stPtr, limit)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(list)
	})

	f.Post("/domains/:domainId/extracted-meeting-tasks", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		domainID, err := uuid.Parse(c.Params("domainId"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid domain id")
		}
		granted, err := d.Access.DomainIDsWithGrant(c.Context(), principal)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		if !domainInGrants(granted, domainID) {
			return fiber.NewError(fiber.StatusForbidden, "access denied")
		}
		var body struct {
			Title                    string      `json:"title"`
			Description              string      `json:"description"`
			SourceFeedID             *uuid.UUID  `json:"source_feed_id"`
			SourceNormalizedRecordID *uuid.UUID  `json:"source_normalized_record_id"`
			LinkedMeetingEntityID    *uuid.UUID  `json:"linked_meeting_entity_id"`
			LinkedDecisionEntityIDs  []uuid.UUID `json:"linked_decision_entity_ids"`
			ParticipantRefs          []string    `json:"participant_refs"`
			AssigneeEmail            *string     `json:"assignee_email"`
			AssigneeDisplay          *string     `json:"assignee_display"`
			DeadlineDate             *string     `json:"deadline_date"`
			Priority                 string      `json:"priority"`
		}
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		if strings.TrimSpace(body.Title) == "" {
			return fiber.NewError(fiber.StatusBadRequest, "title required")
		}
		var deadline *time.Time
		if body.DeadlineDate != nil && strings.TrimSpace(*body.DeadlineDate) != "" {
			t, perr := time.Parse("2006-01-02", strings.TrimSpace(*body.DeadlineDate))
			if perr != nil {
				return fiber.NewError(fiber.StatusBadRequest, "deadline_date must be YYYY-MM-DD")
			}
			deadline = &t
		}
		t, err := d.ExtractedMeetingTasks.Create(c.Context(), extracted_meeting_tasks.CreateInput{
			DomainID:                 domainID,
			SourceFeedID:             body.SourceFeedID,
			SourceNormalizedRecordID: body.SourceNormalizedRecordID,
			LinkedMeetingEntityID:    body.LinkedMeetingEntityID,
			LinkedDecisionEntityIDs:  body.LinkedDecisionEntityIDs,
			ParticipantRefs:          body.ParticipantRefs,
			Title:                    strings.TrimSpace(body.Title),
			Description:              strings.TrimSpace(body.Description),
			AssigneeEmail:            body.AssigneeEmail,
			AssigneeDisplay:          body.AssigneeDisplay,
			DeadlineDate:             deadline,
			Priority:                 body.Priority,
			ActorUserID:              principal,
		})
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		return c.Status(fiber.StatusCreated).JSON(t)
	})

	f.Get("/domains/:domainId/second-brain-metrics", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		domainID, err := uuid.Parse(c.Params("domainId"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid domain id")
		}
		granted, err := d.Access.DomainIDsWithGrant(c.Context(), principal)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		if !domainInGrants(granted, domainID) {
			return fiber.NewError(fiber.StatusForbidden, "access denied")
		}
		m, err := d.ExtractedMeetingTasks.MetricsForDomain(c.Context(), domainID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(m)
	})

	f.Get("/domains/:domainId/second-brain-product-events", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		domainID, err := uuid.Parse(c.Params("domainId"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid domain id")
		}
		granted, err := d.Access.DomainIDsWithGrant(c.Context(), principal)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		if !domainInGrants(granted, domainID) {
			return fiber.NewError(fiber.StatusForbidden, "access denied")
		}
		limit := clampLimit(c.Query("limit"), 50, 200)
		list, err := secondbrain.ListProductEventsForDomain(c.Context(), d.Pool, domainID, limit)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(list)
	})

	f.Get("/extracted-meeting-tasks/:id", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		t, err := d.ExtractedMeetingTasks.Get(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		if t == nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		granted, err := d.Access.DomainIDsWithGrant(c.Context(), principal)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		if !domainInGrants(granted, t.DomainID) {
			return fiber.NewError(fiber.StatusForbidden, "access denied")
		}
		return c.JSON(t)
	})

	f.Patch("/extracted-meeting-tasks/:id", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid id")
		}
		t0, err := d.ExtractedMeetingTasks.Get(c.Context(), id)
		if err != nil || t0 == nil {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		granted, err := d.Access.DomainIDsWithGrant(c.Context(), principal)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		if !domainInGrants(granted, t0.DomainID) {
			return fiber.NewError(fiber.StatusForbidden, "access denied")
		}
		var body struct {
			Title           *string `json:"title"`
			Description     *string `json:"description"`
			AssigneeEmail   *string `json:"assignee_email"`
			AssigneeDisplay *string `json:"assignee_display"`
			DeadlineDate    *string `json:"deadline_date"`
			Priority        *string `json:"priority"`
		}
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		var dd *time.Time
		if body.DeadlineDate != nil && strings.TrimSpace(*body.DeadlineDate) != "" {
			t, perr := time.Parse("2006-01-02", strings.TrimSpace(*body.DeadlineDate))
			if perr != nil {
				return fiber.NewError(fiber.StatusBadRequest, "deadline_date must be YYYY-MM-DD")
			}
			dd = &t
		}
		t, err := d.ExtractedMeetingTasks.PatchDraft(c.Context(), id, principal, extracted_meeting_tasks.PatchDraftInput{
			Title: body.Title, Description: body.Description, AssigneeEmail: body.AssigneeEmail,
			AssigneeDisplay: body.AssigneeDisplay, DeadlineDate: dd, Priority: body.Priority,
		})
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		return c.JSON(t)
	})

	f.Post("/extracted-meeting-tasks/:id/confirm-no-edit", func(c *fiber.Ctx) error {
		return extractedTaskConfirm(c, d, "no-edit")
	})
	f.Post("/extracted-meeting-tasks/:id/confirm-after-edit", func(c *fiber.Ctx) error {
		return extractedTaskConfirm(c, d, "after-edit")
	})
	f.Post("/extracted-meeting-tasks/:id/reject", func(c *fiber.Ctx) error {
		return extractedTaskConfirm(c, d, "reject")
	})

	f.Get("/me/chat-links", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		links, err := secondbrain.GetChatLinks(c.Context(), d.Pool, principal)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(links)
	})

	f.Put("/me/chat-links", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		var body struct {
			TelegramChatID   *string `json:"telegram_chat_id"`
			MattermostUserID *string `json:"mattermost_user_id"`
		}
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		links, err := secondbrain.UpsertChatLinks(c.Context(), d.Pool, principal, body.TelegramChatID, body.MattermostUserID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(links)
	})

	f.Get("/search", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		scenarioCode := strings.TrimSpace(c.Query("scenario_code"))
		if ok, err := d.RoleBuilder.Assignments.PrincipalAllowsScenario(c.Context(), principal, scenarioCode); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		} else if !ok {
			return fiber.NewError(fiber.StatusForbidden, "scenario not permitted for this principal")
		}
		filters := map[string]string{
			"q":                c.Query("q"),
			"type":             c.Query("type"),
			"domain_id":        c.Query("domain_id"),
			"owner_id":         c.Query("owner_id"),
			"truth_mode":       c.Query("truth_mode"),
			"lifecycle_state":  c.Query("lifecycle_state"),
			"freshness_status": c.Query("freshness_status"),
			"approval_status":  c.Query("approval_status"),
			"expand_relations": c.Query("expand_relations"),
		}
		hits, err := d.Retrieval.SearchScoped(c.Context(), principal, filters)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		d.Retrieval.LogSearchInteraction(c.Context(), principal, filters, len(hits))
		return c.JSON(fiber.Map{"hits": hits})
	})

}

func containsUUID(list []uuid.UUID, id uuid.UUID) bool {
	for _, v := range list {
		if v == id {
			return true
		}
	}
	return false
}
