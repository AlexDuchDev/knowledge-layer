package ingestion_connectors

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// SlackChannelRef is a channel the bot can see (external_ref = id).
type SlackChannelRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ListSlackChannels lists public and private channels (first pages; Slack may paginate).
func (s *Service) ListSlackChannels(ctx context.Context, botToken string) ([]SlackChannelRef, error) {
	botToken = strings.TrimSpace(botToken)
	if botToken == "" {
		return nil, fmt.Errorf("slack: bot_token required")
	}
	var all []SlackChannelRef
	cursor := ""
	for i := 0; i < 10; i++ {
		u := "https://slack.com/api/conversations.list?types=public_channel,private_channel&limit=200"
		if cursor != "" {
			u += "&cursor=" + url.QueryEscape(cursor)
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+botToken)
		resp, err := s.HTTP.Do(req)
		if err != nil {
			return nil, err
		}
		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return nil, err
		}
		var parsed struct {
			OK               bool `json:"ok"`
			Error            string
			ResponseMetadata struct {
				NextCursor string `json:"next_cursor"`
			} `json:"response_metadata"`
			Channels []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"channels"`
		}
		if json.Unmarshal(body, &parsed) != nil {
			return nil, fmt.Errorf("slack: invalid json")
		}
		if !parsed.OK {
			return nil, fmt.Errorf("slack: %s", parsed.Error)
		}
		for _, ch := range parsed.Channels {
			if ch.ID != "" && ch.Name != "" {
				all = append(all, SlackChannelRef{ID: ch.ID, Name: ch.Name})
			}
		}
		cursor = strings.TrimSpace(parsed.ResponseMetadata.NextCursor)
		if cursor == "" {
			break
		}
	}
	return all, nil
}
