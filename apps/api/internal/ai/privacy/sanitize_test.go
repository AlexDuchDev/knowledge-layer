package privacy

import (
	"context"
	"strings"
	"testing"
)

func TestSanitizeSegments_EmailTokenized(t *testing.T) {
	ep := &EffectivePolicy{
		ByEntityType: map[SensitiveEntityType]EffectiveRule{
			EntityEmail: {Action: ActionTokenize, RehydrationMode: RehydrationPartial},
		},
		EffectiveRehydration: RehydrationPartial,
	}
	tok := NewPlaceholderTokenizer()
	svc := NewSanitizationService(nil)
	segs := []TextSegment{{Field: "body", Text: "reach us at user@example.com please", Source: "t"}}
	out, rep, err := svc.SanitizeSegments(context.Background(), ep, segs, tok, "\n")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "user@example.com") {
		t.Fatalf("raw email leaked: %q", out)
	}
	if !strings.Contains(out, "EMAIL_1") {
		t.Fatalf("expected placeholder, got %q", out)
	}
	if rep.PlaceholderCount < 1 {
		t.Fatalf("report: %+v", rep)
	}
}

func TestSanitizeSegments_StablePlaceholders(t *testing.T) {
	ep := &EffectivePolicy{
		ByEntityType: map[SensitiveEntityType]EffectiveRule{
			EntityEmail: {Action: ActionTokenize, RehydrationMode: RehydrationPartial},
		},
		EffectiveRehydration: RehydrationPartial,
	}
	tok := NewPlaceholderTokenizer()
	svc := NewSanitizationService(nil)
	segs := []TextSegment{
		{Field: "body", Text: "a@b.co", Source: "a"},
		{Field: "body", Text: "a@b.co", Source: "b"},
	}
	out, _, err := svc.SanitizeSegments(context.Background(), ep, segs, tok, "\n")
	if err != nil {
		t.Fatal(err)
	}
	c := strings.Count(out, "EMAIL_1")
	if c != 2 {
		t.Fatalf("expected same placeholder twice, out=%q", out)
	}
}

func TestSanitizeSegments_StructuredTitleThenPattern(t *testing.T) {
	ep := &EffectivePolicy{
		ByEntityType: map[SensitiveEntityType]EffectiveRule{
			EntityCompanyName: {Action: ActionTokenize, RehydrationMode: RehydrationPartial},
			EntityEmail:       {Action: ActionTokenize, RehydrationMode: RehydrationPartial},
		},
		EffectiveRehydration: RehydrationPartial,
	}
	tok := NewPlaceholderTokenizer()
	svc := NewSanitizationService(nil)
	segs := []TextSegment{{Field: "title", Text: "Acme Corp contact x@y.co", Source: "t"}}
	out, _, err := svc.SanitizeSegments(context.Background(), ep, segs, tok, "\n")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "x@y.co") {
		t.Fatal("email not tokenized")
	}
	if !strings.Contains(out, "EMAIL_1") {
		t.Fatalf("out=%q", out)
	}
}

func TestSanitizeSegments_DisallowSecret(t *testing.T) {
	ep := &EffectivePolicy{
		ByEntityType: map[SensitiveEntityType]EffectiveRule{
			EntitySecuritySecret: {Action: ActionDisallowAI, RehydrationMode: RehydrationNone},
		},
		EffectiveRehydration: RehydrationNone,
	}
	tok := NewPlaceholderTokenizer()
	svc := NewSanitizationService(nil)
	segs := []TextSegment{{Field: "body", Text: "token sk-1234567890123456789012 here", Source: "t"}}
	_, _, err := svc.SanitizeSegments(context.Background(), ep, segs, tok, "\n")
	if err != ErrDisallowAI {
		t.Fatalf("got %v", err)
	}
}
