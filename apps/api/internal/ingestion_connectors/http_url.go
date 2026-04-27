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
	"golang.org/x/net/html"

	"github.com/knowledgelayer/api/internal/ingestion_connectors/families/docs_wiki"
)

const artifactHTTPURLPage = "http_url_page"

// SyncHTTPURL ingests a single URL per source feed: raw artifact + normalized record.
func (s *Service) SyncHTTPURL(ctx context.Context, feedID uuid.UUID) (*IngestionRun, error) {
	feed, err := s.GetSourceFeed(ctx, feedID)
	if err != nil {
		return nil, err
	}
	conn, err := s.GetConnector(ctx, feed.ConnectorID)
	if err != nil {
		return nil, err
	}
	if conn.Type != "http_url" {
		return nil, fmt.Errorf("connector is %s, not http_url", conn.Type)
	}
	if feed.Status != "active" {
		return nil, errors.New("source feed must be active")
	}
	url := strings.TrimSpace(feed.ExternalRef)
	if url == "" {
		return nil, errors.New("source_feeds.external_ref must be a URL")
	}

	runID, err := s.startIngestionRun(ctx, feedID)
	if err != nil {
		return nil, err
	}

	ingested, deduped, warnings, errs := 0, 0, 0, 0

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		errs++
		s.finalizeIngestionRun(ctx, runID, "failed", ingested, deduped, warnings, errs)
		s.completeSourceFeedSync(ctx, feedID, errs, false)
		return nil, err
	}
	req.Header.Set("User-Agent", "KnowledgeLayerBot/1.0")
	resp, err := s.HTTP.Do(req)
	if err != nil {
		errs++
		s.finalizeIngestionRun(ctx, runID, "failed", ingested, deduped, warnings, errs)
		s.completeSourceFeedSync(ctx, feedID, errs, false)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		errs++
		s.finalizeIngestionRun(ctx, runID, "failed", ingested, deduped, warnings, errs)
		s.completeSourceFeedSync(ctx, feedID, errs, false)
		return nil, err
	}

	text, title := extractHTMLTextAndTitle(body)
	if strings.TrimSpace(text) == "" {
		warnings++
	}

	extra := map[string]any{
		"http_url_page": map[string]any{
			"url":          url,
			"http_status":  resp.StatusCode,
			"content_text": text,
			"title":        title,
		},
	}
	metaJSON, err := s.buildRawArtifactMetadataJSON(ctx, *conn, *feed, artifactHTTPURLPage, body, extra)
	if err != nil {
		errs++
		s.finalizeIngestionRun(ctx, runID, "failed", ingested, deduped, warnings, errs)
		s.completeSourceFeedSync(ctx, feedID, errs, false)
		return nil, err
	}

	h := hashBytes(metaJSON)
	rawID, inserted, err := insertRawArtifactRow(ctx, s.pool, feedID, runID, artifactHTTPURLPage, url, h, "", metaJSON, timePtr(time.Now().UTC()), nil)
	if err != nil {
		errs++
		s.finalizeIngestionRun(ctx, runID, "failed", ingested, deduped, warnings, errs)
		s.completeSourceFeedSync(ctx, feedID, errs, false)
		return nil, err
	}
	if !inserted {
		deduped++
		status := syncRunStatusFromCounts(ingested, errs)
		s.finalizeIngestionRun(ctx, runID, status, ingested, deduped, warnings, errs)
		s.completeSourceFeedSync(ctx, feedID, errs, false)
		return s.GetIngestionRun(ctx, runID)
	}

	norm := docs_wiki.NormalizedDocPage{
		SourceFeedID:    feedID,
		ConnectorFamily: "docs_wiki",
		ConnectorType:   "http_url",
		Title:           firstNonEmptyString(title, url),
		ExternalRef:     url,
		LastModifiedAt:  timePtr(time.Now().UTC()),
		BodyText:        text,
		WebViewLink:     url,
		DownstreamHint: map[string]any{
			"suggested_entity_type": "ReferenceDocument",
			"default_truth_mode":    "mirrored_authority",
		},
	}
	normPayload, err := json.Marshal(norm)
	if err != nil {
		errs++
		s.finalizeIngestionRun(ctx, runID, "failed", ingested, deduped, warnings, errs)
		s.completeSourceFeedSync(ctx, feedID, errs, false)
		return nil, err
	}
	nh := hashBytes(normPayload)
	tag, err := s.pool.Exec(ctx, `
		INSERT INTO normalized_records (raw_artifact_id, source_feed_id, record_type, structured_payload_json, record_hash, source_timestamp, detected_author_ref, normalization_version)
		VALUES ($1,$2,$3,$4,$5,$6,$7,1)
		ON CONFLICT (source_feed_id, record_hash) DO NOTHING`,
		rawID, feedID, docs_wiki.RecordTypeDocsPage, normPayload, nh, timePtr(time.Now().UTC()), nil)
	if err != nil {
		errs++
	} else if tag.RowsAffected() == 0 {
		deduped++
	} else {
		ingested++
	}

	status := syncRunStatusFromCounts(ingested, errs)
	s.finalizeIngestionRun(ctx, runID, status, ingested, deduped, warnings, errs)
	s.completeSourceFeedSync(ctx, feedID, errs, false)
	return s.GetIngestionRun(ctx, runID)
}

func timePtr(t time.Time) *time.Time { return &t }

func extractHTMLTextAndTitle(b []byte) (text string, title string) {
	n, err := html.Parse(bytes.NewReader(b))
	if err != nil {
		return string(b), ""
	}
	var sb strings.Builder
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "title" && node.FirstChild != nil {
			title = strings.TrimSpace(node.FirstChild.Data)
		}
		if node.Type == html.TextNode {
			t := strings.TrimSpace(node.Data)
			if t != "" {
				sb.WriteString(t)
				sb.WriteString("\n")
			}
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			// Skip script/style tags entirely.
			if c.Type == html.ElementNode && (c.Data == "script" || c.Data == "style") {
				continue
			}
			walk(c)
		}
	}
	walk(n)
	out := strings.TrimSpace(sb.String())
	if out == "" {
		return string(b), title
	}
	return out, title
}
