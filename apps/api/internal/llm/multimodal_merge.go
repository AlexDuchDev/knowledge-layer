package llm

import "strings"

// MergeTextAndMultimodalExtras builds the user message `content` field for chat/completions.
// When extras is empty, returns plain text (backward compatible). Otherwise returns an array of
// OpenAI/OpenRouter-style parts (text first, then image_url / input_audio, etc.).
func MergeTextAndMultimodalExtras(text string, extras []map[string]any) (any, error) {
	if len(extras) == 0 {
		return text, nil
	}
	t := strings.TrimSpace(text)
	if t == "" {
		t = "Use the evidence blocks and any attachments below."
	}
	parts := make([]map[string]any, 0, 1+len(extras))
	parts = append(parts, map[string]any{"type": "text", "text": t})
	parts = append(parts, extras...)
	return parts, nil
}
