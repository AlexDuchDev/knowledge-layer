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
	"time"

	"github.com/google/uuid"

	"github.com/knowledgelayer/api/internal/ingestion_connectors/families/docs_wiki"
)

const maxConfluencePagesPerSync = 25

// SyncConfluence ingests Confluence Cloud pages via REST (v1).
func (s *Service) SyncConfluence(ctx context.Context, feedID uuid.UUID) (*IngestionRun, error) {
	feed, err := s.GetSourceFeed(ctx, feedID)
	if err != nil {
		return nil, err
	}
	conn, err := s.GetConnector(ctx, feed.ConnectorID)
	if err != nil {
		return nil, err
	}
	if conn.Type != "confluence" {
		return nil, fmt.Errorf("connector is %s, not confluence", conn.Type)
	}
	if feed.Status != "active" {
		return nil, errors.New("source feed must be active")
	}
	cfg, err := parseConfluenceFeedConfig(feed.ConnectorConfigJSON)
	if err != nil {
		return nil, err
	}
	if err := ValidateConfluenceSourceFeedForActivation(feed); err != nil {
		return nil, err
	}

	runID, err := s.startIngestionRun(ctx, feedID)
	if err != nil {
		return nil, err
	}

	ingested, deduped, warnings, errs := 0, 0, 0, 0
	switch cfg.FeedKind {
	case "space":
		n, d, w, e := s.syncConfluenceSpace(ctx, cfg, feed, conn, feedID, runID)
		ingested, deduped, warnings, errs = n, d, w, e
	case "page_collection":
		for _, pid := range splitCommaIDs(feed.ExternalRef) {
			n, d, e := s.syncConfluenceSinglePage(ctx, cfg, feed, conn, feedID, runID, pid, "")
			ingested += n
			deduped += d
			errs += e
		}
	case "content_tree":
		n, d, w, e := s.syncConfluenceContentTree(ctx, cfg, feed, conn, feedID, runID, strings.TrimSpace(feed.ExternalRef))
		ingested, deduped, warnings, errs = n, d, w, e
	default:
		errs++
	}

	status := syncRunStatusFromCounts(ingested, errs)
	s.finalizeIngestionRun(ctx, runID, status, ingested, deduped, warnings, errs)
	s.completeSourceFeedSync(ctx, feedID, errs, false)
	return s.GetIngestionRun(ctx, runID)
}

func (s *Service) confluenceGET(ctx context.Context, fullURL, token string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	resp, err := s.HTTP.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	return body, resp.StatusCode, err
}

func (s *Service) syncConfluenceSpace(ctx context.Context, cfg *confluenceFeedConfig, feed *SourceFeed, conn *Connector, feedID, runID uuid.UUID) (ingested, deduped, warnings, errs int) {
	spaceKey := url.QueryEscape(strings.TrimSpace(feed.ExternalRef))
	u := fmt.Sprintf("%s/rest/api/content?spaceKey=%s&type=page&limit=%d&expand=body.storage,version,history.lastUpdated,metadata.labels,ancestors",
		cfg.BaseURL, spaceKey, maxConfluencePagesPerSync)
	body, code, err := s.confluenceGET(ctx, u, cfg.Auth)
	if err != nil {
		return 0, 0, 0, 1
	}
	if code < 200 || code >= 300 {
		return 0, 0, 0, 1
	}
	var parsed struct {
		Results []json.RawMessage `json:"results"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return 0, 0, 0, 1
	}
	for _, raw := range parsed.Results {
		n, d, e := s.confluencePersistPage(ctx, cfg.BaseURL, feed, conn, feedID, runID, raw, strings.TrimSpace(feed.ExternalRef))
		ingested += n
		deduped += d
		errs += e
	}
	return ingested, deduped, warnings, errs
}

func (s *Service) syncConfluenceSinglePage(ctx context.Context, cfg *confluenceFeedConfig, feed *SourceFeed, conn *Connector, feedID, runID uuid.UUID, pageID, spaceHint string) (ingested, deduped, errs int) {
	u := fmt.Sprintf("%s/rest/api/content/%s?expand=body.storage,version,history.lastUpdated,metadata.labels,ancestors",
		cfg.BaseURL, url.PathEscape(pageID))
	body, code, err := s.confluenceGET(ctx, u, cfg.Auth)
	if err != nil || code < 200 || code >= 300 {
		return 0, 0, 1
	}
	return s.confluencePersistPage(ctx, cfg.BaseURL, feed, conn, feedID, runID, body, spaceHint)
}

func (s *Service) syncConfluenceContentTree(ctx context.Context, cfg *confluenceFeedConfig, feed *SourceFeed, conn *Connector, feedID, runID uuid.UUID, rootID string) (ingested, deduped, warnings, errs int) {
	queue := []string{rootID}
	seen := map[string]struct{}{}
	count := 0
	for len(queue) > 0 && count < maxConfluencePagesPerSync {
		id := queue[0]
		queue = queue[1:]
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		n, d, e := s.syncConfluenceSinglePage(ctx, cfg, feed, conn, feedID, runID, id, "")
		ingested += n
		deduped += d
		errs += e
		if n > 0 {
			count++
		}
		chURL := fmt.Sprintf("%s/rest/api/content/%s/child/page?limit=25", cfg.BaseURL, url.PathEscape(id))
		chBody, code, err := s.confluenceGET(ctx, chURL, cfg.Auth)
		if err != nil || code < 200 || code >= 300 {
			continue
		}
		var chParsed struct {
			Results []struct {
				ID string `json:"id"`
			} `json:"results"`
		}
		if json.Unmarshal(chBody, &chParsed) != nil {
			continue
		}
		childPayload, _ := json.Marshal(chParsed)
		h := hashBytes(childPayload)
		extID := "confluence-children-" + id
		var rawObj map[string]any
		_ = json.Unmarshal(childPayload, &rawObj)
		extra := map[string]any{"confluence_child_pages": rawObj}
		metaJSON, merr := s.buildRawArtifactMetadataJSON(ctx, *conn, *feed, docs_wiki.ArtifactConfluenceChildPages, childPayload, extra)
		if merr == nil {
			_, _, _ = insertRawArtifactRow(ctx, s.pool, feedID, runID, docs_wiki.ArtifactConfluenceChildPages, extID, h, "", metaJSON, nil, nil)
		}
		for _, c := range chParsed.Results {
			if c.ID != "" {
				queue = append(queue, c.ID)
			}
		}
	}
	if count >= maxConfluencePagesPerSync {
		warnings++
	}
	return ingested, deduped, warnings, errs
}

type confluencePageParsed struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Title string `json:"title"`
	Body  struct {
		Storage struct {
			Value          string `json:"value"`
			Representation string `json:"representation"`
		} `json:"storage"`
	} `json:"body"`
	Version struct {
		When string `json:"when"`
	} `json:"version"`
	History struct {
		LastUpdated struct {
			When string `json:"when"`
			By   struct {
				Username    string `json:"username"`
				DisplayName string `json:"displayName"`
			} `json:"by"`
		} `json:"lastUpdated"`
	} `json:"history"`
	Metadata struct {
		Labels struct {
			Results []struct {
				Name string `json:"name"`
			} `json:"results"`
		} `json:"labels"`
	} `json:"metadata"`
	Ancestors []struct {
		ID string `json:"id"`
	} `json:"ancestors"`
	Links struct {
		WebUI string `json:"webui"`
	} `json:"_links"`
}

func (s *Service) confluencePersistPage(ctx context.Context, baseURL string, feed *SourceFeed, conn *Connector, feedID, runID uuid.UUID, rawPage []byte, spaceRef string) (ingested, deduped, errs int) {
	var doc confluencePageParsed
	if err := json.Unmarshal(rawPage, &doc); err != nil || doc.ID == "" {
		return 0, 0, 1
	}
	pageTitle := doc.Title
	h := hashBytes(rawPage)
	extID := "confluence-page-" + doc.ID
	var rawObj map[string]any
	_ = json.Unmarshal(rawPage, &rawObj)
	extra := map[string]any{"confluence_page": rawObj}
	metaPage, merr := s.buildRawArtifactMetadataJSON(ctx, *conn, *feed, docs_wiki.ArtifactConfluencePage, rawPage, extra)
	if merr != nil {
		return 0, 0, 1
	}
	rawPageID, inserted, ierr := insertRawArtifactRow(ctx, s.pool, feedID, runID, docs_wiki.ArtifactConfluencePage, extID, h, "", metaPage, nil, nil)
	if ierr != nil {
		return 0, 0, 1
	}
	if !inserted {
		return 0, 1, 0
	}

	bodyObj := map[string]any{
		"representation": doc.Body.Storage.Representation,
		"value":          doc.Body.Storage.Value,
	}
	bodyPayload, _ := json.Marshal(bodyObj)
	bh := hashBytes(bodyPayload)
	metaBody, _ := s.buildRawArtifactMetadataJSON(ctx, *conn, *feed, docs_wiki.ArtifactConfluencePageBody, bodyPayload, map[string]any{"confluence_page_body": bodyObj})
	_, _, _ = insertRawArtifactRow(ctx, s.pool, feedID, runID, docs_wiki.ArtifactConfluencePageBody, extID+"-body", bh, "", metaBody, nil, nil)

	metaMap := map[string]any{
		"version_when": doc.Version.When,
		"labels":       doc.Metadata.Labels.Results,
		"ancestor_ids": ancestorIDs(doc.Ancestors),
	}
	metaPayload, _ := json.Marshal(metaMap)
	mh := hashBytes(metaPayload)
	metaMeta, _ := s.buildRawArtifactMetadataJSON(ctx, *conn, *feed, docs_wiki.ArtifactConfluencePageMetadata, metaPayload, map[string]any{"confluence_page_metadata": metaMap})
	_, _, _ = insertRawArtifactRow(ctx, s.pool, feedID, runID, docs_wiki.ArtifactConfluencePageMetadata, extID+"-meta", mh, "", metaMeta, nil, nil)

	var mod *time.Time
	if t := firstNonEmptyTime(doc.Version.When, doc.History.LastUpdated.When); t != "" {
		if parsed, err := parseConfluenceTime(t); err == nil {
			mod = &parsed
		}
	}
	parentRef := ""
	parentRefs := ancestorIDs(doc.Ancestors)
	if len(parentRefs) > 0 {
		parentRef = parentRefs[len(parentRefs)-1]
	}
	labels := make([]string, 0, len(doc.Metadata.Labels.Results))
	for _, l := range doc.Metadata.Labels.Results {
		if l.Name != "" {
			labels = append(labels, l.Name)
		}
	}
	editor := strings.TrimSpace(doc.History.LastUpdated.By.DisplayName)
	if editor == "" {
		editor = strings.TrimSpace(doc.History.LastUpdated.By.Username)
	}
	web := strings.TrimSuffix(baseURL, "/")
	if doc.Links.WebUI != "" {
		if strings.HasPrefix(doc.Links.WebUI, "http") {
			web = doc.Links.WebUI
		} else {
			web = web + doc.Links.WebUI
		}
	}

	norm := docs_wiki.NormalizedDocPage{
		SourceFeedID:    feedID,
		ConnectorFamily: "docs_wiki",
		ConnectorType:   "confluence",
		Title:           pageTitle,
		ExternalRef:     doc.ID,
		ExternalDocID:   doc.ID,
		ParentRef:       parentRef,
		ParentRefs:      parentRefs,
		SpaceRef:        spaceRef,
		LastModifiedAt:  mod,
		EditorRef:       editor,
		Labels:          labels,
		BodyText:        doc.Body.Storage.Value,
		WebViewLink:     web,
		DownstreamHint: map[string]any{
			"suggested_entity_type": "ReferenceDocument",
			"default_truth_mode":    "mirrored_authority",
		},
	}
	normPayload, _ := json.Marshal(norm)
	nh := hashBytes(normPayload)
	tag, qerr := s.pool.Exec(ctx, `
		INSERT INTO normalized_records (raw_artifact_id, source_feed_id, record_type, structured_payload_json, record_hash)
		VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT (source_feed_id, record_hash) DO NOTHING`,
		rawPageID, feedID, docs_wiki.RecordTypeDocsPage, normPayload, nh)
	if qerr != nil {
		return 0, 0, 1
	}
	if tag.RowsAffected() == 0 {
		return 0, 1, 0
	}
	return 1, 0, 0
}

func ancestorIDs(as []struct {
	ID string `json:"id"`
}) []string {
	out := make([]string, 0, len(as))
	for _, a := range as {
		if a.ID != "" {
			out = append(out, a.ID)
		}
	}
	return out
}

func firstNonEmptyTime(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return strings.TrimSpace(a)
	}
	return strings.TrimSpace(b)
}

func parseConfluenceTime(s string) (time.Time, error) {
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.000-07:00",
		"2006-01-02T15:04:05.000Z07:00",
	}
	var last error
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), nil
		} else {
			last = err
		}
	}
	return time.Time{}, last
}
