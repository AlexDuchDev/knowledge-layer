package privacy

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/knowledgelayer/api/internal/audit"
	"github.com/knowledgelayer/api/internal/identity_access"
	"github.com/knowledgelayer/api/internal/knowledge_core"
	"github.com/knowledgelayer/api/internal/llm"
)

// ChatCompleter is satisfied by *llm.OpenAIClient (and test doubles).
type ChatCompleter interface {
	ChatScenario(ctx context.Context, scenario, system, user string) (string, error)
}

// PrivacyGateway is the single entry for governed LLM completions (sanitize → call → rehydrate).
type PrivacyGateway struct {
	policy   *SensitiveDataPolicyService
	sanitize *SanitizationService
	vault    *SecureEntityMapStore
	rehydr   *RehydrationService
	access   *identity_access.AccessEvaluator
}

// GatewayConfig wires dependencies (vault may be nil if encryption not configured).
// Audit is required for production-grade vault auditability; tests may pass nil.
type GatewayConfig struct {
	Policy   *SensitiveDataPolicyService
	Sanitize *SanitizationService
	Vault    *SecureEntityMapStore
	Access   *identity_access.AccessEvaluator
	Audit    *audit.Service
}

// NewPrivacyGateway constructs the gateway.
func NewPrivacyGateway(c GatewayConfig) *PrivacyGateway {
	var rehydr *RehydrationService
	if c.Vault != nil {
		rehydr = NewRehydrationService(c.Vault, c.Audit)
	} else {
		rehydr = NewRehydrationService(nil, c.Audit)
	}
	san := c.Sanitize
	if san == nil {
		san = NewSanitizationService(nil)
	}
	return &PrivacyGateway{
		policy:   c.Policy,
		sanitize: san,
		vault:    c.Vault,
		rehydr:   rehydr,
		access:   c.Access,
	}
}

// InvokeInput is one governed completion.
type InvokeInput struct {
	System    string
	Segments  []TextSegment
	PolicyCtx PolicyContext

	// PromptTemplateID is the registered prompt id (e.g. "ask_global_qa.v1")
	// the caller resolved via internal/ai/prompts. Recorded in the privacy
	// trace so audit / answer_traces can show which template version produced
	// a given answer (Phase 4.1.1). Optional — empty = legacy inline prompt.
	PromptTemplateID string

	CorrelationID string
	Principal     uuid.UUID
	JobRunID      *uuid.UUID

	RehydrationMode   RehydrationMode
	EvidenceEntities  []*knowledge_core.Entity
	PublicationMode   string
	OutputSensitivity int

	VaultTTL time.Duration

	// LLMUserExtras optional OpenAI-style user message parts (e.g. image_url), appended after sanitized text.
	LLMUserExtras []map[string]any
}

// InvokeResult is returned to callers (no raw vault values).
type InvokeResult struct {
	Answer            string
	Redaction         *RedactionReport
	Rehydrated        bool
	PrivacyTraceJSON  json.RawMessage
	SanitizedUserText string // for tests / internal diagnostics only; do not log externally
}

// InvokeOpenAI runs policy → sanitize → optional vault persist → Chat → rehydrate.
func (g *PrivacyGateway) InvokeOpenAI(ctx context.Context, client ChatCompleter, in InvokeInput) (*InvokeResult, error) {
	if client == nil {
		return nil, errors.New("privacy gateway: nil llm client")
	}
	var ep *EffectivePolicy
	var err error
	if g != nil && g.policy != nil {
		ep, err = g.policy.Resolve(ctx, in.PolicyCtx)
		if err != nil {
			return nil, err
		}
	} else {
		ep = defaultEffectivePolicy()
	}
	tok := NewPlaceholderTokenizer()
	san := g.sanitize
	if g == nil {
		return nil, errors.New("privacy gateway: nil")
	}
	if san == nil {
		san = NewSanitizationService(nil)
	}
	userSan, rep, err := san.SanitizeSegments(ctx, ep, in.Segments, tok, "\n\n---\n\n")
	if err != nil {
		return nil, err
	}
	if g.vault != nil && len(tok.Mappings()) > 0 {
		ttl := in.VaultTTL
		if ttl <= 0 {
			ttl = 48 * time.Hour
		}
		var entries []VaultEntry
		for _, m := range tok.Mappings() {
			val, _, ok := tok.RawValueByPlaceholder(m.Placeholder)
			if !ok {
				continue
			}
			entries = append(entries, VaultEntry{
				Placeholder:        m.Placeholder,
				EntityType:         m.EntityType,
				Value:              val,
				PolicySnapshotJSON: policySnapshotJSON(in.PolicyCtx, ep),
			})
		}
		pid := in.Principal
		if err := g.vault.PutBatch(ctx, in.CorrelationID, &pid, in.JobRunID, ttl, entries); err != nil {
			return nil, err
		}
	}
	raw := ""
	llmMeta := map[string]any{}
	merged, merr := llm.MergeTextAndMultimodalExtras(userSan, in.LLMUserExtras)
	if merr != nil {
		return nil, merr
	}
	if mx, ok := any(client).(llm.MetaUserContentCaller); ok {
		content, meta, cerr := mx.ChatScenarioWithMetaUserContent(ctx, strings.TrimSpace(in.PolicyCtx.Scenario), in.System, merged)
		if cerr != nil {
			return nil, cerr
		}
		raw = content
		llmMeta = meta
	} else {
		s, ok := merged.(string)
		if !ok {
			return nil, errors.New("privacy gateway: multimodal LLM extras require *llm.OpenAIClient (MetaUserContentCaller)")
		}
		if m, ok2 := any(client).(interface {
			ChatScenarioWithMeta(ctx context.Context, scenario, system, user string) (string, map[string]any, error)
		}); ok2 {
			content, meta, cerr := m.ChatScenarioWithMeta(ctx, strings.TrimSpace(in.PolicyCtx.Scenario), in.System, s)
			if cerr != nil {
				return nil, cerr
			}
			raw = content
			llmMeta = meta
		} else {
			raw, err = client.ChatScenario(ctx, strings.TrimSpace(in.PolicyCtx.Scenario), in.System, s)
			if err != nil {
				return nil, err
			}
		}
	}
	mode := in.RehydrationMode
	if mode == "" {
		mode = RehydrationPartial
	}
	out, changed, err := RehydrateFromTokenizer(raw, tok, ep, mode, g.access, ctx, in.Principal, in.EvidenceEntities, in.PublicationMode, in.OutputSensitivity)
	if err != nil {
		return nil, err
	}
	trace := map[string]any{
		"rehydration_mode":   string(mode),
		"rehydrated":         changed,
		"redaction":          rep,
		"llm":                llmMeta,
		"prompt_template_id": in.PromptTemplateID,
		"policy_context": map[string]any{
			"scenario":    in.PolicyCtx.Scenario,
			"output_type": in.PolicyCtx.OutputType,
			"job_type":    in.PolicyCtx.JobType,
		},
	}
	tb, _ := json.Marshal(trace)
	return &InvokeResult{
		Answer:            strings.TrimSpace(out),
		Redaction:         rep,
		Rehydrated:        changed,
		PrivacyTraceJSON:  tb,
		SanitizedUserText: userSan,
	}, nil
}

func policySnapshotJSON(pctx PolicyContext, ep *EffectivePolicy) json.RawMessage {
	m := map[string]any{
		"scenario":    pctx.Scenario,
		"output_type": pctx.OutputType,
		"job_type":    pctx.JobType,
	}
	if ep != nil {
		m["effective_rehydration"] = string(ep.EffectiveRehydration)
	}
	b, _ := json.Marshal(m)
	return b
}
