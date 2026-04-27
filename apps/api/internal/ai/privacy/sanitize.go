package privacy

import (
	"context"
	"encoding/json"
	"strings"
)

// RedactionReport summarizes sanitization without leaking raw values.
type RedactionReport struct {
	CountsByType        map[string]int `json:"counts_by_type"`
	PlaceholderCount    int            `json:"placeholder_count"`
	RemovedSpanCount    int            `json:"removed_span_count"`
	KeptSpanCount       int            `json:"kept_span_count"`
	DisallowAITriggered bool           `json:"disallow_ai_triggered"`
}

// SanitizationService runs structured → pattern → NER detection order per segment.
type SanitizationService struct {
	NER EntityExtractor
}

// NewSanitizationService creates a sanitizer with optional NER hook.
func NewSanitizationService(ner EntityExtractor) *SanitizationService {
	if ner == nil {
		ner = NoopExtractor{}
	}
	return &SanitizationService{NER: ner}
}

// SanitizeSegments joins segments with a separator after sanitizing each piece.
func (s *SanitizationService) SanitizeSegments(ctx context.Context, ep *EffectivePolicy, segments []TextSegment, tok *PlaceholderTokenizer, sep string) (string, *RedactionReport, error) {
	if ep == nil {
		ep = defaultEffectivePolicy()
	}
	if tok == nil {
		tok = NewPlaceholderTokenizer()
	}
	if s == nil {
		s = NewSanitizationService(nil)
	}
	report := &RedactionReport{CountsByType: map[string]int{}}
	var parts []string
	for _, seg := range segments {
		t := strings.TrimSpace(seg.Text)
		if t == "" {
			continue
		}
		out, err := s.sanitizeOne(ctx, ep, seg.Field, t, tok, report, seg.SkipPatternDetection)
		if err != nil {
			return "", report, err
		}
		if seg.Source != "" {
			parts = append(parts, "["+seg.Source+"]\n"+out)
		} else {
			parts = append(parts, out)
		}
	}
	return strings.Join(parts, sep), report, nil
}

func (s *SanitizationService) sanitizeOne(ctx context.Context, ep *EffectivePolicy, field, text string, tok *PlaceholderTokenizer, report *RedactionReport, skipPatterns bool) (string, error) {
	var patterns []detectedSpan
	if !skipPatterns {
		patterns = findPatternSpans(text)
	}
	var structured SensitiveEntityType
	if st, ok := StructuredTypeForField(field); ok {
		structured = st
	}
	spans := addStructuredGapSpans(text, structured, patterns)

	nerSpans, err := s.NER.Extract(ctx, text)
	if err != nil {
		return "", err
	}
	spans = appendNERSpans(text, spans, spansToDetected(text, nerSpans))

	var b strings.Builder
	last := 0
	for _, sp := range spans {
		if sp.Start < last {
			continue
		}
		if last < sp.Start {
			b.WriteString(text[last:sp.Start])
		}
		act := ep.ActionFor(sp.Type)
		report.CountsByType[string(sp.Type)]++
		switch act {
		case ActionDisallowAI:
			report.DisallowAITriggered = true
			return "", ErrDisallowAI
		case ActionRemove:
			report.RemovedSpanCount++
		case ActionKeep:
			b.WriteString(text[sp.Start:sp.End])
			report.KeptSpanCount++
		case ActionTokenize:
			ph := tok.Placeholder(sp.Type, text[sp.Start:sp.End])
			b.WriteString(ph)
			report.PlaceholderCount++
		default:
			ph := tok.Placeholder(sp.Type, text[sp.Start:sp.End])
			b.WriteString(ph)
			report.PlaceholderCount++
		}
		last = sp.End
	}
	if last < len(text) {
		b.WriteString(text[last:])
	}
	return b.String(), nil
}

// RedactionReportJSON returns JSON for traces (no raw secrets).
func RedactionReportJSON(r *RedactionReport) json.RawMessage {
	if r == nil {
		return []byte("{}")
	}
	b, _ := json.Marshal(r)
	return b
}
