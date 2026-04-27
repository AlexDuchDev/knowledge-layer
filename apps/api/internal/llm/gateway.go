package llm

import (
	"context"
	"encoding/json"
	"fmt"
)

// ResponseFormat matches OpenRouter/OpenAI-compatible structured outputs.
// See: https://openrouter.ai/docs/api/reference/overview (response_format).
type ResponseFormat struct {
	Type       string          `json:"type"`
	JSONSchema *JSONSchemaSpec `json:"json_schema,omitempty"`
}

type JSONSchemaSpec struct {
	Name   string          `json:"name"`
	Strict *bool           `json:"strict,omitempty"`
	Schema json.RawMessage `json:"schema"`
}

type ChatCompletionInput struct {
	Scenario string
	System   string
	User     string

	Temperature    *float64
	ResponseFormat *ResponseFormat
	Model          string // optional override
}

// ChatCompletion is the forward-looking gateway API (scenario-aware + structured output support).
// It currently delegates to the existing OpenAI-compatible request shape used by OpenAIClient.
func (c *OpenAIClient) ChatCompletion(ctx context.Context, in ChatCompletionInput) (string, error) {
	if c == nil {
		return "", fmt.Errorf("llm: nil client")
	}
	system := in.System
	user := in.User
	if system == "" || user == "" {
		// Keep strict for now: callers should always provide both.
		return "", fmt.Errorf("llm: system and user are required")
	}
	var rf any
	if in.ResponseFormat != nil {
		rf = in.ResponseFormat
	}
	return c.chatWithOptions(ctx, in.Scenario, system, user, in.Temperature, rf, in.Model)
}
