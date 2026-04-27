package ingestion_connectors

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/knowledgelayer/api/internal/ingestion_connectors/families/email"
)

type gmailFeedConfig struct {
	AccessToken   string `json:"gmail_oauth_access_token"`
	RefreshToken  string `json:"gmail_oauth_refresh_token,omitempty"`
	ExpiryRFC3339 string `json:"gmail_oauth_expiry,omitempty"`
}

// ValidateGmailSourceFeedForActivation checks OAuth access token for active feeds.
func ValidateGmailSourceFeedForActivation(feed *SourceFeed) error {
	if feed == nil {
		return errors.New("gmail: nil feed")
	}
	_, err := parseGmailFeedConfig(feed.ConnectorConfigJSON)
	return err
}

func parseGmailFeedConfig(raw json.RawMessage) (*gmailFeedConfig, error) {
	var c gmailFeedConfig
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, err
	}
	if strings.TrimSpace(c.AccessToken) == "" && strings.TrimSpace(c.RefreshToken) == "" {
		return nil, errors.New("gmail: gmail_oauth_access_token or gmail_oauth_refresh_token required in connector_config_json")
	}
	return &c, nil
}

// SyncGmail lists recent message ids and fetches metadata (v1).
func (s *Service) SyncGmail(ctx context.Context, feedID uuid.UUID) (*IngestionRun, error) {
	feed, err := s.GetSourceFeed(ctx, feedID)
	if err != nil {
		return nil, err
	}
	conn, err := s.GetConnector(ctx, feed.ConnectorID)
	if err != nil {
		return nil, err
	}
	if conn.Type != "gmail" {
		return nil, fmt.Errorf("connector is %s, not gmail", conn.Type)
	}
	if feed.Status != "active" {
		return nil, errors.New("source feed must be active")
	}
	cfg, err := parseGmailFeedConfig(feed.ConnectorConfigJSON)
	if err != nil {
		return nil, err
	}
	access, err := s.resolveGmailAccessToken(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(access) == "" {
		return nil, errors.New("gmail: no valid access token (refresh failed or missing)")
	}

	runID, err := s.startIngestionRun(ctx, feedID)
	if err != nil {
		return nil, err
	}

	listURL := "https://gmail.googleapis.com/gmail/v1/users/me/messages?maxResults=15"
	if q := strings.TrimSpace(feed.ExternalRef); q != "" {
		listURL += "&q=" + url.QueryEscape(q)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, listURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+access)
	resp, err := s.HTTP.Do(req)
	if err != nil {
		s.finalizeIngestionRun(ctx, runID, "failed", 0, 0, 0, 1)
		s.completeSourceFeedSync(ctx, feedID, 1, false)
		return nil, err
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		s.finalizeIngestionRun(ctx, runID, "failed", 0, 0, 0, 1)
		s.completeSourceFeedSync(ctx, feedID, 1, false)
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		s.finalizeIngestionRun(ctx, runID, "failed", 0, 0, 0, 1)
		s.completeSourceFeedSync(ctx, feedID, 1, false)
		return nil, fmt.Errorf("gmail list: status %d: %s", resp.StatusCode, string(body))
	}
	var list struct {
		Messages []struct {
			ID string `json:"id"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(body, &list); err != nil {
		s.finalizeIngestionRun(ctx, runID, "failed", 0, 0, 0, 1)
		s.completeSourceFeedSync(ctx, feedID, 1, false)
		return nil, err
	}

	ingested, deduped, warnings, errs := 0, 0, 0, 0
	for _, ref := range list.Messages {
		if ref.ID == "" {
			continue
		}
		msgURL := "https://gmail.googleapis.com/gmail/v1/users/me/messages/" + ref.ID + "?format=metadata&metadataHeaders=Subject&metadataHeaders=From"
		mreq, err := http.NewRequestWithContext(ctx, http.MethodGet, msgURL, nil)
		if err != nil {
			errs++
			continue
		}
		mreq.Header.Set("Authorization", "Bearer "+access)
		mresp, err := s.HTTP.Do(mreq)
		if err != nil {
			errs++
			continue
		}
		mb, err := io.ReadAll(mresp.Body)
		mresp.Body.Close()
		if err != nil || mresp.StatusCode < 200 || mresp.StatusCode >= 300 {
			errs++
			continue
		}
		var msg struct {
			ID           string `json:"id"`
			Snippet      string `json:"snippet"`
			InternalDate string `json:"internalDate"`
			Payload      struct {
				Headers []struct {
					Name  string `json:"name"`
					Value string `json:"value"`
				} `json:"headers"`
			} `json:"payload"`
		}
		if err := json.Unmarshal(mb, &msg); err != nil {
			errs++
			continue
		}
		subj, from := "", ""
		for _, h := range msg.Payload.Headers {
			switch strings.ToLower(h.Name) {
			case "subject":
				subj = h.Value
			case "from":
				from = h.Value
			}
		}
		rawPayload := mb
		h := hashBytes(rawPayload)
		extID := "gmail-" + msg.ID
		var rawObj map[string]any
		_ = json.Unmarshal(rawPayload, &rawObj)
		extra := map[string]any{"gmail_message": rawObj}
		metaJSON, merr := s.buildRawArtifactMetadataJSON(ctx, *conn, *feed, email.ArtifactKindRFC822, rawPayload, extra)
		if merr != nil {
			errs++
			continue
		}
		rawID, inserted, ierr := insertRawArtifactRow(ctx, s.pool, feedID, runID, "gmail_message", extID, h, "", metaJSON, nil, nil)
		if ierr != nil {
			errs++
			continue
		}
		if !inserted {
			deduped++
			continue
		}
		var internal *time.Time
		if msg.InternalDate != "" {
			if ms, err := strconv.ParseInt(msg.InternalDate, 10, 64); err == nil && ms > 0 {
				t := time.UnixMilli(ms).UTC()
				internal = &t
			}
		}
		norm := email.NormalizedEmailMessage{
			SourceFeedID:    feedID,
			ConnectorFamily: "email",
			ConnectorType:   "gmail",
			ExternalRef:     msg.ID,
			Subject:         subj,
			Snippet:         msg.Snippet,
			InternalDate:    internal,
			FromRef:         from,
		}
		normPayload, _ := json.Marshal(norm)
		nh := hashBytes(normPayload)
		tag, qerr := s.pool.Exec(ctx, `
			INSERT INTO normalized_records (raw_artifact_id, source_feed_id, record_type, structured_payload_json, record_hash)
			VALUES ($1,$2,$3,$4,$5)
			ON CONFLICT (source_feed_id, record_hash) DO NOTHING`,
			rawID, feedID, email.RecordTypeEmailMessage, normPayload, nh)
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
	s.completeSourceFeedSync(ctx, feedID, errs, false)
	return s.GetIngestionRun(ctx, runID)
}
