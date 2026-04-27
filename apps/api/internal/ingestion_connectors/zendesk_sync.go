package ingestion_connectors

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/knowledgelayer/api/internal/ingestion_connectors/families/crm_support"
)

const maxZendeskTicketsPerSync = 15

type zendeskFeedConfig struct {
	Subdomain string `json:"zendesk_subdomain"`
	Email     string `json:"zendesk_email"`
	APIToken  string `json:"zendesk_api_token"`
	FeedKind  string `json:"zendesk_feed_kind"` // all | view
}

func parseZendeskFeedConfig(raw json.RawMessage) (*zendeskFeedConfig, error) {
	var c zendeskFeedConfig
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, err
	}
	c.Subdomain = strings.TrimSpace(c.Subdomain)
	c.Email = strings.TrimSpace(c.Email)
	c.APIToken = strings.TrimSpace(c.APIToken)
	c.FeedKind = strings.ToLower(strings.TrimSpace(c.FeedKind))
	if c.FeedKind == "" {
		c.FeedKind = "all"
	}
	if c.Subdomain == "" || c.Email == "" || c.APIToken == "" {
		return nil, errors.New("zendesk: zendesk_subdomain, zendesk_email, and zendesk_api_token required")
	}
	switch c.FeedKind {
	case "all", "view":
	default:
		return nil, errors.New("zendesk: zendesk_feed_kind must be all or view")
	}
	return &c, nil
}

// ValidateZendeskSourceFeedForActivation validates credentials; view mode requires external_ref (view id).
func ValidateZendeskSourceFeedForActivation(feed *SourceFeed) error {
	if feed == nil {
		return errors.New("zendesk: nil feed")
	}
	cfg, err := parseZendeskFeedConfig(feed.ConnectorConfigJSON)
	if err != nil {
		return err
	}
	if cfg.FeedKind == "view" && strings.TrimSpace(feed.ExternalRef) == "" {
		return errors.New("zendesk view: external_ref required (numeric view id)")
	}
	return nil
}

func zendeskBasicAuth(email, token string) string {
	creds := email + "/token:" + token
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(creds))
}

// SyncZendesk lists tickets (or a view) and ingests ticket + comment rows (v1).
func (s *Service) SyncZendesk(ctx context.Context, feedID uuid.UUID) (*IngestionRun, error) {
	feed, err := s.GetSourceFeed(ctx, feedID)
	if err != nil {
		return nil, err
	}
	conn, err := s.GetConnector(ctx, feed.ConnectorID)
	if err != nil {
		return nil, err
	}
	if conn.Type != "zendesk" {
		return nil, fmt.Errorf("connector is %s, not zendesk", conn.Type)
	}
	if feed.Status != "active" {
		return nil, errors.New("source feed must be active")
	}
	cfg, err := parseZendeskFeedConfig(feed.ConnectorConfigJSON)
	if err != nil {
		return nil, err
	}
	if err := ValidateZendeskSourceFeedForActivation(feed); err != nil {
		return nil, err
	}

	runID, err := s.startIngestionRun(ctx, feedID)
	if err != nil {
		return nil, err
	}

	base := fmt.Sprintf("https://%s.zendesk.com", strings.TrimSpace(cfg.Subdomain))
	var listURL string
	if cfg.FeedKind == "view" {
		vid := url.PathEscape(strings.TrimSpace(feed.ExternalRef))
		listURL = fmt.Sprintf("%s/api/v2/views/%s/tickets.json?per_page=%d", base, vid, maxZendeskTicketsPerSync)
	} else {
		listURL = fmt.Sprintf("%s/api/v2/tickets.json?per_page=%d", base, maxZendeskTicketsPerSync)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, listURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", zendeskBasicAuth(cfg.Email, cfg.APIToken))
	resp, err := s.HTTP.Do(req)
	if err != nil {
		s.finalizeIngestionRun(ctx, runID, "failed", 0, 0, 0, 1)
		s.completeSourceFeedSync(ctx, feedID, 1, false)
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.finalizeIngestionRun(ctx, runID, "failed", 0, 0, 0, 1)
		s.completeSourceFeedSync(ctx, feedID, 1, false)
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		s.finalizeIngestionRun(ctx, runID, "failed", 0, 0, 0, 1)
		s.completeSourceFeedSync(ctx, feedID, 1, false)
		return nil, fmt.Errorf("zendesk tickets: status %d: %s", resp.StatusCode, string(body))
	}

	var parsed struct {
		Tickets []zendeskTicket `json:"tickets"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		s.finalizeIngestionRun(ctx, runID, "failed", 0, 0, 0, 1)
		s.completeSourceFeedSync(ctx, feedID, 1, false)
		return nil, err
	}

	ingested, deduped, warnings, errs := 0, 0, 0, 0
	auth := zendeskBasicAuth(cfg.Email, cfg.APIToken)

	for _, tk := range parsed.Tickets {
		rawPayload, _ := json.Marshal(tk)
		h := hashBytes(rawPayload)
		extID := fmt.Sprintf("zendesk-ticket-%d", tk.ID)
		var rawObj map[string]any
		_ = json.Unmarshal(rawPayload, &rawObj)
		extra := map[string]any{"zendesk_ticket": rawObj}
		metaJSON, merr := s.buildRawArtifactMetadataJSON(ctx, *conn, *feed, crm_support.ArtifactKindZendeskTicket, rawPayload, extra)
		if merr != nil {
			errs++
			continue
		}
		rawID, inserted, ierr := insertRawArtifactRow(ctx, s.pool, feedID, runID, crm_support.ArtifactKindZendeskTicket, extID, h, "", metaJSON, nil, nil)
		if ierr != nil {
			errs++
			continue
		}
		if !inserted {
			deduped++
			continue
		}

		commentsPreview, _ := s.syncZendeskComments(ctx, base, auth, feedID, runID, conn, feed, tk.ID)

		var createdAt, updatedAt *time.Time
		if tk.CreatedAt != nil {
			createdAt = tk.CreatedAt
		}
		if tk.UpdatedAt != nil {
			updatedAt = tk.UpdatedAt
		}
		norm := crm_support.NormalizedSupportTicket{
			SourceFeedID:     feedID,
			ConnectorFamily:  "crm_support",
			ConnectorType:    "zendesk",
			ExternalObjectID: fmt.Sprintf("%d", tk.ID),
			ObjectType:       "ticket",
			Title:            tk.Subject,
			StageOrStatus:    tk.Status,
			RequesterRef:     fmt.Sprintf("%d", tk.RequesterID),
			CommentsPreview:  commentsPreview,
			CreatedAt:        createdAt,
			UpdatedAt:        updatedAt,
		}
		normPayload, _ := json.Marshal(norm)
		nh := hashBytes(normPayload)
		tag, qerr := s.pool.Exec(ctx, `
			INSERT INTO normalized_records (raw_artifact_id, source_feed_id, record_type, structured_payload_json, record_hash)
			VALUES ($1,$2,$3,$4,$5)
			ON CONFLICT (source_feed_id, record_hash) DO NOTHING`,
			rawID, feedID, crm_support.RecordTypeSupportTicket, normPayload, nh)
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

type zendeskTicket struct {
	ID          int64      `json:"id"`
	Subject     string     `json:"subject"`
	Status      string     `json:"status"`
	RequesterID int64      `json:"requester_id"`
	CreatedAt   *time.Time `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`
}

func (s *Service) syncZendeskComments(ctx context.Context, base, auth string, feedID, runID uuid.UUID, conn *Connector, feed *SourceFeed, ticketID int64) (preview string, ids []string) {
	u := fmt.Sprintf("%s/api/v2/tickets/%d/comments.json", base, ticketID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", nil
	}
	req.Header.Set("Authorization", auth)
	resp, err := s.HTTP.Do(req)
	if err != nil {
		return "", nil
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil || resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", nil
	}
	var parsed struct {
		Comments []struct {
			ID   int64  `json:"id"`
			Body string `json:"body"`
		} `json:"comments"`
	}
	if json.Unmarshal(body, &parsed) != nil {
		return "", nil
	}
	var b strings.Builder
	outIDs := make([]string, 0, len(parsed.Comments))
	for _, c := range parsed.Comments {
		cPayload, _ := json.Marshal(c)
		h := hashBytes(cPayload)
		extID := fmt.Sprintf("zendesk-comment-%d-%d", ticketID, c.ID)
		var rawObj map[string]any
		_ = json.Unmarshal(cPayload, &rawObj)
		extra := map[string]any{"zendesk_comment": rawObj, "ticket_id": ticketID}
		metaJSON, merr := s.buildRawArtifactMetadataJSON(ctx, *conn, *feed, crm_support.ArtifactKindZendeskComment, cPayload, extra)
		if merr == nil {
			_, _, _ = insertRawArtifactRow(ctx, s.pool, feedID, runID, crm_support.ArtifactKindZendeskComment, extID, h, "", metaJSON, nil, nil)
		}
		outIDs = append(outIDs, fmt.Sprintf("%d", c.ID))
		if b.Len() < 500 && c.Body != "" {
			frag := strings.TrimSpace(c.Body)
			if len(frag) > 200 {
				frag = frag[:200] + "…"
			}
			if b.Len() > 0 {
				b.WriteString(" | ")
			}
			b.WriteString(frag)
		}
	}
	return b.String(), outIDs
}
