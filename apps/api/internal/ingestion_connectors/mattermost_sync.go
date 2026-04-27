package ingestion_connectors

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"

	"github.com/knowledgelayer/api/internal/ingestion_connectors/families/chat"
)

// mmPost is a subset of Mattermost post JSON (v4 API).
type mmPost struct {
	ID        string `json:"id"`
	CreateAt  int64  `json:"create_at"`
	UserID    string `json:"user_id"`
	Message   string `json:"message"`
	RootID    string `json:"root_id"`
	ChannelID string `json:"channel_id"`
}

type mmPostsResp struct {
	Order []string          `json:"order"`
	Posts map[string]mmPost `json:"posts"`
}

// SyncMattermost pulls channel posts via Mattermost v4 REST (personal access token).
func (s *Service) SyncMattermost(ctx context.Context, feedID uuid.UUID) (*IngestionRun, error) {
	feed, err := s.GetSourceFeed(ctx, feedID)
	if err != nil {
		return nil, err
	}
	conn, err := s.GetConnector(ctx, feed.ConnectorID)
	if err != nil {
		return nil, err
	}
	if conn.Type != "mattermost" {
		return nil, fmt.Errorf("connector is %s, not mattermost", conn.Type)
	}
	if feed.Status != "active" {
		return nil, errors.New("source feed must be active")
	}

	cfg, err := ParseMattermostFeedConfig(feed.ConnectorConfigJSON)
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	if err := ValidateMattermostV1ForActivation(feed, cfg); err != nil {
		return nil, err
	}

	channelID := strings.TrimSpace(feed.ExternalRef)
	runID, err := s.startIngestionRun(ctx, feedID)
	if err != nil {
		return nil, err
	}

	ingested := 0
	deduped := 0
	warnings := 0
	errs := 0

	since := ""
	if cfg.SyncState != nil {
		since = strings.TrimSpace(cfg.SyncState.LastPostID)
	}

	u := mattermostPostsURL(cfg.BaseURL, channelID, since)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		s.finalizeIngestionRun(ctx, runID, "failed", 0, 0, 0, 1)
		s.completeSourceFeedSync(ctx, feedID, 1, true)
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.Token)

	resp, err := s.HTTP.Do(req)
	if err != nil {
		s.finalizeIngestionRun(ctx, runID, "failed", 0, 0, 0, 1)
		s.completeSourceFeedSync(ctx, feedID, 1, true)
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.finalizeIngestionRun(ctx, runID, "failed", 0, 0, 0, 1)
		s.completeSourceFeedSync(ctx, feedID, 1, true)
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		s.finalizeIngestionRun(ctx, runID, "failed", 0, 0, 0, 1)
		s.completeSourceFeedSync(ctx, feedID, 1, true)
		return nil, fmt.Errorf("mattermost: http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var parsed mmPostsResp
	if err := json.Unmarshal(body, &parsed); err != nil {
		s.finalizeIngestionRun(ctx, runID, "failed", 0, 0, 0, 1)
		s.completeSourceFeedSync(ctx, feedID, 1, true)
		return nil, fmt.Errorf("mattermost: json: %w", err)
	}

	var bestPostID string
	var bestCreate int64
	for _, pid := range parsed.Order {
		p, ok := parsed.Posts[pid]
		if !ok {
			warnings++
			continue
		}
		if p.ID == "" {
			warnings++
			continue
		}
		if p.CreateAt >= bestCreate {
			bestCreate = p.CreateAt
			bestPostID = p.ID
		}

		rawPayload, jerr := json.Marshal(p)
		if jerr != nil {
			errs++
			continue
		}
		h := hashBytes(rawPayload)
		extID := "mm-" + chat.MattermostExternalMessageID(channelID, p.ID)

		var postObj map[string]any
		_ = json.Unmarshal(rawPayload, &postObj)
		extra := map[string]any{"mattermost_post": postObj}
		metaJSON, merr := s.buildRawArtifactMetadataJSON(ctx, *conn, *feed, string(chat.ArtifactKindMessage), rawPayload, extra)
		if merr != nil {
			errs++
			continue
		}
		rawID, inserted, ierr := insertRawArtifactRow(ctx, s.pool, feedID, runID, "mattermost_post", extID, h, "", metaJSON, nil, nil)
		if ierr != nil {
			errs++
			continue
		}
		if !inserted {
			deduped++
			continue
		}

		chRef := channelID
		if strings.TrimSpace(p.ChannelID) != "" {
			chRef = p.ChannelID
		}
		normMsg := chat.FromMattermostPost(feedID, chRef, p.ID, p.RootID, p.UserID, p.Message, p.CreateAt)
		normPayload, nerr := json.Marshal(normMsg)
		if nerr != nil {
			errs++
			continue
		}
		nh := hashBytes(normPayload)
		tag, qerr := s.pool.Exec(ctx, `
			INSERT INTO normalized_records (raw_artifact_id, source_feed_id, record_type, structured_payload_json, record_hash)
			VALUES ($1,$2,$3,$4,$5)
			ON CONFLICT (source_feed_id, record_hash) DO NOTHING`,
			rawID, feedID, chat.RecordTypeChatMessage, normPayload, nh)
		if qerr != nil {
			errs++
			continue
		}
		if tag.RowsAffected() == 0 {
			deduped++
			continue
		}
		ingested++
	}

	status := syncRunStatusFromCounts(ingested, errs)
	s.finalizeIngestionRun(ctx, runID, status, ingested, deduped, warnings, errs)
	s.completeSourceFeedSync(ctx, feedID, errs, true)

	if bestPostID != "" {
		newCfg, err := MergeMattermostLastPostID(feed.ConnectorConfigJSON, bestPostID)
		if err == nil {
			_, _ = s.pool.Exec(ctx, `UPDATE source_feeds SET connector_config_json=$2, updated_at=now() WHERE id=$1`, feedID, newCfg)
		}
	}

	return s.GetIngestionRun(ctx, runID)
}

func mattermostPostsURL(base, channelID, sincePostID string) string {
	b := strings.TrimRight(strings.TrimSpace(base), "/")
	u := fmt.Sprintf("%s/api/v4/channels/%s/posts", b, url.PathEscape(channelID))
	q := url.Values{}
	q.Set("per_page", "100")
	if sincePostID != "" {
		q.Set("since", sincePostID)
	}
	return u + "?" + q.Encode()
}
