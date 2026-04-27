package qa

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/knowledgelayer/api/internal/ai/privacy"
	"github.com/knowledgelayer/api/internal/knowledge_core"
)

// AskImageAttachment optional image for vision-capable chat models (OpenAI / OpenRouter).
// Provide either URL (https:// or data:image/...) or raw base64 with media_type (e.g. image/png).
type AskImageAttachment struct {
	URL        string `json:"url,omitempty"`
	DataBase64 string `json:"data_base64,omitempty"`
	MediaType  string `json:"media_type,omitempty"`
	Detail     string `json:"detail,omitempty"` // auto | low | high
}

type AskEntityInput struct {
	Question       string `json:"question"`
	IncludeRelated bool   `json:"include_related"`
	AnswerStrategy string `json:"answer_strategy"` // standard | best_trusted
	// ScenarioCode optional; when non-empty, same PrincipalAllowsScenario gate as POST /ask and GET /search.
	ScenarioCode string `json:"scenario_code"`
	// Optional multimodal (sent to LLM after evidence text). Voice is transcribed in PreprocessAskMultimodal.
	Images      []AskImageAttachment `json:"images,omitempty"`
	AudioBase64 string               `json:"audio_base64,omitempty"`
	AudioFormat string               `json:"audio_format,omitempty"` // wav, mp3, m4a, ...
}

type SupportingEntity struct {
	EntityID        uuid.UUID `json:"entity_id"`
	Title           string    `json:"title"`
	DomainID        uuid.UUID `json:"domain_id"`
	EntityType      string    `json:"entity_type"`
	TruthMode       string    `json:"truth_mode"`
	LifecycleState  string    `json:"lifecycle_state"`
	FreshnessStatus string    `json:"freshness_status"`
}

type Citation struct {
	EntityID uuid.UUID `json:"entity_id"`
	ChunkID  uuid.UUID `json:"chunk_id,omitempty"`
	Quote    string    `json:"quote,omitempty"`
}

type AskEntityOutput struct {
	TraceID            string             `json:"trace_id"`
	Answer             string             `json:"answer"`
	Citations          []Citation         `json:"citations"`
	SupportingEntities []SupportingEntity `json:"supporting_entities"`
	Scope              map[string]any     `json:"scope"`
	EvidenceGraph      map[string]any     `json:"evidence_graph,omitempty"`
	PrivacyTraceJSON   json.RawMessage    `json:"privacy_trace_json,omitempty"`
}

// ContextPiece is ordered retrieval context (chunk text or empty to fall back to full entity body).
type ContextPiece struct {
	EntityID uuid.UUID
	ChunkID  uuid.UUID
	Text     string
}

type Service struct {
	entities *knowledge_core.EntityRepo
	privacy  *privacy.PrivacyGateway
}

func NewService(entities *knowledge_core.EntityRepo, gw *privacy.PrivacyGateway) *Service {
	return &Service{entities: entities, privacy: gw}
}

func (s *Service) AskEntity(ctx context.Context, principal uuid.UUID, entityID uuid.UUID, in AskEntityInput, canView func(*knowledge_core.Entity) error) (*AskEntityOutput, error) {
	q := strings.TrimSpace(in.Question)
	if q == "" && len(in.Images) == 0 {
		return nil, errors.New("question required (or attach images / audio_base64)")
	}
	if q == "" {
		q = "Answer using the attached image(s) and the evidence blocks below."
	}

	root, err := s.entities.Get(ctx, entityID)
	if err != nil {
		return nil, err
	}
	if err := canView(root); err != nil {
		return nil, err
	}

	var evidence []*knowledge_core.Entity
	evidence = append(evidence, root)

	if in.IncludeRelated {
		links, err := s.entities.ListLinks(ctx, entityID)
		if err != nil {
			return nil, err
		}
		seen := map[uuid.UUID]bool{entityID: true}
		for _, l := range links {
			if len(evidence) >= 8 {
				break
			}
			other := l.ToEntityID
			if other == entityID {
				other = l.FromEntityID
			}
			if seen[other] {
				continue
			}
			seen[other] = true
			e, err := s.entities.Get(ctx, other)
			if err != nil {
				continue
			}
			if err := canView(e); err != nil {
				continue
			}
			evidence = append(evidence, e)
		}
	}

	OrderEvidenceForAsk(entityID, evidence)

	return s.synthesizeFromEvidence(ctx, principal, evidence, q, in, map[string]any{
		"entity_id":       entityID.String(),
		"include_related": in.IncludeRelated,
		"ask_mode":        "entity",
	})
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func truncate(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max]
}
