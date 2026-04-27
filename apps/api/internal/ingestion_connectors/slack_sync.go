package ingestion_connectors

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"

	"github.com/google/uuid"

	"github.com/knowledgelayer/api/internal/ingestion_connectors/families/chat"
)

// SyncSlack performs Slack v1 sync: channel history + thread replies, raw + normalized chat rows.
func (s *Service) SyncSlack(ctx context.Context, feedID uuid.UUID) (*IngestionRun, error) {
	feed, err := s.GetSourceFeed(ctx, feedID)
	if err != nil {
		return nil, err
	}
	conn, err := s.GetConnector(ctx, feed.ConnectorID)
	if err != nil {
		return nil, err
	}
	if conn.Type != "slack" {
		return nil, fmt.Errorf("connector is %s, not slack", conn.Type)
	}
	if feed.Status != "active" {
		return nil, errors.New("source feed must be active")
	}

	cfg, err := ParseSlackFeedConfig(feed.ConnectorConfigJSON)
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	if err := ValidateSlackV1ForActivation(feed, cfg); err != nil {
		return nil, err
	}

	runID, err := s.startIngestionRun(ctx, feedID)
	if err != nil {
		return nil, err
	}

	channelID := feed.ExternalRef
	ingested := 0
	deduped := 0
	warnings := 0
	errs := 0

	var maxTS string
	seenTS := map[string]bool{}

	appendMax := func(ts string) {
		if ts == "" {
			return
		}
		if maxTS == "" || ts > maxTS {
			maxTS = ts
		}
	}

	persist := func(msg slackHistoryMessage) {
		if msg.Type != "" && msg.Type != "message" {
			warnings++
			return
		}
		if msg.Ts == "" {
			warnings++
			return
		}
		if seenTS[msg.Ts] {
			deduped++
			return
		}
		seenTS[msg.Ts] = true
		appendMax(msg.Ts)

		rawPayload, jerr := json.Marshal(msg)
		if jerr != nil {
			errs++
			return
		}
		h := hashBytes(rawPayload)
		extID := fmt.Sprintf("slack-%s-%s", channelID, msg.Ts)

		var msgObj map[string]any
		_ = json.Unmarshal(rawPayload, &msgObj)
		extra := map[string]any{"slack_message": msgObj}
		metaJSON, merr := s.buildRawArtifactMetadataJSON(ctx, *conn, *feed, string(chat.ArtifactKindMessage), rawPayload, extra)
		if merr != nil {
			errs++
			return
		}
		rawID, inserted, ierr := insertRawArtifactRow(ctx, s.pool, feedID, runID, "slack_message", extID, h, "", metaJSON, nil, nil)
		if ierr != nil {
			errs++
			return
		}
		if !inserted {
			deduped++
			return
		}

		threadTs := msg.ThreadTs
		if threadTs == "" {
			threadTs = msg.Ts
		}
		normMsg := chat.FromSlackMessage(feedID, channelID, msg.Ts, msg.User, msg.Text, threadTs)
		normPayload, nerr := json.Marshal(normMsg)
		if nerr != nil {
			errs++
			return
		}
		nh := hashBytes(normPayload)
		tag, qerr := s.pool.Exec(ctx, `
			INSERT INTO normalized_records (raw_artifact_id, source_feed_id, record_type, structured_payload_json, record_hash)
			VALUES ($1,$2,$3,$4,$5)
			ON CONFLICT (source_feed_id, record_hash) DO NOTHING`,
			rawID, feedID, chat.RecordTypeChatMessage, normPayload, nh)
		if qerr != nil {
			errs++
			return
		}
		if tag.RowsAffected() == 0 {
			deduped++
			return
		}
		ingested++
	}

	cursor := ""
	for page := 0; page < 20; page++ {
		hist, hErr := s.slackConversationsHistory(ctx, cfg.BotToken, channelID, cursor, slackOldestParam(cfg))
		if hErr != nil {
			errs++
			break
		}
		if !hist.OK {
			errs++
			break
		}
		msgs := append([]slackHistoryMessage(nil), hist.Messages...)
		sort.Slice(msgs, func(i, j int) bool { return msgs[i].Ts < msgs[j].Ts })

		for _, msg := range msgs {
			persist(msg)
			if msg.ReplyCount > 0 && msg.ThreadTs == msg.Ts {
				repCursor := ""
				for rpage := 0; rpage < 20; rpage++ {
					rep, rErr := s.slackConversationsReplies(ctx, cfg.BotToken, channelID, msg.Ts, repCursor)
					if rErr != nil {
						errs++
						break
					}
					if !rep.OK {
						errs++
						break
					}
					rmsgs := append([]slackHistoryMessage(nil), rep.Messages...)
					sort.Slice(rmsgs, func(i, j int) bool { return rmsgs[i].Ts < rmsgs[j].Ts })
					for _, rm := range rmsgs {
						persist(rm)
					}
					repCursor = rep.ResponseMetadata.NextCursor
					if repCursor == "" || !rep.HasMore {
						break
					}
				}
			}
		}
		cursor = hist.ResponseMetadata.NextCursor
		if cursor == "" || !hist.HasMore {
			break
		}
	}

	status := syncRunStatusFromCounts(ingested, errs)
	s.finalizeIngestionRun(ctx, runID, status, ingested, deduped, warnings, errs)
	s.completeSourceFeedSync(ctx, feedID, errs, true)

	if maxTS != "" {
		newCfg, err := MergeSlackLastMessageTS(feed.ConnectorConfigJSON, maxTS)
		if err == nil {
			_, _ = s.pool.Exec(ctx, `UPDATE source_feeds SET connector_config_json=$2, updated_at=now() WHERE id=$1`, feedID, newCfg)
		}
	}

	return s.GetIngestionRun(ctx, runID)
}

func slackOldestParam(cfg *SlackFeedConfig) string {
	if cfg == nil || cfg.SyncState == nil {
		return ""
	}
	return cfg.SyncState.LastMessageTS
}

func (s *Service) slackConversationsHistory(ctx context.Context, token, channelID, cursor, oldest string) (*slackConversationsHistoryResp, error) {
	q := url.Values{}
	q.Set("channel", channelID)
	q.Set("limit", "200")
	if cursor != "" {
		q.Set("cursor", cursor)
	}
	if oldest != "" {
		q.Set("oldest", oldest)
	}
	u := "https://slack.com/api/conversations.history?" + q.Encode()
	return slackGET[slackConversationsHistoryResp](ctx, s.HTTP, token, u)
}

func (s *Service) slackConversationsReplies(ctx context.Context, token, channelID, threadTs, cursor string) (*slackConversationsRepliesResp, error) {
	q := url.Values{}
	q.Set("channel", channelID)
	q.Set("ts", threadTs)
	q.Set("limit", "200")
	if cursor != "" {
		q.Set("cursor", cursor)
	}
	u := "https://slack.com/api/conversations.replies?" + q.Encode()
	return slackGET[slackConversationsRepliesResp](ctx, s.HTTP, token, u)
}

func slackGET[T any](ctx context.Context, client *http.Client, token, u string) (*T, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var out T
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}
