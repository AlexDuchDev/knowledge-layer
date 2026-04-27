package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

type OpenAIClient struct {
	http    *http.Client
	baseURL string
	apiKey  string
	model   string
	headers map[string]string
	mock    bool
	mockOut string
}

func NewOpenAIFromEnv() (*OpenAIClient, error) {
	if os.Getenv("OPENAI_MOCK") == "1" {
		out := os.Getenv("OPENAI_MOCK_OUTPUT")
		if out == "" {
			out = `{"answer":"mock answer","citations":[]}`
		}
		return &OpenAIClient{
			http:    &http.Client{Timeout: 25 * time.Second},
			baseURL: "mock",
			apiKey:  "mock",
			model:   "mock",
			headers: map[string]string{},
			mock:    true,
			mockOut: out,
		}, nil
	}
	apiKey := strings.TrimSpace(os.Getenv("OPENROUTER_API_KEY"))
	baseURL := strings.TrimSpace(os.Getenv("OPENROUTER_BASE_URL"))
	if apiKey != "" {
		if baseURL == "" {
			baseURL = "https://openrouter.ai/api/v1"
		}
		if strings.HasSuffix(baseURL, "/") {
			baseURL = strings.TrimSuffix(baseURL, "/")
		}
		return &OpenAIClient{
			http:    &http.Client{Timeout: 25 * time.Second},
			baseURL: baseURL,
			apiKey:  apiKey,
			model:   readDefaultChatModel(),
			headers: readOpenRouterAttributionHeaders(),
		}, nil
	}

	apiKey = strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	if apiKey == "" {
		return nil, errors.New("OPENAI_API_KEY or OPENROUTER_API_KEY not set")
	}
	baseURL = strings.TrimSpace(os.Getenv("OPENAI_BASE_URL"))
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}
	model := readDefaultChatModel()
	return &OpenAIClient{
		http:    &http.Client{Timeout: 25 * time.Second},
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
		headers: map[string]string{},
	}, nil
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
	// Structured outputs (OpenRouter/OpenAI compatible)
	ResponseFormat any `json:"response_format,omitempty"`
}

func marshalChatRequest(model, system string, userContent any, temp float64, responseFormat any) ([]byte, error) {
	userMsg := map[string]any{"role": "user"}
	switch v := userContent.(type) {
	case string:
		userMsg["content"] = v
	case []map[string]any:
		userMsg["content"] = v
	default:
		return nil, fmt.Errorf("openai: unsupported user content type %T", userContent)
	}
	reqMap := map[string]any{
		"model":       model,
		"messages":    []any{map[string]any{"role": "system", "content": system}, userMsg},
		"temperature": temp,
	}
	if responseFormat != nil {
		reqMap["response_format"] = responseFormat
	}
	return json.Marshal(reqMap)
}

type chatResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int     `json:"prompt_tokens"`
		CompletionTokens int     `json:"completion_tokens"`
		TotalTokens      int     `json:"total_tokens"`
		Cost             float64 `json:"cost,omitempty"`
	} `json:"usage,omitempty"`
}

func (c *OpenAIClient) Chat(ctx context.Context, system string, user string) (string, error) {
	return c.ChatScenario(ctx, "", system, user)
}

func (c *OpenAIClient) ChatScenario(ctx context.Context, scenario string, system string, user string) (string, error) {
	return c.chatWithOptions(ctx, scenario, system, user, nil, nil, "")
}

func (c *OpenAIClient) ChatScenarioWithMeta(ctx context.Context, scenario string, system string, user string) (string, map[string]any, error) {
	return c.ChatScenarioWithMetaUserContent(ctx, scenario, system, any(user))
}

// ChatScenarioWithMetaAndResponseFormat runs a chat completion with an OpenAI/OpenRouter-compatible
// response_format payload (structured outputs). The returned content is the model's message content
// (typically JSON when response_format is set).
func (c *OpenAIClient) ChatScenarioWithMetaAndResponseFormat(ctx context.Context, scenario string, system string, user string, responseFormat any) (string, map[string]any, error) {
	start := time.Now()
	content, out, model, err := c.chatWithOptionsRaw(ctx, scenario, system, any(user), nil, responseFormat, "")
	if err != nil {
		return "", nil, err
	}
	meta := map[string]any{
		"id":          strings.TrimSpace(out.ID),
		"model":       model,
		"scenario":    strings.TrimSpace(scenario),
		"duration_ms": time.Since(start).Milliseconds(),
	}
	if out.Usage != nil {
		meta["prompt_tokens"] = out.Usage.PromptTokens
		meta["completion_tokens"] = out.Usage.CompletionTokens
		meta["total_tokens"] = out.Usage.TotalTokens
		meta["cost"] = out.Usage.Cost
	}
	return content, meta, nil
}

// ChatScenarioWithMetaUserContent is like ChatScenarioWithMeta but allows multimodal user `content`
// (string or []map for text / image_url / input_audio parts per OpenAI & OpenRouter).
func (c *OpenAIClient) ChatScenarioWithMetaUserContent(ctx context.Context, scenario string, system string, userContent any) (string, map[string]any, error) {
	start := time.Now()
	content, out, model, err := c.chatWithOptionsRaw(ctx, scenario, system, userContent, nil, nil, "")
	if err != nil {
		return "", nil, err
	}
	meta := map[string]any{
		"id":          strings.TrimSpace(out.ID),
		"model":       model,
		"scenario":    strings.TrimSpace(scenario),
		"duration_ms": time.Since(start).Milliseconds(),
	}
	if out.Usage != nil {
		meta["prompt_tokens"] = out.Usage.PromptTokens
		meta["completion_tokens"] = out.Usage.CompletionTokens
		meta["total_tokens"] = out.Usage.TotalTokens
		meta["cost"] = out.Usage.Cost
	}
	return content, meta, nil
}

func (c *OpenAIClient) chatWithOptions(ctx context.Context, scenario string, system string, user string, temperature *float64, responseFormat any, modelOverride string) (string, error) {
	return c.chatWithOptionsUserContent(ctx, scenario, system, any(user), temperature, responseFormat, modelOverride)
}

func (c *OpenAIClient) chatWithOptionsUserContent(ctx context.Context, scenario string, system string, userContent any, temperature *float64, responseFormat any, modelOverride string) (string, error) {
	if c.mock {
		if _, ok := userContent.(string); ok {
			return c.mockOut, nil
		}
		return c.mockOut, nil
	}
	start := time.Now()
	content, out, model, err := c.chatWithOptionsRaw(ctx, scenario, system, userContent, temperature, responseFormat, modelOverride)
	if err != nil {
		return "", err
	}
	if os.Getenv("LLM_LOG") == "1" {
		d := time.Since(start)
		pt, ct, tt := 0, 0, 0
		cost := 0.0
		if out.Usage != nil {
			pt, ct, tt = out.Usage.PromptTokens, out.Usage.CompletionTokens, out.Usage.TotalTokens
			cost = out.Usage.Cost
		}
		log.Printf("[LLM] chat scenario=%s model=%s dur_ms=%d prompt_tokens=%d completion_tokens=%d total_tokens=%d cost=%g id=%s",
			strings.TrimSpace(scenario), model, d.Milliseconds(), pt, ct, tt, cost, strings.TrimSpace(out.ID))
	}
	return content, nil
}

func (c *OpenAIClient) chatWithOptionsRaw(ctx context.Context, scenario string, system string, userContent any, temperature *float64, responseFormat any, modelOverride string) (string, chatResponse, string, error) {
	if c.mock {
		return c.mockOut, chatResponse{ID: "mock"}, "mock", nil
	}
	model := modelForScenario(c.model, scenario)
	if strings.TrimSpace(modelOverride) != "" {
		model = strings.TrimSpace(modelOverride)
	}
	temp := 0.2
	if temperature != nil {
		temp = *temperature
	}
	b, err := marshalChatRequest(model, system, userContent, temp, responseFormat)
	if err != nil {
		return "", chatResponse{}, model, err
	}
	url := strings.TrimSpace(c.baseURL)
	endpoint := url + "/v1/chat/completions"
	if strings.Contains(url, "/api/v1") {
		endpoint = url + "/chat/completions"
	}
	payload := b
	backoff := []time.Duration{200 * time.Millisecond, 600 * time.Millisecond}
	for attempt := 0; attempt <= len(backoff); attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
		if err != nil {
			return "", chatResponse{}, model, fmt.Errorf("openai: request: %w", err)
		}
		applyAuthHeaders(req, c.apiKey, c.headers)

		resp, err := c.http.Do(req)
		if err != nil {
			if attempt < len(backoff) {
				time.Sleep(backoff[attempt])
				continue
			}
			return "", chatResponse{}, model, fmt.Errorf("openai: do: %w", err)
		}

		var out chatResponse
		func() {
			defer resp.Body.Close()
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				var raw bytes.Buffer
				_, _ = raw.ReadFrom(resp.Body)
				msg := raw.String()
				if (resp.StatusCode == 429 || resp.StatusCode >= 500) && attempt < len(backoff) {
					err = fmt.Errorf("openai: retryable status %d: %s", resp.StatusCode, msg)
					return
				}
				err = fmt.Errorf("openai: status %d: %s", resp.StatusCode, msg)
				return
			}
			if derr := json.NewDecoder(resp.Body).Decode(&out); derr != nil {
				err = fmt.Errorf("openai: decode: %w", derr)
				return
			}
		}()
		if err != nil {
			if attempt < len(backoff) {
				time.Sleep(backoff[attempt])
				continue
			}
			return "", chatResponse{}, model, err
		}
		if len(out.Choices) == 0 {
			return "", chatResponse{}, model, errors.New("openai: empty choices")
		}
		return out.Choices[0].Message.Content, out, model, nil
	}
	return "", chatResponse{}, model, errors.New("openai: retry loop exhausted")
}

func decodeChat(req *http.Request, h *http.Client) (chatResponse, error) {
	resp, err := h.Do(req)
	if err != nil {
		return chatResponse{}, fmt.Errorf("openai: do: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var raw bytes.Buffer
		_, _ = raw.ReadFrom(resp.Body)
		return chatResponse{}, fmt.Errorf("openai: status %d: %s", resp.StatusCode, raw.String())
	}
	var out chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return chatResponse{}, fmt.Errorf("openai: decode: %w", err)
	}
	return out, nil
}

func readDefaultChatModel() string {
	model := strings.TrimSpace(os.Getenv("OPENAI_MODEL"))
	if model == "" {
		model = "gpt-4o-mini"
	}
	return model
}

var nonAlnum = regexp.MustCompile(`[^A-Za-z0-9]+`)

func modelForScenario(defaultModel string, scenario string) string {
	sc := strings.TrimSpace(strings.ToLower(scenario))
	if sc == "" {
		return defaultModel
	}
	key := strings.ToUpper(nonAlnum.ReplaceAllString(sc, "_"))
	envKey := "AI_MODEL_SCENARIO_" + key
	if m := strings.TrimSpace(os.Getenv(envKey)); m != "" {
		return m
	}
	return defaultModel
}

func readOpenRouterAttributionHeaders() map[string]string {
	h := map[string]string{}
	if v := strings.TrimSpace(os.Getenv("OPENROUTER_HTTP_REFERER")); v != "" {
		h["HTTP-Referer"] = v
	}
	if v := strings.TrimSpace(os.Getenv("OPENROUTER_TITLE")); v != "" {
		h["X-OpenRouter-Title"] = v
	}
	if v := strings.TrimSpace(os.Getenv("OPENROUTER_CATEGORIES")); v != "" {
		h["X-OpenRouter-Categories"] = v
	}
	return h
}

func applyAuthHeaders(req *http.Request, apiKey string, extra map[string]string) {
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	for k, v := range extra {
		if strings.TrimSpace(v) == "" {
			continue
		}
		req.Header.Set(k, v)
	}
}

// MetaUserContentCaller is implemented by *OpenAIClient for governed calls that may send multimodal user payloads.
type MetaUserContentCaller interface {
	ChatScenarioWithMetaUserContent(ctx context.Context, scenario, system string, userContent any) (string, map[string]any, error)
}

var _ MetaUserContentCaller = (*OpenAIClient)(nil)
