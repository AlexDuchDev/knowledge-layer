package ingestion_connectors

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// MattermostChannelRef is a channel in a team (external_ref = channel id).
type MattermostChannelRef struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	TeamID      string `json:"team_id"`
	TeamName    string `json:"team_name,omitempty"`
}

// ListMattermostChannels aggregates channels from all teams the user belongs to.
func (s *Service) ListMattermostChannels(ctx context.Context, cfg *MattermostFeedConfig) ([]MattermostChannelRef, error) {
	if cfg == nil || cfg.BaseURL == "" || cfg.Token == "" {
		return nil, fmt.Errorf("mattermost: base_url and token required")
	}
	base := strings.TrimRight(cfg.BaseURL, "/")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/v4/users/me/teams", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.Token)
	resp, err := s.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("mattermost: teams status %d: %s", resp.StatusCode, string(body))
	}
	var teams []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if json.Unmarshal(body, &teams) != nil {
		return nil, fmt.Errorf("mattermost: decode teams")
	}
	var out []MattermostChannelRef
	const maxTotal = 300
	for _, t := range teams {
		if len(out) >= maxTotal {
			break
		}
		chURL := fmt.Sprintf("%s/api/v4/users/me/teams/%s/channels?page=0&per_page=200", base, t.ID)
		creq, err := http.NewRequestWithContext(ctx, http.MethodGet, chURL, nil)
		if err != nil {
			return nil, err
		}
		creq.Header.Set("Authorization", "Bearer "+cfg.Token)
		cresp, err := s.HTTP.Do(creq)
		if err != nil {
			return nil, err
		}
		cbody, _ := io.ReadAll(cresp.Body)
		_ = cresp.Body.Close()
		if cresp.StatusCode < 200 || cresp.StatusCode >= 300 {
			continue
		}
		var channels []struct {
			ID          string `json:"id"`
			DisplayName string `json:"display_name"`
			Type        string `json:"type"`
			DeleteAt    int64  `json:"delete_at"`
		}
		if json.Unmarshal(cbody, &channels) != nil {
			continue
		}
		for _, ch := range channels {
			if len(out) >= maxTotal {
				return out, nil
			}
			if ch.DeleteAt != 0 || ch.Type == "D" || ch.ID == "" {
				continue
			}
			name := ch.DisplayName
			if name == "" {
				name = ch.ID
			}
			out = append(out, MattermostChannelRef{
				ID:          ch.ID,
				DisplayName: name,
				TeamID:      t.ID,
				TeamName:    t.Name,
			})
		}
	}
	return out, nil
}
