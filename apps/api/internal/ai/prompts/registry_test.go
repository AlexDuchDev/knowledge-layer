package prompts

import (
	"strings"
	"testing"
)

// Sanity checks that the embedded templates load and the well-known reference
// prompt is callable. Adding a new template? Add a smoke test here that
// asserts the system_prompt mentions the rule it's supposed to enforce.

func TestGet_AskGlobalQA_V1_Loads(t *testing.T) {
	p, err := Get("ask_global_qa.v1")
	if err != nil {
		t.Fatalf("Get(ask_global_qa.v1): %v", err)
	}
	if p.ID != "ask_global_qa.v1" {
		t.Fatalf("ID drift: %q", p.ID)
	}
	mustContain(t, p.SystemPrompt, "governed Q&A assistant")
	mustContain(t, p.SystemPrompt, "Output MUST be valid JSON")
	mustContain(t, p.SystemPrompt, "ENTITY blocks as evidence")
}

func TestGet_AskGlobalQA_BestTrusted_V1_AddsTrustedClause(t *testing.T) {
	p, err := Get("ask_global_qa_best_trusted.v1")
	if err != nil {
		t.Fatalf("Get(ask_global_qa_best_trusted.v1): %v", err)
	}
	mustContain(t, p.SystemPrompt, "best_trusted")
	mustContain(t, p.SystemPrompt, "canonical_in_platform")
}

func TestGet_UnknownPrompt(t *testing.T) {
	_, err := Get("does_not_exist.v9")
	if err == nil {
		t.Fatal("expected error for unknown prompt id")
	}
	if !strings.Contains(err.Error(), "unknown id") {
		t.Fatalf("error message should hint at lookup failure: %v", err)
	}
}

func TestRender_LeavesUnknownPlaceholdersAsLiterals(t *testing.T) {
	p := Prompt{
		UserTemplate: "Question: {{q}}\nContext: {{ctx}}\nUnknown: {{leftover}}",
	}
	out := p.Render(map[string]string{
		"q":   "what",
		"ctx": "stuff",
	})
	mustContain(t, out, "Question: what")
	mustContain(t, out, "Context: stuff")
	// {{leftover}} stays — call site can grep for it in tests to assert no
	// placeholder was forgotten before sending to the LLM.
	mustContain(t, out, "{{leftover}}")
}

func TestIDs_IncludesEmbeddedTemplates(t *testing.T) {
	ids := IDs()
	want := map[string]bool{
		"ask_global_qa.v1":              false,
		"ask_global_qa_best_trusted.v1": false,
		"ai_summarize.v1":               false,
		"ai_draft_suggestions.v1":       false,
		"graphrag_entity_extract.v1":    false,
	}
	for _, id := range ids {
		if _, ok := want[id]; ok {
			want[id] = true
		}
	}
	for id, found := range want {
		if !found {
			t.Errorf("expected %q in IDs(), got %v", id, ids)
		}
	}
}

// Smoke tests for the templates extracted in Phase 4.1.1 follow-up. Adding a
// new template? Add a similar one-line load + key-phrase assertion here so a
// rename of the .json file fails CI loudly.
func TestGet_AISummarize_V1_Loads(t *testing.T) {
	p, err := Get("ai_summarize.v1")
	if err != nil {
		t.Fatalf("Get(ai_summarize.v1): %v", err)
	}
	mustContain(t, p.SystemPrompt, "summarize internal operational text")
}

func TestGet_AIDraftSuggestions_V1_Loads(t *testing.T) {
	p, err := Get("ai_draft_suggestions.v1")
	if err != nil {
		t.Fatalf("Get(ai_draft_suggestions.v1): %v", err)
	}
	mustContain(t, p.SystemPrompt, "valid Markdown only")
	mustContain(t, p.SystemPrompt, "you are not an authority")
}

func TestGet_GraphRAGEntityExtract_V1_Loads(t *testing.T) {
	p, err := Get("graphrag_entity_extract.v1")
	if err != nil {
		t.Fatalf("Get(graphrag_entity_extract.v1): %v", err)
	}
	mustContain(t, p.SystemPrompt, "GraphRAG system")
	mustContain(t, p.SystemPrompt, "Do NOT invent facts")
	// User template carries {{chunks}} — caller fills via Render().
	mustContain(t, p.UserTemplate, "{{chunks}}")
	rendered := p.Render(map[string]string{"chunks": "CHUNK_BODY"})
	mustContain(t, rendered, "CHUNK_BODY")
	if strings.Contains(rendered, "{{chunks}}") {
		t.Fatalf("Render() should substitute {{chunks}}, got %q", rendered)
	}
}

func mustContain(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Fatalf("expected %q to contain %q", truncate(haystack, 80), needle)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
