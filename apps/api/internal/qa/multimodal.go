package qa

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/knowledgelayer/api/internal/llm"
)

const maxAskImages = 8
const maxAskAudioBytes = 25 * 1024 * 1024

// PreprocessAskMultimodal decodes optional voice input into text (appended to Question) and
// sets a default Question when only images are provided. Clears Audio* fields after success.
func (s *Service) PreprocessAskMultimodal(ctx context.Context, in *AskEntityInput) error {
	if in == nil {
		return nil
	}
	audioB64 := strings.TrimSpace(in.AudioBase64)
	if audioB64 != "" {
		raw, err := base64.StdEncoding.DecodeString(audioB64)
		if err != nil {
			return fmt.Errorf("audio_base64: %w", err)
		}
		if len(raw) > maxAskAudioBytes {
			return errors.New("audio payload too large (max 25MB decoded)")
		}
		client, err := llm.NewOpenAIFromEnv()
		if err != nil {
			return err
		}
		tr, err := client.Transcribe(ctx, raw, in.AudioFormat)
		if err != nil {
			return fmt.Errorf("transcribe: %w", err)
		}
		q := strings.TrimSpace(in.Question)
		if q == "" {
			in.Question = tr
		} else {
			in.Question = q + "\n\n[Transcribed audio]\n" + tr
		}
		in.AudioBase64 = ""
		in.AudioFormat = ""
	}
	if strings.TrimSpace(in.Question) == "" && len(in.Images) > 0 {
		in.Question = "Answer using the attached image(s) and the evidence blocks below."
	}
	return nil
}

func askImagesToLLMParts(in []AskImageAttachment) ([]map[string]any, error) {
	var out []map[string]any
	for _, im := range in {
		if len(out) >= maxAskImages {
			break
		}
		p, err := imageAttachmentToPart(im)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

func imageAttachmentToPart(im AskImageAttachment) (map[string]any, error) {
	u := strings.TrimSpace(im.URL)
	b64 := strings.TrimSpace(im.DataBase64)
	if u != "" {
		parsed, err := url.Parse(u)
		if err != nil || parsed.Scheme == "" {
			return nil, fmt.Errorf("invalid image url")
		}
		scheme := strings.ToLower(parsed.Scheme)
		if scheme != "https" && scheme != "data" {
			return nil, fmt.Errorf("image url must use https or data: scheme")
		}
		detail := strings.TrimSpace(im.Detail)
		if detail == "" {
			detail = "auto"
		}
		return map[string]any{
			"type": "image_url",
			"image_url": map[string]any{
				"url":    u,
				"detail": detail,
			},
		}, nil
	}
	if b64 == "" {
		return nil, errors.New("each image needs url or data_base64")
	}
	mt := strings.TrimSpace(im.MediaType)
	if mt == "" {
		mt = "image/png"
	}
	if !strings.HasPrefix(strings.ToLower(mt), "image/") {
		return nil, fmt.Errorf("media_type must be image/* (got %q)", mt)
	}
	dataURL := "data:" + mt + ";base64," + b64
	detail := strings.TrimSpace(im.Detail)
	if detail == "" {
		detail = "auto"
	}
	return map[string]any{
		"type": "image_url",
		"image_url": map[string]any{
			"url":    dataURL,
			"detail": detail,
		},
	}, nil
}
