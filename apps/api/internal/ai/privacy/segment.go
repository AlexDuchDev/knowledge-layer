package privacy

import "github.com/google/uuid"

// TextSegment is one contiguous string with optional structured classification (field path).
type TextSegment struct {
	Source   string // logical source e.g. "entity:title" (optional; may be omitted for LLM prompts)
	EntityID uuid.UUID
	ChunkID  uuid.UUID
	Field    string // title | summary | body | chunk.text | question | entity_header | free_text
	Text     string
	// SkipPatternDetection when true skips regex/UUID patterns (e.g. ENTITY id lines must stay intact).
	SkipPatternDetection bool
}
