package ingestion_connectors

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/knowledgelayer/api/internal/ingestion_connectors/families/docs_wiki"
)

// SyncNotion ingests a Notion page or database (v1: shallow database query + per-page blocks).
func (s *Service) SyncNotion(ctx context.Context, feedID uuid.UUID) (*IngestionRun, error) {
	feed, err := s.GetSourceFeed(ctx, feedID)
	if err != nil {
		return nil, err
	}
	conn, err := s.GetConnector(ctx, feed.ConnectorID)
	if err != nil {
		return nil, err
	}
	if conn.Type != "notion" {
		return nil, fmt.Errorf("connector is %s, not notion", conn.Type)
	}
	if feed.Status != "active" {
		return nil, errors.New("source feed must be active")
	}

	token, err := docs_wiki.RequireNotionToken(feed.ConnectorConfigJSON)
	if err != nil {
		return nil, err
	}
	scopeCfg, err := docs_wiki.ParseNotionConfig(feed.ConnectorConfigJSON)
	if err != nil {
		return nil, err
	}
	if err := ValidateNotionV1ForActivation(feed, token, scopeCfg); err != nil {
		return nil, err
	}

	runID, err := s.startIngestionRun(ctx, feedID)
	if err != nil {
		return nil, err
	}

	ingested := 0
	deduped := 0
	warnings := 0
	errs := 0

	scope := strings.ToLower(strings.TrimSpace(scopeCfg.Scope))
	switch scope {
	case "page":
		n, d, e := s.syncNotionPage(ctx, token, feedID, runID, conn, feed, feed.ExternalRef)
		ingested += n
		deduped += d
		errs += e
	case "database":
		pageIDs, qerr := s.notionDatabaseQueryAllPageIDs(ctx, token, feed.ExternalRef)
		if qerr != nil {
			errs++
		} else {
			const maxPages = 25
			for i, pid := range pageIDs {
				if i >= maxPages {
					warnings++
					break
				}
				n, d, e := s.syncNotionPage(ctx, token, feedID, runID, conn, feed, pid)
				ingested += n
				deduped += d
				errs += e
			}
		}
	default:
		errs++
	}

	status := syncRunStatusFromCounts(ingested, errs)
	s.finalizeIngestionRun(ctx, runID, status, ingested, deduped, warnings, errs)
	s.completeSourceFeedSync(ctx, feedID, errs, false)

	return s.GetIngestionRun(ctx, runID)
}

func (s *Service) syncNotionPage(ctx context.Context, token string, feedID, runID uuid.UUID, conn *Connector, feed *SourceFeed, pageID string) (ingested, deduped, errs int) {
	pageBody, err := s.notionGET(ctx, token, "https://api.notion.com/v1/pages/"+strings.TrimSpace(pageID))
	if err != nil {
		return 0, 0, 1
	}
	var props notionPageProperties
	if err := json.Unmarshal(pageBody, &props); err != nil {
		return 0, 0, 1
	}
	title := notionExtractTitle(props)
	if title == "" {
		title = "Untitled"
	}

	var textParts []string
	cursor := ""
	for i := 0; i < 50; i++ {
		u := fmt.Sprintf("https://api.notion.com/v1/blocks/%s/children?page_size=100", strings.TrimSpace(pageID))
		if cursor != "" {
			u += "&start_cursor=" + cursor
		}
		blockBody, err := s.notionGET(ctx, token, u)
		if err != nil {
			errs++
			return ingested, deduped, errs
		}
		var lb notionListBlocksResp
		if err := json.Unmarshal(blockBody, &lb); err != nil {
			errs++
			return ingested, deduped, errs
		}
		for _, raw := range lb.Results {
			for _, line := range notionBlockPlainLines(raw) {
				textParts = append(textParts, line)
			}
		}
		cursor = lb.NextCursor
		if !lb.HasMore || cursor == "" {
			break
		}
	}
	body := strings.Join(textParts, "\n")

	rawPayload, err := json.Marshal(map[string]any{
		"page_id": pageID,
		"title":   title,
		"body":    body,
	})
	if err != nil {
		errs++
		return ingested, deduped, errs
	}
	h := hashBytes(rawPayload)
	extID := "notion-" + pageID
	var rawObj map[string]any
	_ = json.Unmarshal(rawPayload, &rawObj)
	extra := map[string]any{"notion_page": rawObj}
	metaJSON, err := s.buildRawArtifactMetadataJSON(ctx, *conn, *feed, docs_wiki.ArtifactKindPage, rawPayload, extra)
	if err != nil {
		errs++
		return ingested, deduped, errs
	}
	rawID, inserted, err := insertRawArtifactRow(ctx, s.pool, feedID, runID, "notion_page", extID, h, "", metaJSON, nil, nil)
	if err != nil {
		errs++
		return ingested, deduped, errs
	}
	if !inserted {
		deduped++
		return ingested, deduped, errs
	}

	now := time.Now().UTC()
	norm := docs_wiki.NormalizedDocPage{
		SourceFeedID:    feedID,
		ConnectorFamily: "docs_wiki",
		ConnectorType:   "notion",
		Title:           title,
		ExternalRef:     pageID,
		LastModifiedAt:  &now,
		BodyText:        body,
		DownstreamHint: map[string]any{
			"suggested_entity_type": "ReferenceDocument",
		},
	}
	normPayload, err := json.Marshal(norm)
	if err != nil {
		errs++
		return ingested, deduped, errs
	}
	nh := hashBytes(normPayload)
	tag, err := s.pool.Exec(ctx, `
		INSERT INTO normalized_records (raw_artifact_id, source_feed_id, record_type, structured_payload_json, record_hash)
		VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT (source_feed_id, record_hash) DO NOTHING`,
		rawID, feedID, docs_wiki.RecordTypeDocsPage, normPayload, nh)
	if err != nil {
		errs++
		return ingested, deduped, errs
	}
	if tag.RowsAffected() == 0 {
		deduped++
		return ingested, deduped, errs
	}
	ingested++
	return ingested, deduped, errs
}

func (s *Service) notionDatabaseQueryAllPageIDs(ctx context.Context, token, databaseID string) ([]string, error) {
	var ids []string
	cursor := ""
	for i := 0; i < 20; i++ {
		body := map[string]any{"page_size": 100}
		if cursor != "" {
			body["start_cursor"] = cursor
		}
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost,
			"https://api.notion.com/v1/databases/"+strings.TrimSpace(databaseID)+"/query",
			bytes.NewReader(raw))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Notion-Version", notionAPIVersion)
		req.Header.Set("Content-Type", "application/json")
		resp, err := s.HTTP.Do(req)
		if err != nil {
			return nil, err
		}
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("notion database query: status %d: %s", resp.StatusCode, string(b))
		}
		var dq notionDatabaseQueryResp
		if err := json.Unmarshal(b, &dq); err != nil {
			return nil, err
		}
		for _, r := range dq.Results {
			if r.ID != "" {
				ids = append(ids, r.ID)
			}
		}
		if !dq.HasMore || dq.NextCursor == "" {
			break
		}
		cursor = dq.NextCursor
	}
	return ids, nil
}

func (s *Service) notionGET(ctx context.Context, token, u string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Notion-Version", notionAPIVersion)
	resp, err := s.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("notion GET %s: status %d: %s", u, resp.StatusCode, string(b))
	}
	return b, nil
}
