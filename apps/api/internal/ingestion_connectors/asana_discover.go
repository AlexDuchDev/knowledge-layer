package ingestion_connectors

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// AsanaProjectRef is a project the PAT can see (with workspace label for UI).
type AsanaProjectRef struct {
	GID           string `json:"gid"`
	Name          string `json:"name"`
	WorkspaceGID  string `json:"workspace_gid,omitempty"`
	WorkspaceName string `json:"workspace_name,omitempty"`
	PermalinkURL  string `json:"permalink_url,omitempty"`
}

// ListAsanaProjects lists projects across all workspaces for the PAT (bounded).
func (s *Service) ListAsanaProjects(ctx context.Context, pat string) ([]AsanaProjectRef, error) {
	pat = strings.TrimSpace(pat)
	if pat == "" {
		return nil, fmt.Errorf("asana: token required")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://app.asana.com/api/1.0/workspaces?limit=50", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+pat)
	resp, err := s.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("asana: workspaces status %d: %s", resp.StatusCode, string(body))
	}
	var wsWrap struct {
		Data []struct {
			GID  string `json:"gid"`
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &wsWrap); err != nil {
		return nil, fmt.Errorf("asana: decode workspaces: %w", err)
	}
	var out []AsanaProjectRef
	const maxProjects = 200
	for _, w := range wsWrap.Data {
		if len(out) >= maxProjects {
			break
		}
		pURL := fmt.Sprintf("https://app.asana.com/api/1.0/projects?workspace=%s&limit=100&opt_fields=name,gid,permalink_url", w.GID)
		preq, err := http.NewRequestWithContext(ctx, http.MethodGet, pURL, nil)
		if err != nil {
			return nil, err
		}
		preq.Header.Set("Authorization", "Bearer "+pat)
		presp, err := s.HTTP.Do(preq)
		if err != nil {
			return nil, err
		}
		pbody, _ := io.ReadAll(presp.Body)
		_ = presp.Body.Close()
		if presp.StatusCode < 200 || presp.StatusCode >= 300 {
			continue
		}
		var projWrap struct {
			Data []struct {
				GID          string `json:"gid"`
				Name         string `json:"name"`
				PermalinkURL string `json:"permalink_url"`
			} `json:"data"`
		}
		if json.Unmarshal(pbody, &projWrap) != nil {
			continue
		}
		for _, p := range projWrap.Data {
			if len(out) >= maxProjects {
				return out, nil
			}
			out = append(out, AsanaProjectRef{
				GID:           p.GID,
				Name:          p.Name,
				WorkspaceGID:  w.GID,
				WorkspaceName: w.Name,
				PermalinkURL:  p.PermalinkURL,
			})
		}
	}
	return out, nil
}
