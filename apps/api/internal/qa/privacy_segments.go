package qa

import (
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/knowledgelayer/api/internal/ai/privacy"
	"github.com/knowledgelayer/api/internal/knowledge_core"
)

// BuildPrivacySegmentsFromQuestion wraps the user question as a segment.
func BuildPrivacySegmentsFromQuestion(q string) []privacy.TextSegment {
	return []privacy.TextSegment{
		{Field: "question", Text: strings.TrimSpace(q)},
	}
}

// BuildPrivacySegmentsFromEvidence builds segments mirroring governed Q&A evidence blocks.
// Entity UUID lines use SkipPatternDetection so UUID-based citation targets are not tokenized.
func BuildPrivacySegmentsFromEvidence(q string, evidence []*knowledge_core.Entity, contentMaxRunes int) []privacy.TextSegment {
	segs := BuildPrivacySegmentsFromQuestion(q)
	for _, e := range evidence {
		txt := strings.TrimSpace(strings.Join([]string{e.Title, deref(e.Summary), deref(e.Body)}, "\n\n"))
		if contentMaxRunes > 0 && len(txt) > contentMaxRunes {
			txt = txt[:contentMaxRunes]
		}
		header := fmt.Sprintf("ENTITY %s\nTYPE: %s\nDOMAIN: %s\nTRUST: truth=%s lifecycle=%s freshness=%s\nCONTENT:\n",
			e.ID.String(), e.Type, e.DomainID.String(), e.TruthMode, e.LifecycleState, e.FreshnessStatus)
		segs = append(segs, privacy.TextSegment{
			EntityID:             e.ID,
			Field:                "entity_header",
			Text:                 header,
			SkipPatternDetection: true,
		})
		if strings.TrimSpace(txt) != "" {
			segs = append(segs, privacy.TextSegment{
				EntityID: e.ID,
				Field:    "body",
				Text:     txt,
			})
		}
	}
	return segs
}

// BuildPrivacySegmentsFromContextPieces builds header + content for chunk or entity fallback text.
func BuildPrivacySegmentsFromContextPieces(q string, evidence []*knowledge_core.Entity, pieces []ContextPiece, contentMaxRunes int) []privacy.TextSegment {
	segs := BuildPrivacySegmentsFromQuestion(q)
	entityByID := map[uuid.UUID]*knowledge_core.Entity{}
	for _, e := range evidence {
		entityByID[e.ID] = e
	}
	for _, p := range pieces {
		e := entityByID[p.EntityID]
		if e == nil {
			continue
		}
		txt := strings.TrimSpace(p.Text)
		if txt == "" {
			txt = strings.TrimSpace(strings.Join([]string{e.Title, deref(e.Summary), deref(e.Body)}, "\n\n"))
		}
		if contentMaxRunes > 0 && len(txt) > contentMaxRunes {
			txt = txt[:contentMaxRunes]
		}
		chunkTag := ""
		if p.ChunkID != uuid.Nil {
			chunkTag = " CHUNK=" + p.ChunkID.String()
		}
		header := fmt.Sprintf("ENTITY %s%s\nTYPE: %s\nDOMAIN: %s\nTRUST: truth=%s lifecycle=%s freshness=%s\nCONTENT:\n",
			e.ID.String(), chunkTag, e.Type, e.DomainID.String(), e.TruthMode, e.LifecycleState, e.FreshnessStatus)
		segs = append(segs, privacy.TextSegment{
			EntityID:             p.EntityID,
			ChunkID:              p.ChunkID,
			Field:                "entity_header",
			Text:                 header,
			SkipPatternDetection: true,
		})
		if strings.TrimSpace(txt) != "" {
			field := "chunk.text"
			if p.ChunkID == uuid.Nil {
				field = "body"
			}
			segs = append(segs, privacy.TextSegment{
				EntityID: p.EntityID,
				ChunkID:  p.ChunkID,
				Field:    field,
				Text:     txt,
			})
		}
	}
	return segs
}
