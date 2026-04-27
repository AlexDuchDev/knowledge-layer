package privacy

import "context"

// Span is a byte-offset range in the original string (same convention as regexp indices).
type Span struct {
	Start, End int
	Type       SensitiveEntityType
}

// EntityExtractor is an optional NER / heuristic pass (lowest priority vs patterns).
type EntityExtractor interface {
	Extract(ctx context.Context, text string) ([]Span, error)
}

// NoopExtractor is the default extractor (no extra spans).
type NoopExtractor struct{}

// Extract implements EntityExtractor.
func (NoopExtractor) Extract(context.Context, string) ([]Span, error) {
	return nil, nil
}

func spansToDetected(text string, spans []Span) []detectedSpan {
	var out []detectedSpan
	for _, s := range spans {
		if s.Start < 0 || s.End > len(text) || s.Start >= s.End {
			continue
		}
		out = append(out, detectedSpan{
			Start: s.Start, End: s.End, Type: s.Type, Priority: prioNER,
			Value: text[s.Start:s.End],
		})
	}
	return out
}
