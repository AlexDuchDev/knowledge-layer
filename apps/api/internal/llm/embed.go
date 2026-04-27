package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"net/http"
	"os"
	"strings"
	"time"
)

const DefaultEmbeddingModel = "text-embedding-3-small"
const EmbeddingDimensions = 1536

type embeddingsRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type embeddingsResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
}

// Embed returns a single text embedding (1536 dims for text-embedding-3-small).
// When OPENAI_MOCK=1, returns a deterministic pseudo-vector derived from input (no API call).
func (c *OpenAIClient) Embed(ctx context.Context, text string) ([]float32, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, errors.New("llm: empty embed input")
	}
	if c.mock {
		return mockEmbeddingVector(text), nil
	}
	reqBody := embeddingsRequest{
		Model: DefaultEmbeddingModel,
		Input: []string{text},
	}
	if m := os.Getenv("OPENAI_EMBEDDING_MODEL"); m != "" {
		reqBody.Model = m
	}
	b, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	url := strings.TrimSpace(c.baseURL)
	endpoint := url + "/v1/embeddings"
	if strings.Contains(url, "/api/v1") {
		endpoint = url + "/embeddings"
	}
	payload := b
	backoff := []time.Duration{200 * time.Millisecond, 600 * time.Millisecond}
	var out embeddingsResponse
	for attempt := 0; attempt <= len(backoff); attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
		if err != nil {
			return nil, err
		}
		applyAuthHeaders(req, c.apiKey, c.headers)
		resp, err := c.http.Do(req)
		if err != nil {
			if attempt < len(backoff) {
				time.Sleep(backoff[attempt])
				continue
			}
			return nil, err
		}
		var reqErr error
		func() {
			defer resp.Body.Close()
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				var raw bytes.Buffer
				_, _ = raw.ReadFrom(resp.Body)
				msg := raw.String()
				if (resp.StatusCode == 429 || resp.StatusCode >= 500) && attempt < len(backoff) {
					reqErr = fmt.Errorf("openai embeddings: retryable status %d: %s", resp.StatusCode, msg)
					return
				}
				reqErr = fmt.Errorf("openai embeddings: status %d: %s", resp.StatusCode, msg)
				return
			}
			if derr := json.NewDecoder(resp.Body).Decode(&out); derr != nil {
				reqErr = derr
				return
			}
		}()
		if reqErr != nil {
			if attempt < len(backoff) {
				time.Sleep(backoff[attempt])
				continue
			}
			return nil, reqErr
		}
		break
	}
	if len(out.Data) == 0 || len(out.Data[0].Embedding) == 0 {
		return nil, errors.New("openai embeddings: empty data")
	}
	raw := out.Data[0].Embedding
	v := make([]float32, len(raw))
	for i, x := range raw {
		v[i] = float32(x)
	}
	return v, nil
}

func mockEmbeddingVector(text string) []float32 {
	v := make([]float32, EmbeddingDimensions)
	h := fnv.New32a()
	_, _ = h.Write([]byte(text))
	seed := h.Sum32()
	for i := range v {
		seed = seed*1103515245 + uint32(i) + 1
		v[i] = float32(seed%1024) / 1024.0 * 0.02
	}
	return v
}
