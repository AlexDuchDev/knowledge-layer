package llm

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"
)

// Transcribe turns short audio bytes into plain text.
// - OpenAI-compatible hosts: multipart POST …/v1/audio/transcriptions (Whisper).
// - OpenRouter: POST …/chat/completions with input_audio (see OPENROUTER_TRANSCRIPTION_MODEL).
func (c *OpenAIClient) Transcribe(ctx context.Context, audio []byte, format string) (string, error) {
	if c.mock {
		return "mock voice transcript", nil
	}
	if len(audio) == 0 {
		return "", fmt.Errorf("llm: empty audio")
	}
	format = strings.ToLower(strings.TrimSpace(format))
	if format == "" {
		format = "wav"
	}
	lower := strings.ToLower(c.baseURL)
	if strings.Contains(lower, "openrouter") {
		return c.transcribeOpenRouterChat(ctx, audio, format)
	}
	return c.transcribeOpenAIWhisper(ctx, audio, format)
}

func getenvDefault(k, def string) string {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return def
	}
	return v
}

func (c *OpenAIClient) transcribeOpenAIWhisper(ctx context.Context, audio []byte, format string) (string, error) {
	model := getenvDefault("OPENAI_TRANSCRIPTION_MODEL", "whisper-1")
	buf := &bytes.Buffer{}
	mw := multipart.NewWriter(buf)
	part, err := mw.CreateFormFile("file", "upload."+format)
	if err != nil {
		return "", fmt.Errorf("llm: multipart file: %w", err)
	}
	if _, err := part.Write(audio); err != nil {
		return "", err
	}
	if err := mw.WriteField("model", model); err != nil {
		return "", err
	}
	if err := mw.Close(); err != nil {
		return "", err
	}
	base := strings.TrimRight(strings.TrimSpace(c.baseURL), "/")
	endpoint := base + "/v1/audio/transcriptions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	for k, v := range c.headers {
		if strings.TrimSpace(v) != "" {
			req.Header.Set(k, v)
		}
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("llm: transcribe: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("llm: transcribe status %d: %s", resp.StatusCode, string(body))
	}
	var out struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", fmt.Errorf("llm: transcribe decode: %w", err)
	}
	return strings.TrimSpace(out.Text), nil
}

func (c *OpenAIClient) transcribeOpenRouterChat(ctx context.Context, audio []byte, format string) (string, error) {
	model := getenvDefault("OPENROUTER_TRANSCRIPTION_MODEL", "openai/whisper-large-v3")
	b64 := base64.StdEncoding.EncodeToString(audio)
	parts := []map[string]any{
		{"type": "text", "text": "Transcribe the attached audio verbatim. Output only the transcript, no preamble."},
		{
			"type": "input_audio",
			"input_audio": map[string]any{
				"data":   b64,
				"format": format,
			},
		},
	}
	msgs := []map[string]any{
		{"role": "system", "content": "You transcribe user audio. Reply with transcript only."},
		{"role": "user", "content": parts},
	}
	reqMap := map[string]any{
		"model":       model,
		"messages":    msgs,
		"temperature": 0.0,
	}
	payload, err := json.Marshal(reqMap)
	if err != nil {
		return "", err
	}
	base := strings.TrimRight(strings.TrimSpace(c.baseURL), "/")
	endpoint := base + "/chat/completions"
	if !strings.Contains(base, "/api/v1") {
		endpoint = base + "/api/v1/chat/completions"
	}
	backoff := []time.Duration{200 * time.Millisecond, 600 * time.Millisecond}
	var lastErr error
	for attempt := 0; attempt <= len(backoff); attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
		if err != nil {
			return "", err
		}
		applyAuthHeaders(req, c.apiKey, c.headers)
		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("llm: openrouter transcribe: %w", err)
			if attempt < len(backoff) {
				time.Sleep(backoff[attempt])
				continue
			}
			return "", lastErr
		}
		txt, rerr := func() (string, error) {
			defer resp.Body.Close()
			rb, _ := io.ReadAll(resp.Body)
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				return "", fmt.Errorf("llm: openrouter transcribe status %d: %s", resp.StatusCode, string(rb))
			}
			var out chatResponse
			if err := json.Unmarshal(rb, &out); err != nil {
				return "", fmt.Errorf("llm: openrouter transcribe decode: %w", err)
			}
			if len(out.Choices) == 0 {
				return "", fmt.Errorf("llm: openrouter transcribe: empty choices")
			}
			return strings.TrimSpace(out.Choices[0].Message.Content), nil
		}()
		if rerr == nil {
			return txt, nil
		}
		lastErr = rerr
		if (strings.Contains(rerr.Error(), "status 429") || strings.Contains(rerr.Error(), "status 5")) && attempt < len(backoff) {
			time.Sleep(backoff[attempt])
			continue
		}
		return "", rerr
	}
	return "", lastErr
}
