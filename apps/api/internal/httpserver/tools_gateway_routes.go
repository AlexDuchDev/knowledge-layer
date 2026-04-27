package httpserver

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/knowledgelayer/api/internal/answertrace"
	"github.com/knowledgelayer/api/internal/app"
	"github.com/knowledgelayer/api/internal/audit"
	"github.com/knowledgelayer/api/internal/httpcontext"
	"github.com/knowledgelayer/api/internal/identity_access"
	"github.com/knowledgelayer/api/internal/knowledge_core"
	"github.com/knowledgelayer/api/internal/qa"
	"github.com/knowledgelayer/api/internal/retrieval_intelligence"
)

type toolCallRequest struct {
	Tool string          `json:"tool"`
	Args json.RawMessage `json:"args"`
}

type toolCallError struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

type toolCallTrace struct {
	ToolCallID string    `json:"tool_call_id"`
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
	TraceRef   string    `json:"trace_ref"`
}

type toolCallResponse struct {
	Tool   string         `json:"tool"`
	OK     bool           `json:"ok"`
	Result any            `json:"result,omitempty"`
	Error  *toolCallError `json:"error,omitempty"`
	Trace  toolCallTrace  `json:"trace"`
}

func mountToolGatewayRoutes(f *fiber.App, d *app.Deps) {
	f.Post("/tools/call", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}

		var req toolCallRequest
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		tool := strings.TrimSpace(req.Tool)
		if tool == "" {
			return fiber.NewError(fiber.StatusBadRequest, "tool required")
		}

		callID := uuid.New().String()
		started := time.Now().UTC()
		traceRef := callID

		var (
			outOK           *bool
			outErrCode      *string
			outErrMsg       *string
			outTargetType   *string
			outTargetID     *uuid.UUID
			outArgsRedacted any
			outMetadata     any
		)

		_, _ = d.Pool.Exec(c.Context(), `
			INSERT INTO tool_calls (id, principal_id, tool_name, trace_ref, started_at)
			VALUES ($1,$2,$3,$4,$5)`,
			callID, principal, tool, traceRef, started)

		defer func() {
			finished := time.Now().UTC()

			argsJSON := []byte("{}")
			if outArgsRedacted != nil {
				if b, err := json.Marshal(outArgsRedacted); err == nil {
					argsJSON = b
				}
			}
			metaJSON := []byte("{}")
			if outMetadata != nil {
				if b, err := json.Marshal(outMetadata); err == nil {
					metaJSON = b
				}
			}

			_, _ = d.Pool.Exec(context.Background(), `
				UPDATE tool_calls
				SET finished_at=$2, ok=$3, error_code=$4, error_message=$5,
				    target_type=$6, target_id=$7,
				    args_redacted_json=$8, metadata_json=$9
				WHERE id=$1`,
				callID,
				finished,
				outOK,
				outErrCode,
				outErrMsg,
				outTargetType,
				outTargetID,
				argsJSON,
				metaJSON,
			)
		}()

		writeAudit := func(ctx context.Context, eventType string, targetType string, targetID *uuid.UUID, metadata any) {
			metaJSON := []byte("{}")
			if metadata != nil {
				if b, err := json.Marshal(metadata); err == nil {
					metaJSON = b
				}
			}
			tr := callID
			_ = d.AuditOps.Write(ctx, audit.WriteInput{
				EventType:    eventType,
				ActorType:    "user",
				ActorID:      &principal,
				TargetType:   targetType,
				TargetID:     targetID,
				TraceRef:     &tr,
				MetadataJSON: metaJSON,
			})
		}

		deny := func(status int, code, msg string, details map[string]any) error {
			finished := time.Now().UTC()
			okv := false
			outOK = &okv
			outErrCode = &code
			outErrMsg = &msg
			outMetadata = details
			resp := toolCallResponse{
				Tool: tool,
				OK:   false,
				Error: &toolCallError{
					Code:    code,
					Message: msg,
					Details: details,
				},
				Trace: toolCallTrace{ToolCallID: callID, StartedAt: started, FinishedAt: finished, TraceRef: callID},
			}
			return c.Status(status).JSON(resp)
		}

		ok := func(result any) error {
			finished := time.Now().UTC()
			okv := true
			outOK = &okv
			resp := toolCallResponse{
				Tool:   tool,
				OK:     true,
				Result: result,
				Trace:  toolCallTrace{ToolCallID: callID, StartedAt: started, FinishedAt: finished, TraceRef: callID},
			}
			return c.JSON(resp)
		}

		switch tool {
		case "search":
			var args struct {
				Query        string `json:"query"`
				Limit        int    `json:"limit"`
				Offset       int    `json:"offset"`
				DomainID     string `json:"domain_id"`
				EntityType   string `json:"entity_type"`
				ScenarioCode string `json:"scenario_code"`
			}
			if err := json.Unmarshal(req.Args, &args); err != nil {
				return fiber.NewError(fiber.StatusBadRequest, "invalid args")
			}
			q := strings.TrimSpace(args.Query)
			if q == "" {
				return fiber.NewError(fiber.StatusBadRequest, "query required")
			}
			outArgsRedacted = map[string]any{
				"q_len":         len(q),
				"domain_id":     strings.TrimSpace(args.DomainID),
				"entity_type":   strings.TrimSpace(args.EntityType),
				"scenario_code": strings.TrimSpace(args.ScenarioCode),
			}
			if okScenario, err := d.RoleBuilder.Assignments.PrincipalAllowsScenario(c.Context(), principal, strings.TrimSpace(args.ScenarioCode)); err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, err.Error())
			} else if !okScenario {
				writeAudit(c.Context(), "tool.search", "search", nil, map[string]any{"denied": true, "reason": "scenario_forbidden"})
				return deny(fiber.StatusForbidden, "forbidden", "scenario not permitted for this principal", map[string]any{"scenario_code": args.ScenarioCode})
			}
			filters := map[string]string{
				"q":         q,
				"domain_id": strings.TrimSpace(args.DomainID),
				"type":      strings.TrimSpace(args.EntityType),
			}
			hits, err := d.Search.Search(c.Context(), principal, filters)
			if err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, err.Error())
			}
			tt := "search"
			outTargetType = &tt
			writeAudit(c.Context(), "tool.search", "search", nil, map[string]any{"q_len": len(q), "hit_count": len(hits)})
			return ok(map[string]any{"hits": hits, "total": len(hits)})

		case "ask":
			var args struct {
				Question      string `json:"question"`
				RetrievalMode string `json:"retrieval_mode"`
				DomainID      string `json:"domain_id"`
				EntityType    string `json:"entity_type"`
				ScenarioCode  string `json:"scenario_code"`
			}
			if err := json.Unmarshal(req.Args, &args); err != nil {
				return fiber.NewError(fiber.StatusBadRequest, "invalid args")
			}
			question := strings.TrimSpace(args.Question)
			if question == "" {
				return fiber.NewError(fiber.StatusBadRequest, "question required")
			}
			outArgsRedacted = map[string]any{
				"question_len":   len(question),
				"domain_id":      strings.TrimSpace(args.DomainID),
				"entity_type":    strings.TrimSpace(args.EntityType),
				"retrieval_mode": strings.TrimSpace(args.RetrievalMode),
				"scenario_code":  strings.TrimSpace(args.ScenarioCode),
			}
			if okScenario, err := d.RoleBuilder.Assignments.PrincipalAllowsScenario(c.Context(), principal, strings.TrimSpace(args.ScenarioCode)); err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, err.Error())
			} else if !okScenario {
				writeAudit(c.Context(), "tool.ask", "qa", nil, map[string]any{"denied": true, "reason": "scenario_forbidden"})
				return deny(fiber.StatusForbidden, "forbidden", "scenario not permitted for this principal", map[string]any{"scenario_code": args.ScenarioCode})
			}
			filters := retrieval_intelligence.BuildGlobalAskSearchFilters(
				question,
				strings.TrimSpace(args.DomainID),
				strings.TrimSpace(args.EntityType),
				"", "", "", "",
			)
			askIn := qa.AskEntityInput{
				Question:       question,
				IncludeRelated: false,
				AnswerStrategy: "",
				ScenarioCode:   strings.TrimSpace(args.ScenarioCode),
			}
			out, gtrace, err := d.Retrieval.AskGlobal(
				c.Context(),
				principal,
				askIn,
				filters,
				strings.TrimSpace(args.RetrievalMode),
			)
			if err != nil {
				return fiber.NewError(fiber.StatusBadRequest, err.Error())
			}
			anchorID := uuid.Nil
			if aid, ok2 := out.Scope["anchor_entity_id"].(string); ok2 {
				if parsed, perr := uuid.Parse(aid); perr == nil {
					anchorID = parsed
				}
			}
			if anchorID == uuid.Nil && len(out.SupportingEntities) > 0 {
				anchorID = out.SupportingEntities[0].EntityID
			}
			tt := "entity"
			outTargetType = &tt
			outTargetID = &anchorID
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
					Question:               question,
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
			writeAudit(c.Context(), "tool.ask", "entity", &anchorID, map[string]any{"question_len": len(question), "trace_id": out.TraceID})
			return ok(map[string]any{
				"answer_markdown": out.Answer,
				"citations":       out.Citations,
				"answer_trace_id": out.TraceID,
			})

		case "entity.get":
			var args struct {
				EntityID string `json:"entity_id"`
			}
			if err := json.Unmarshal(req.Args, &args); err != nil {
				return fiber.NewError(fiber.StatusBadRequest, "invalid args")
			}
			id, err := uuid.Parse(strings.TrimSpace(args.EntityID))
			if err != nil {
				return fiber.NewError(fiber.StatusBadRequest, "invalid entity_id")
			}
			outArgsRedacted = map[string]any{"entity_id": id.String()}
			ent, err := d.Entities.Get(c.Context(), id)
			if err != nil {
				return fiber.NewError(fiber.StatusNotFound, "not found")
			}
			if err := requireEntityAction(c, d, principal, ent, "view"); err != nil {
				writeAudit(c.Context(), "tool.entity_get", "entity", &id, map[string]any{"denied": true})
				return deny(fiber.StatusForbidden, "forbidden", "access denied", map[string]any{"entity_id": id.String()})
			}
			tt := "entity"
			outTargetType = &tt
			outTargetID = &id
			writeAudit(c.Context(), "tool.entity_get", "entity", &id, nil)
			return ok(ent)

		case "entity.related":
			var args struct {
				EntityID string `json:"entity_id"`
				Depth    int    `json:"depth"`
			}
			if err := json.Unmarshal(req.Args, &args); err != nil {
				return fiber.NewError(fiber.StatusBadRequest, "invalid args")
			}
			id, err := uuid.Parse(strings.TrimSpace(args.EntityID))
			if err != nil {
				return fiber.NewError(fiber.StatusBadRequest, "invalid entity_id")
			}
			outArgsRedacted = map[string]any{"entity_id": id.String(), "depth": args.Depth}
			root, err := d.Entities.Get(c.Context(), id)
			if err != nil {
				return fiber.NewError(fiber.StatusNotFound, "not found")
			}
			if err := requireEntityAction(c, d, principal, root, "view"); err != nil {
				writeAudit(c.Context(), "tool.entity_related", "entity", &id, map[string]any{"denied": true})
				return deny(fiber.StatusForbidden, "forbidden", "access denied", map[string]any{"entity_id": id.String()})
			}
			limit := 12
			depth := 1
			if args.Depth == 2 {
				depth = 2
			}
			tt := "entity"
			outTargetType = &tt
			outTargetID = &id

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

			writeAudit(c.Context(), "tool.entity_related", "entity", &id, map[string]any{"depth": depth, "count": len(out)})
			return ok(map[string]any{"items": out})

		case "job.run":
			var args struct {
				JobID string `json:"job_id"`
			}
			if err := json.Unmarshal(req.Args, &args); err != nil {
				return fiber.NewError(fiber.StatusBadRequest, "invalid args")
			}
			jobID, err := uuid.Parse(strings.TrimSpace(args.JobID))
			if err != nil {
				return fiber.NewError(fiber.StatusBadRequest, "invalid job_id")
			}
			outArgsRedacted = map[string]any{"job_id": jobID.String()}
			j, err := d.Jobs.Get(c.Context(), jobID)
			if err != nil {
				return fiber.NewError(fiber.StatusNotFound, "not found")
			}
			tt := "knowledge_job"
			outTargetType = &tt
			outTargetID = &jobID
			if !principalMayRunKnowledgeJob(c.Context(), d, principal, j) {
				writeAudit(c.Context(), "tool.job_run", "knowledge_job", &jobID, map[string]any{"denied": true})
				return deny(fiber.StatusForbidden, "forbidden", "access denied", map[string]any{"job_id": jobID.String()})
			}
			run, err := d.Jobs.Run(c.Context(), jobID, principal)
			if err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, err.Error())
			}
			writeAudit(c.Context(), "tool.job_run", "knowledge_job", &jobID, map[string]any{"job_run_id": run.ID.String()})
			return ok(map[string]any{"job_run_id": run.ID.String(), "status": run.Status})

		case "job.status":
			var args struct {
				JobRunID string `json:"job_run_id"`
			}
			if err := json.Unmarshal(req.Args, &args); err != nil {
				return fiber.NewError(fiber.StatusBadRequest, "invalid args")
			}
			runID, err := uuid.Parse(strings.TrimSpace(args.JobRunID))
			if err != nil {
				return fiber.NewError(fiber.StatusBadRequest, "invalid job_run_id")
			}
			outArgsRedacted = map[string]any{"job_run_id": runID.String()}
			run, err := d.Jobs.GetRun(c.Context(), runID)
			if err != nil {
				return fiber.NewError(fiber.StatusNotFound, "not found")
			}
			j, err := d.Jobs.Get(c.Context(), run.KnowledgeJobID)
			if err != nil {
				return fiber.NewError(fiber.StatusNotFound, "not found")
			}
			if !principalCanViewKnowledgeJob(c.Context(), d, principal, j) {
				writeAudit(c.Context(), "tool.job_status", "knowledge_job_run", &runID, map[string]any{"denied": true})
				return deny(fiber.StatusForbidden, "forbidden", "access denied", map[string]any{"job_run_id": runID.String()})
			}
			tt := "knowledge_job_run"
			outTargetType = &tt
			outTargetID = &runID
			writeAudit(c.Context(), "tool.job_status", "knowledge_job_run", &runID, nil)
			return ok(run)

		case "sourceFeed.list":
			var args struct {
				DomainID string `json:"domain_id"`
				Limit    int    `json:"limit"`
			}
			if err := json.Unmarshal(req.Args, &args); err != nil {
				return fiber.NewError(fiber.StatusBadRequest, "invalid args")
			}
			granted, err := d.Access.DomainIDsWithGrant(c.Context(), principal)
			if err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, err.Error())
			}
			outArgsRedacted = map[string]any{"domain_id": strings.TrimSpace(args.DomainID), "limit": args.Limit}
			if did := strings.TrimSpace(args.DomainID); did != "" {
				want, perr := uuid.Parse(did)
				if perr != nil {
					return fiber.NewError(fiber.StatusBadRequest, "invalid domain_id")
				}
				okDom := false
				for _, g := range granted {
					if g == want {
						okDom = true
						break
					}
				}
				if !okDom {
					writeAudit(c.Context(), "tool.source_feed_list", "source_feed", nil, map[string]any{"denied": true})
					return deny(fiber.StatusForbidden, "forbidden", "domain not granted", map[string]any{"domain_id": want.String()})
				}
				granted = []uuid.UUID{want}
			}
			limit := args.Limit
			list, err := d.Ingestion.ListSourceFeedsInDomains(c.Context(), granted, limit)
			if err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, err.Error())
			}
			tt := "source_feed"
			outTargetType = &tt
			writeAudit(c.Context(), "tool.source_feed_list", "source_feed", nil, map[string]any{"count": len(list)})
			return ok(map[string]any{"source_feeds": list})

		case "sourceFeed.sync":
			var args struct {
				SourceFeedID string `json:"source_feed_id"`
			}
			if err := json.Unmarshal(req.Args, &args); err != nil {
				return fiber.NewError(fiber.StatusBadRequest, "invalid args")
			}
			id, err := uuid.Parse(strings.TrimSpace(args.SourceFeedID))
			if err != nil {
				return fiber.NewError(fiber.StatusBadRequest, "invalid source_feed_id")
			}
			outArgsRedacted = map[string]any{"source_feed_id": id.String()}
			sf, err := d.Ingestion.GetSourceFeed(c.Context(), id)
			if err != nil {
				return fiber.NewError(fiber.StatusNotFound, "not found")
			}
			tt := "source_feed"
			outTargetType = &tt
			outTargetID = &id
			if err := requireManageSourceFeed(c, d, principal, sf); err != nil {
				writeAudit(c.Context(), "tool.source_feed_sync", "source_feed", &id, map[string]any{"denied": true})
				return deny(fiber.StatusForbidden, "forbidden", "access denied", map[string]any{"source_feed_id": id.String()})
			}
			run, err := d.Ingestion.SyncSourceFeed(c.Context(), id)
			if err != nil {
				return fiber.NewError(fiber.StatusBadRequest, err.Error())
			}
			writeAudit(c.Context(), "tool.source_feed_sync", "source_feed", &id, map[string]any{"ingestion_run_id": run.ID.String()})
			return ok(run)

		case "rawArtifact.get":
			var args struct {
				RawArtifactID string `json:"raw_artifact_id"`
			}
			if err := json.Unmarshal(req.Args, &args); err != nil {
				return fiber.NewError(fiber.StatusBadRequest, "invalid args")
			}
			id, err := uuid.Parse(strings.TrimSpace(args.RawArtifactID))
			if err != nil {
				return fiber.NewError(fiber.StatusBadRequest, "invalid raw_artifact_id")
			}
			outArgsRedacted = map[string]any{"raw_artifact_id": id.String()}
			raw, err := d.Ingestion.GetRawArtifact(c.Context(), id)
			if err != nil {
				return fiber.NewError(fiber.StatusNotFound, "not found")
			}
			tt := "raw_artifact"
			outTargetType = &tt
			outTargetID = &id
			if err := requireViewRawArtifact(c, d, principal, raw.SourceFeedID); err != nil {
				writeAudit(c.Context(), "tool.raw_artifact_get", "raw_artifact", &id, map[string]any{"denied": true})
				return deny(fiber.StatusForbidden, "forbidden", "access denied", map[string]any{"raw_artifact_id": id.String()})
			}
			writeAudit(c.Context(), "tool.raw_artifact_get", "raw_artifact", &id, nil)
			return ok(raw)

		default:
			return fiber.NewError(fiber.StatusBadRequest, "unknown tool")
		}
	})
}

func requireViewRawArtifact(c *fiber.Ctx, d *app.Deps, principal uuid.UUID, sourceFeedID uuid.UUID) error {
	// SourceFeed lookup is required to find the governance domain; raw access is separately permissioned.
	sf, err := d.Ingestion.GetSourceFeed(c.Context(), sourceFeedID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "source feed not found")
	}
	dec, err := d.Access.Evaluate(c.Context(), identity_access.EvaluateInput{
		PrincipalID:  principal,
		Action:       "view_raw",
		ResourceType: "source_feed",
		ResourceID:   &sf.ID,
		DomainID:     &sf.DomainID,
	})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if !dec.Allow || !dec.SensitivityOK {
		return fiber.NewError(fiber.StatusForbidden, "access denied")
	}
	return nil
}
