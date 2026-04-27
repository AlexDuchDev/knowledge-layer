package llm

import (
	"encoding/json"
	"testing"
)

func TestMergeTextAndMultimodalExtras_plainText(t *testing.T) {
	out, err := MergeTextAndMultimodalExtras("hello", nil)
	if err != nil {
		t.Fatal(err)
	}
	if s, ok := out.(string); !ok || s != "hello" {
		t.Fatalf("got %v (%T)", out, out)
	}
}

func TestMergeTextAndMultimodalExtras_withImageParts(t *testing.T) {
	extras := []map[string]any{
		{"type": "image_url", "image_url": map[string]any{"url": "https://example.com/x.png", "detail": "low"}},
	}
	out, err := MergeTextAndMultimodalExtras("evidence here", extras)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(map[string]any{"messages": []any{map[string]any{"role": "user", "content": out}}})
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(raw) {
		t.Fatalf("invalid json: %s", string(raw))
	}
}
