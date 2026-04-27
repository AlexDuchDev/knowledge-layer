package httpserver

import (
	"context"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/knowledgelayer/api/internal/app"
	"github.com/knowledgelayer/api/internal/audit"
	"github.com/knowledgelayer/api/internal/extracted_meeting_tasks"
	"github.com/knowledgelayer/api/internal/httpcontext"
	"github.com/knowledgelayer/api/internal/identity_access"
	"github.com/knowledgelayer/api/internal/knowledge_core"
	"github.com/knowledgelayer/api/internal/knowledge_jobs"
	"github.com/knowledgelayer/api/internal/review"
)

// requireGovernancePublish allows governance center APIs when the principal may publish in at least one granted domain.
func requireGovernancePublish(c *fiber.Ctx, d *app.Deps, principal uuid.UUID) error {
	_, err := governancePublishDomainIDs(c, d, principal)
	if err != nil {
		return err
	}
	return nil
}

func governancePublishDomainIDs(c *fiber.Ctx, d *app.Deps, principal uuid.UUID) ([]uuid.UUID, error) {
	domainIDs, err := d.Access.DomainIDsWithGrant(c.Context(), principal)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	var publishDomains []uuid.UUID
	for _, dom := range domainIDs {
		did := dom
		dec, err := d.Access.Evaluate(c.Context(), identity_access.EvaluateInput{
			PrincipalID:  principal,
			Action:       "publish",
			ResourceType: "domain",
			DomainID:     &did,
		})
		if err != nil {
			return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		if dec.Allow && dec.SensitivityOK {
			publishDomains = append(publishDomains, did)
		}
	}
	if len(publishDomains) == 0 {
		return nil, fiber.NewError(fiber.StatusForbidden, "access denied")
	}
	return publishDomains, nil
}

func truncateStr(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func domainInGrants(granted []uuid.UUID, id uuid.UUID) bool {
	for _, g := range granted {
		if g == id {
			return true
		}
	}
	return false
}

// resolveIntegrationOnboardingDomain enforces domain grants and manage_source_feed for connector discovery (onboarding pickers).
func resolveIntegrationOnboardingDomain(c *fiber.Ctx, d *app.Deps, principal uuid.UUID, bodyDomainID uuid.UUID) (uuid.UUID, error) {
	granted, err := d.Access.DomainIDsWithGrant(c.Context(), principal)
	if err != nil {
		return uuid.Nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if len(granted) == 0 {
		return uuid.Nil, fiber.NewError(fiber.StatusForbidden, "no domain grant")
	}
	domainID := bodyDomainID
	if domainID == uuid.Nil {
		domainID = granted[0]
	} else if !domainInGrants(granted, domainID) {
		return uuid.Nil, fiber.NewError(fiber.StatusForbidden, "access denied")
	}
	if err := requireManageSourceFeedNewInDomain(c, d, principal, domainID, 1); err != nil {
		return uuid.Nil, err
	}
	return domainID, nil
}

func clampLimit(raw string, def int, max int) int {
	if def <= 0 {
		def = 200
	}
	if max <= 0 {
		max = 500
	}
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return def
	}
	if n > max {
		return max
	}
	return n
}

func entityViewOK(ctx context.Context, d *app.Deps, principal uuid.UUID, ent *knowledge_core.Entity) bool {
	domainID := ent.DomainID
	sens := ent.SensitivityLevel
	rid := ent.ID
	et := ent.Type
	dec, err := d.Access.Evaluate(ctx, identity_access.EvaluateInput{
		PrincipalID:      principal,
		Action:           "view",
		ResourceType:     "entity",
		ResourceID:       &rid,
		DomainID:         &domainID,
		SensitivityLevel: &sens,
		EntityType:       &et,
	})
	return err == nil && dec.Allow && dec.SensitivityOK
}

// FilterReviewTasksForPrincipal keeps review tasks whose entity targets pass Evaluate(view).
// Non-entity targets are omitted until a dedicated permission model exists for them.
func FilterReviewTasksForPrincipal(ctx context.Context, d *app.Deps, principal uuid.UUID, tasks []review.Task) []review.Task {
	out := make([]review.Task, 0, len(tasks))
	for _, t := range tasks {
		if t.TargetType != "entity" {
			continue
		}
		ent, err := d.Entities.Get(ctx, t.TargetID)
		if err != nil {
			continue
		}
		if entityViewOK(ctx, d, principal, ent) {
			out = append(out, t)
		}
	}
	return out
}

// requireReviewTaskEntityActions loads a review task and enforces each action on the target entity (entity targets only).
func requireReviewTaskEntityActions(c *fiber.Ctx, d *app.Deps, principal uuid.UUID, taskID uuid.UUID, actions []string) (*review.Task, error) {
	t, err := d.Review.Get(c.Context(), taskID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusNotFound, "not found")
	}
	if t.TargetType != "entity" {
		return nil, fiber.NewError(fiber.StatusForbidden, "unsupported review target")
	}
	ent, err := d.Entities.Get(c.Context(), t.TargetID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusNotFound, "not found")
	}
	for _, a := range actions {
		if err := requireEntityAction(c, d, principal, ent, a); err != nil {
			return nil, err
		}
	}
	return t, nil
}

func requireEntityAction(c *fiber.Ctx, d *app.Deps, principal uuid.UUID, ent *knowledge_core.Entity, action string) error {
	domainID := ent.DomainID
	sens := ent.SensitivityLevel
	rid := ent.ID
	et := ent.Type
	dec, err := d.Access.Evaluate(c.Context(), identity_access.EvaluateInput{
		PrincipalID:      principal,
		Action:           action,
		ResourceType:     "entity",
		ResourceID:       &rid,
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
	return nil
}

func requireViewRaw(c *fiber.Ctx, d *app.Deps, principal uuid.UUID, domainID uuid.UUID, sensitivity int, resourceID *uuid.UUID) error {
	dec, err := d.Access.Evaluate(c.Context(), identity_access.EvaluateInput{
		PrincipalID:      principal,
		Action:           "view_raw",
		ResourceType:     "raw_artifact",
		ResourceID:       resourceID,
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

func principalCanViewKnowledgeJob(ctx context.Context, d *app.Deps, principal uuid.UUID, job *knowledge_jobs.KnowledgeJob) bool {
	if job.OwnerID == principal {
		return true
	}
	if job.OutputDomainID == nil {
		return false
	}
	did := *job.OutputDomainID
	if ok, err := allowAction(ctx, d, principal, did, "view"); err == nil && ok {
		return true
	}
	if ok, err := allowAction(ctx, d, principal, did, "manage_jobs"); err == nil && ok {
		return true
	}
	return false
}

func principalCanManageKnowledgeJob(ctx context.Context, d *app.Deps, principal uuid.UUID, job *knowledge_jobs.KnowledgeJob) bool {
	if job.OwnerID == principal {
		return true
	}
	if job.OutputDomainID == nil {
		return false
	}
	ok, err := allowAction(ctx, d, principal, *job.OutputDomainID, "manage_jobs")
	return err == nil && ok
}

func requireCreateKnowledgeJobCapability(c *fiber.Ctx, d *app.Deps, principal uuid.UUID, in knowledge_jobs.CreateJobInput) error {
	if in.OutputDomainID != nil {
		ok, err := allowAction(c.Context(), d, principal, *in.OutputDomainID, "manage_jobs")
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		if !ok {
			return fiber.NewError(fiber.StatusForbidden, "manage_jobs required on output domain")
		}
		return nil
	}
	if in.OwnerID != principal {
		return fiber.NewError(fiber.StatusForbidden, "access denied")
	}
	return nil
}

func principalMayRunKnowledgeJob(ctx context.Context, d *app.Deps, principal uuid.UUID, job *knowledge_jobs.KnowledgeJob) bool {
	if !job.AllowDomainRunJob {
		if d.Jobs.OperatorCanRun(ctx, job, principal) {
			return true
		}
		return false
	}
	if d.Jobs.OperatorCanRun(ctx, job, principal) {
		return true
	}
	if job.OutputDomainID == nil {
		return false
	}
	did := *job.OutputDomainID
	if ok, err := allowAction(ctx, d, principal, did, "run_job"); err == nil && ok {
		return true
	}
	if ok, err := allowAction(ctx, d, principal, did, "manage_jobs"); err == nil && ok {
		return true
	}
	return false
}

// requireManageKnowledgeJobByID loads the job and ensures the principal may manage it (Job Builder / triggers).
func requireManageKnowledgeJobByID(c *fiber.Ctx, d *app.Deps, principal, jobID uuid.UUID) (*knowledge_jobs.KnowledgeJob, error) {
	j, err := d.Jobs.Get(c.Context(), jobID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusNotFound, "not found")
	}
	if !principalCanManageKnowledgeJob(c.Context(), d, principal, j) {
		return nil, fiber.NewError(fiber.StatusForbidden, "access denied")
	}
	return j, nil
}

func requireIngestionDerivedView(c *fiber.Ctx, d *app.Deps, principal uuid.UUID, domainID uuid.UUID, sensitivity int, resourceID *uuid.UUID) error {
	dec, err := d.Access.Evaluate(c.Context(), identity_access.EvaluateInput{
		PrincipalID:      principal,
		Action:           "view",
		ResourceType:     "normalized_record",
		ResourceID:       resourceID,
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

func auditInput(event, actorType string, actorID *uuid.UUID, targetType string, targetID *uuid.UUID, decision *string) audit.WriteInput {
	return audit.WriteInput{
		EventType:  event,
		ActorType:  actorType,
		ActorID:    actorID,
		TargetType: targetType,
		TargetID:   targetID,
		Decision:   decision,
	}
}

func ptr(s string) *string { return &s }

func extractedTaskConfirm(c *fiber.Ctx, d *app.Deps, mode string) error {
	principal, err := httpcontext.RequirePrincipal(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}
	t0, err := d.ExtractedMeetingTasks.Get(c.Context(), id)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if t0 == nil {
		return fiber.NewError(fiber.StatusNotFound, "not found")
	}
	granted, err := d.Access.DomainIDsWithGrant(c.Context(), principal)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if !domainInGrants(granted, t0.DomainID) {
		return fiber.NewError(fiber.StatusForbidden, "access denied")
	}
	var t *extracted_meeting_tasks.Task
	switch mode {
	case "no-edit":
		t, err = d.ExtractedMeetingTasks.ConfirmNoEdit(c.Context(), id, principal)
	case "after-edit":
		t, err = d.ExtractedMeetingTasks.ConfirmAfterEdit(c.Context(), id, principal)
	case "reject":
		t, err = d.ExtractedMeetingTasks.Reject(c.Context(), id, principal)
	default:
		return fiber.NewError(fiber.StatusInternalServerError, "invalid confirm mode")
	}
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return c.JSON(t)
}
