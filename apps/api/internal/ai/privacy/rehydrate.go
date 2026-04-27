package privacy

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"

	"github.com/knowledgelayer/api/internal/audit"
	"github.com/knowledgelayer/api/internal/identity_access"
	"github.com/knowledgelayer/api/internal/knowledge_core"
)

// RehydrationService restores placeholders using vault + policy + permissions.
// When wired with an audit.Service it emits a `vault.rehydration_applied`
// event for every RehydrateFromVault call so the principal-context for each
// rehydration is auditable (counterpart to vault.placeholder_decrypted, which
// only carries the correlation_id).
type RehydrationService struct {
	vault *SecureEntityMapStore
	audit *audit.Service
}

// NewRehydrationService constructs a rehydration service. auditSvc may be nil in tests.
func NewRehydrationService(vault *SecureEntityMapStore, auditSvc *audit.Service) *RehydrationService {
	return &RehydrationService{vault: vault, audit: auditSvc}
}

// RehydrationArgs controls governed replacement.
type RehydrationArgs struct {
	Text              string
	CorrelationID     string
	Principal         uuid.UUID
	Mode              RehydrationMode // output / job cap (e.g. none for broad distribution)
	EP                *EffectivePolicy
	Access            *identity_access.AccessEvaluator
	EvidenceEntities  []*knowledge_core.Entity // domains for view checks
	OutputSensitivity int                      // higher → stricter (>=8 forces none)
	PublicationMode   string                   // auto_publish / draft_only / reviewed_publish
}

// rehydrateAllowedForType returns whether we may restore this sensitive type under Mode and EP.
func rehydrateAllowedForType(ep *EffectivePolicy, typ SensitiveEntityType, mode RehydrationMode) bool {
	if mode == RehydrationNone {
		return false
	}
	rm := RehydrationNone
	if ep != nil {
		rm = ep.RehydrationFor(typ)
	}
	switch mode {
	case RehydrationFull:
		return rm == RehydrationFull
	case RehydrationPartial:
		return rm == RehydrationPartial || rm == RehydrationFull
	default:
		return false
	}
}

func outputPolicyForcesNone(pubMode string, sensitivity int) bool {
	if sensitivity >= 8 {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(pubMode)) {
	case "auto_publish", "autopublish":
		return true
	default:
		return false
	}
}

func (r *RehydrationService) viewerMayFullRehydrate(ctx context.Context, principal uuid.UUID, evidence []*knowledge_core.Entity, access *identity_access.AccessEvaluator) bool {
	if access == nil || len(evidence) == 0 {
		return true
	}
	for _, e := range evidence {
		if e == nil {
			continue
		}
		d := e.DomainID
		dec, err := access.Evaluate(ctx, identity_access.EvaluateInput{
			PrincipalID:      principal,
			Action:           "view",
			ResourceType:     "entity",
			ResourceID:       &e.ID,
			DomainID:         &d,
			EntityType:       &e.Type,
			SensitivityLevel: &e.SensitivityLevel,
		})
		if err != nil || dec == nil || !dec.Allow {
			return false
		}
	}
	return true
}

// RehydrateFromVault replaces placeholders in text when policy and permissions allow.
func (r *RehydrationService) RehydrateFromVault(ctx context.Context, args RehydrationArgs) (string, bool, error) {
	if r == nil || r.vault == nil {
		return args.Text, false, nil
	}
	mode := args.Mode
	if outputPolicyForcesNone(args.PublicationMode, args.OutputSensitivity) {
		mode = RehydrationNone
	}
	if mode == RehydrationNone {
		return args.Text, false, nil
	}
	maps, err := r.vault.ListDecryptedForCorrelation(ctx, args.CorrelationID)
	if err != nil {
		return "", false, err
	}
	out := args.Text
	changed := false
	allowFull := true
	if mode == RehydrationFull {
		allowFull = r.viewerMayFullRehydrate(ctx, args.Principal, args.EvidenceEntities, args.Access)
		if !allowFull {
			mode = RehydrationPartial
		}
	}
	for _, m := range maps {
		if !strings.Contains(out, m.Placeholder) {
			continue
		}
		if !rehydrateAllowedForType(args.EP, m.EntityType, mode) {
			continue
		}
		if mode == RehydrationFull && !allowFull {
			continue
		}
		out = strings.ReplaceAll(out, m.Placeholder, m.Value)
		changed = true
	}
	if r.audit != nil {
		meta, _ := json.Marshal(map[string]any{
			"correlation_id":     args.CorrelationID,
			"mode":               string(mode),
			"rehydrated":         changed,
			"output_sensitivity": args.OutputSensitivity,
			"publication_mode":   args.PublicationMode,
			"vault_entries":      len(maps),
		})
		pid := args.Principal
		_ = r.audit.Write(ctx, audit.WriteInput{
			EventType:    "vault.rehydration_applied",
			ActorType:    "system",
			ActorID:      &pid,
			TargetType:   "placeholder_mapping",
			MetadataJSON: meta,
		})
	}
	return out, changed, nil
}

// RehydrateFromTokenizer uses in-memory tokenizer (same request) with the same policy gates.
func RehydrateFromTokenizer(text string, tok *PlaceholderTokenizer, ep *EffectivePolicy, mode RehydrationMode, access *identity_access.AccessEvaluator, ctx context.Context, principal uuid.UUID, evidence []*knowledge_core.Entity, pubMode string, outSens int) (string, bool, error) {
	if tok == nil {
		return text, false, nil
	}
	if outputPolicyForcesNone(pubMode, outSens) {
		return text, false, nil
	}
	if mode == RehydrationNone {
		return text, false, nil
	}
	allowFull := true
	if mode == RehydrationFull {
		rs := NewRehydrationService(nil, nil)
		allowFull = rs.viewerMayFullRehydrate(ctx, principal, evidence, access)
		if !allowFull {
			mode = RehydrationPartial
		}
	}
	out := text
	changed := false
	for _, m := range tok.Mappings() {
		if !strings.Contains(out, m.Placeholder) {
			continue
		}
		val, typ, ok := tok.RawValueByPlaceholder(m.Placeholder)
		if !ok {
			continue
		}
		if !rehydrateAllowedForType(ep, typ, mode) {
			continue
		}
		out = strings.ReplaceAll(out, m.Placeholder, val)
		changed = true
	}
	return out, changed, nil
}
