package llm

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestModelForScenario_EnvOverride(t *testing.T) {
	t.Setenv("AI_MODEL_SCENARIO_AI_SUMMARIZE", "openai/gpt-5.2")
	got := modelForScenario("gpt-4o-mini", "ai_summarize")
	if got != "openai/gpt-5.2" {
		t.Fatalf("expected override model, got %q", got)
	}
}

func TestOpenRouterBaseURL_ChatAndEmbeddingsPathsAndHeaders(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "k")
	t.Setenv("OPENROUTER_BASE_URL", "https://openrouter.ai/api/v1")
	t.Setenv("OPENROUTER_HTTP_REFERER", "https://example.com")
	t.Setenv("OPENROUTER_TITLE", "KnowledgeLayer")
	// Ensure we don't fall back to OPENAI_*.
	t.Setenv("OPENAI_API_KEY", "")

	var seen []string
	var authOK, refererOK, titleOK int

	rt := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		seen = append(seen, r.URL.String())
		if r.Header.Get("Authorization") == "Bearer k" {
			authOK++
		}
		if r.Header.Get("HTTP-Referer") == "https://example.com" {
			refererOK++
		}
		if r.Header.Get("X-OpenRouter-Title") == "KnowledgeLayer" {
			titleOK++
		}
		body := ""
		switch {
		case strings.HasSuffix(r.URL.Path, "/chat/completions"):
			body = `{"id":"gen-1","choices":[{"message":{"role":"assistant","content":"ok"}}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2,"cost":0.0001}}`
		case strings.HasSuffix(r.URL.Path, "/embeddings"):
			body = `{"data":[{"embedding":[0.1,0.2]}]}`
		default:
			body = `{}`
		}
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewBufferString(body)),
			Header:     make(http.Header),
		}, nil
	})

	c, err := NewOpenAIFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	c.http = &http.Client{Transport: rt}

	if _, err := c.ChatScenario(context.Background(), "ai_summarize", "sys", "user"); err != nil {
		t.Fatalf("chat: %v", err)
	}
	if _, err := c.Embed(context.Background(), "hello"); err != nil {
		t.Fatalf("embed: %v", err)
	}

	if len(seen) != 2 {
		t.Fatalf("expected 2 requests, got %d: %v", len(seen), seen)
	}
	if !strings.Contains(seen[0], "/api/v1/chat/completions") {
		t.Fatalf("expected chat via /api/v1/chat/completions, got %q", seen[0])
	}
	if !strings.Contains(seen[1], "/api/v1/embeddings") {
		t.Fatalf("expected embeddings via /api/v1/embeddings, got %q", seen[1])
	}
	// Headers present for both calls (attribution + bearer).
	if authOK != 2 {
		t.Fatalf("expected bearer auth on 2 requests, got %d", authOK)
	}
	if refererOK != 2 {
		t.Fatalf("expected HTTP-Referer on 2 requests, got %d", refererOK)
	}
	if titleOK != 2 {
		t.Fatalf("expected X-OpenRouter-Title on 2 requests, got %d", titleOK)
	}

	// Ensure OPENROUTER_* was preferred (sanity check for this test).
	if strings.TrimSpace(os.Getenv("OPENROUTER_API_KEY")) == "" {
		t.Fatalf("expected OPENROUTER_API_KEY set")
	}
}
