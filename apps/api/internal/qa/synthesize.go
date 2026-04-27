package qa

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
	"github.com/knowledgelayer/api/internal/ai/privacy"
	"github.com/knowledgelayer/api/internal/ai/prompts"
	"github.com/knowledgelayer/api/internal/knowledge_core"
	"github.com/knowledgelayer/api/internal/llm"
)

type modelOut struct {
	Answer    string `json:"answer"`
	Citations []struct {
		EntityID string `json:"entity_id"`
		ChunkID  string `json:"chunk_id,omitempty"`
		Quote    string `json:"quote"`
	} `json:"citations"`
}

func maxEvidenceSensitivity(evidence []*knowledge_core.Entity) int {
	m := 0
	for _, e := range evidence {
		if e != nil && e.SensitivityLevel > m {
			m = e.SensitivityLevel
		}
	}
	return m
}

func scenarioFromScope(scope map[string]any) string {
	if v, ok := scope["ask_mode"].(string); ok && v == "entity" {
		return "ask_entity"
	}
	return "ask_global"
}

// buildSystemPrompt looks up the registered prompt for the requested answer
// strategy. The prompt id is returned alongside the system text so callers
// can record it in the privacy trace (Phase 4.1.1 audit-trail).
//
// If the registry returns an unexpected error (e.g. a templates/ file got
// renamed without bumping the call site), we log loudly and fall back to an
// empty system prompt — never silently downgrading to an inline copy that
// would diverge from the registered version.
func buildSystemPrompt(in AskEntityInput) (system string, promptID string) {
	id := "ask_global_qa.v1"
	if strings.EqualFold(strings.TrimSpace(in.AnswerStrategy), "best_trusted") {
		id = "ask_global_qa_best_trusted.v1"
	}
	p, err := prompts.Get(id)
	if err != nil {
		log.Printf("qa: prompt registry lookup failed for %q: %v", id, err)
		return "", id
	}
	return p.SystemPrompt, p.ID
}

func parseModelOutput(raw string, evidenceMap map[string]SupportingEntity) ([]Citation, string, error) {
	var mo modelOut
	if err := json.Unmarshal([]byte(raw), &mo); err != nil {
		return nil, "", fmt.Errorf("ask: model output not json: %w", err)
	}
	if strings.TrimSpace(mo.Answer) == "" {
		return nil, "", errors.New("ask: empty answer")
	}
	var citations []Citation
	for _, c := range mo.Citations {
		id, err := uuid.Parse(c.EntityID)
		if err != nil {
			continue
		}
		if _, ok := evidenceMap[id.String()]; !ok {
			continue
		}
		var chunkID uuid.UUID
		if strings.TrimSpace(c.ChunkID) != "" {
			if parsed, err := uuid.Parse(strings.TrimSpace(c.ChunkID)); err == nil {
				chunkID = parsed
			}
		}
		quote := strings.TrimSpace(c.Quote)
		if len(quote) > 300 {
			quote = quote[:300]
		}
		citations = append(citations, Citation{EntityID: id, ChunkID: chunkID, Quote: quote})
	}
	return citations, strings.TrimSpace(mo.Answer), nil
}

// synthesizeFromEvidence runs the governed LLM step for ordered entity evidence.
func (s *Service) synthesizeFromEvidence(ctx context.Context, principal uuid.UUID, evidence []*knowledge_core.Entity, q string, in AskEntityInput, scope map[string]any) (*AskEntityOutput, error) {
	if len(evidence) == 0 {
		return nil, errors.New("no evidence entities")
	}
	traceID := uuid.New()

	evidenceMap := map[string]SupportingEntity{}
	for _, e := range evidence {
		se := SupportingEntity{
			EntityID:        e.ID,
			Title:           e.Title,
			DomainID:        e.DomainID,
			EntityType:      e.Type,
			TruthMode:       e.TruthMode,
			LifecycleState:  e.LifecycleState,
			FreshnessStatus: e.FreshnessStatus,
		}
		evidenceMap[e.ID.String()] = se
	}

	system, promptID := buildSystemPrompt(in)
	segs := BuildPrivacySegmentsFromEvidence(q, evidence, 2000)

	extras, err := askImagesToLLMParts(in.Images)
	if err != nil {
		return nil, err
	}

	client, err := llm.NewOpenAIFromEnv()
	if err != nil {
		return nil, err
	}

	domainID := evidence[0].DomainID
	policyCtx := privacy.PolicyContext{
		DomainID:   &domainID,
		Scenario:   scenarioFromScope(scope),
		OutputType: "qa_answer",
	}

	var raw string
	var privTrace json.RawMessage
	if s.privacy != nil {
		inv := privacy.InvokeInput{
			System:            system,
			Segments:          segs,
			PolicyCtx:         policyCtx,
			PromptTemplateID:  promptID,
			CorrelationID:     traceID.String(),
			Principal:         principal,
			RehydrationMode:   privacy.RehydrationPartial,
			EvidenceEntities:  evidence,
			PublicationMode:   "reviewed_publish",
			OutputSensitivity: maxEvidenceSensitivity(evidence),
			LLMUserExtras:     extras,
		}
		res, err := s.privacy.InvokeOpenAI(ctx, client, inv)
		if err != nil {
			return nil, err
		}
		raw = res.Answer
		privTrace = res.PrivacyTraceJSON
	} else {
		var parts []string
		for _, seg := range segs {
			parts = append(parts, seg.Text)
		}
		user := "QUESTION_AND_EVIDENCE:\n\n" + strings.Join(parts, "\n\n---\n\n")
		merged, merr := llm.MergeTextAndMultimodalExtras(user, extras)
		if merr != nil {
			return nil, merr
		}
		if mx, ok := any(client).(llm.MetaUserContentCaller); ok {
			raw, _, err = mx.ChatScenarioWithMetaUserContent(ctx, policyCtx.Scenario, system, merged)
		} else {
			if _, multi := merged.([]map[string]any); multi {
				return nil, errors.New("qa: multimodal images require *llm.OpenAIClient")
			}
			raw, err = client.ChatScenario(ctx, policyCtx.Scenario, system, merged.(string))
		}
		if err != nil {
			return nil, err
		}
	}

	citations, answer, err := parseModelOutput(raw, evidenceMap)
	if err != nil {
		return nil, err
	}

	var supporting []SupportingEntity
	for _, e := range evidence {
		supporting = append(supporting, evidenceMap[e.ID.String()])
	}

	outScope := map[string]any{
		"principal_id":    principal.String(),
		"answer_strategy": strings.TrimSpace(in.AnswerStrategy),
	}
	for k, v := range scope {
		outScope[k] = v
	}

	return &AskEntityOutput{
		TraceID:            traceID.String(),
		Answer:             answer,
		Citations:          citations,
		SupportingEntities: supporting,
		Scope:              outScope,
		PrivacyTraceJSON:   privTrace,
	}, nil
}

// synthesizeFromContextPieces runs governed Q&A using ordered chunk or entity snippets (retrieval layer).
func (s *Service) synthesizeFromContextPieces(ctx context.Context, principal uuid.UUID, evidence []*knowledge_core.Entity, pieces []ContextPiece, q string, in AskEntityInput, scope map[string]any) (*AskEntityOutput, error) {
	if len(evidence) == 0 || len(pieces) == 0 {
		return nil, errors.New("no evidence for context pieces")
	}
	traceID := uuid.New()

	evidenceMap := map[string]SupportingEntity{}
	for _, e := range evidence {
		se := SupportingEntity{
			EntityID:        e.ID,
			Title:           e.Title,
			DomainID:        e.DomainID,
			EntityType:      e.Type,
			TruthMode:       e.TruthMode,
			LifecycleState:  e.LifecycleState,
			FreshnessStatus: e.FreshnessStatus,
		}
		evidenceMap[e.ID.String()] = se
	}

	system, promptID := buildSystemPrompt(in)
	segs := BuildPrivacySegmentsFromContextPieces(q, evidence, pieces, 2000)
	if len(segs) <= 1 {
		return nil, errors.New("no blocks built from context pieces")
	}

	extras, err := askImagesToLLMParts(in.Images)
	if err != nil {
		return nil, err
	}

	client, err := llm.NewOpenAIFromEnv()
	if err != nil {
		return nil, err
	}

	domainID := evidence[0].DomainID
	policyCtx := privacy.PolicyContext{
		DomainID:   &domainID,
		Scenario:   scenarioFromScope(scope),
		OutputType: "qa_answer",
	}

	var raw string
	var privTrace json.RawMessage
	if s.privacy != nil {
		inv := privacy.InvokeInput{
			System:            system,
			Segments:          segs,
			PolicyCtx:         policyCtx,
			PromptTemplateID:  promptID,
			CorrelationID:     traceID.String(),
			Principal:         principal,
			RehydrationMode:   privacy.RehydrationPartial,
			EvidenceEntities:  evidence,
			PublicationMode:   "reviewed_publish",
			OutputSensitivity: maxEvidenceSensitivity(evidence),
			LLMUserExtras:     extras,
		}
		res, err := s.privacy.InvokeOpenAI(ctx, client, inv)
		if err != nil {
			return nil, err
		}
		raw = res.Answer
		privTrace = res.PrivacyTraceJSON
	} else {
		var parts []string
		for _, seg := range segs {
			parts = append(parts, seg.Text)
		}
		user := "QUESTION_AND_EVIDENCE:\n\n" + strings.Join(parts, "\n\n---\n\n")
		merged, merr := llm.MergeTextAndMultimodalExtras(user, extras)
		if merr != nil {
			return nil, merr
		}
		if mx, ok := any(client).(llm.MetaUserContentCaller); ok {
			raw, _, err = mx.ChatScenarioWithMetaUserContent(ctx, policyCtx.Scenario, system, merged)
		} else {
			if _, multi := merged.([]map[string]any); multi {
				return nil, errors.New("qa: multimodal images require *llm.OpenAIClient")
			}
			raw, err = client.ChatScenario(ctx, policyCtx.Scenario, system, merged.(string))
		}
		if err != nil {
			return nil, err
		}
	}

	citations, answer, err := parseModelOutput(raw, evidenceMap)
	if err != nil {
		return nil, err
	}

	var supporting []SupportingEntity
	for _, e := range evidence {
		supporting = append(supporting, evidenceMap[e.ID.String()])
	}

	outScope := map[string]any{
		"principal_id":    principal.String(),
		"answer_strategy": strings.TrimSpace(in.AnswerStrategy),
	}
	for k, v := range scope {
		outScope[k] = v
	}

	return &AskEntityOutput{
		TraceID:            traceID.String(),
		Answer:             answer,
		Citations:          citations,
		SupportingEntities: supporting,
		Scope:              outScope,
		PrivacyTraceJSON:   privTrace,
	}, nil
}
