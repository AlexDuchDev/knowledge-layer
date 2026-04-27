package ingestion_connectors

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/knowledgelayer/api/internal/ingestion_connectors/families/crm_support"
)

const maxHubSpotObjectsPerSync = 25

type hubspotFeedConfig struct {
	Token    string `json:"hubspot_private_app_token"`
	TokenAlt string `json:"hubspot_access_token"`
	FeedKind string `json:"hubspot_feed_kind"` // contacts | companies | deals
}

func parseHubSpotFeedConfig(raw json.RawMessage) (*hubspotFeedConfig, error) {
	var c hubspotFeedConfig
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, err
	}
	if strings.TrimSpace(c.Token) == "" {
		c.Token = strings.TrimSpace(c.TokenAlt)
	}
	c.Token = strings.TrimSpace(c.Token)
	c.FeedKind = strings.ToLower(strings.TrimSpace(c.FeedKind))
	if c.Token == "" {
		return nil, errors.New("hubspot: hubspot_private_app_token or hubspot_access_token required")
	}
	switch c.FeedKind {
	case "contacts", "companies", "deals":
	default:
		return nil, errors.New("hubspot: hubspot_feed_kind must be contacts, companies, or deals")
	}
	return &c, nil
}

// ValidateHubSpotSourceFeedForActivation validates token and feed kind for active feeds.
func ValidateHubSpotSourceFeedForActivation(feed *SourceFeed) error {
	if feed == nil {
		return errors.New("hubspot: nil feed")
	}
	_, err := parseHubSpotFeedConfig(feed.ConnectorConfigJSON)
	return err
}

func hubspotObjectPath(kind string) (path, props string) {
	switch kind {
	case "contacts":
		return "/crm/v3/objects/contacts", "firstname,lastname,email,hs_lastmodifieddate,createdate"
	case "companies":
		return "/crm/v3/objects/companies", "name,domain,hs_lastmodifieddate,createdate"
	case "deals":
		return "/crm/v3/objects/deals", "dealname,dealstage,pipeline,hs_lastmodifieddate,createdate"
	default:
		return "", ""
	}
}

func hubspotDisplayTitle(kind string, props map[string]string) string {
	switch kind {
	case "contacts":
		fn := props["firstname"]
		ln := props["lastname"]
		em := props["email"]
		s := strings.TrimSpace(fn + " " + ln)
		if s != "" {
			return s
		}
		return em
	case "companies":
		if n := props["name"]; n != "" {
			return n
		}
		return props["domain"]
	case "deals":
		return props["dealname"]
	}
	return ""
}

// SyncHubSpot lists CRM objects for the configured kind (v1 cap 25).
func (s *Service) SyncHubSpot(ctx context.Context, feedID uuid.UUID) (*IngestionRun, error) {
	feed, err := s.GetSourceFeed(ctx, feedID)
	if err != nil {
		return nil, err
	}
	conn, err := s.GetConnector(ctx, feed.ConnectorID)
	if err != nil {
		return nil, err
	}
	if conn.Type != "hubspot" {
		return nil, fmt.Errorf("connector is %s, not hubspot", conn.Type)
	}
	if feed.Status != "active" {
		return nil, errors.New("source feed must be active")
	}
	cfg, err := parseHubSpotFeedConfig(feed.ConnectorConfigJSON)
	if err != nil {
		return nil, err
	}

	runID, err := s.startIngestionRun(ctx, feedID)
	if err != nil {
		return nil, err
	}

	path, props := hubspotObjectPath(cfg.FeedKind)
	u := fmt.Sprintf("https://api.hubapi.com%s?limit=%d&properties=%s", path, maxHubSpotObjectsPerSync, props)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.Token)
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
		return nil, fmt.Errorf("hubspot: status %d: %s", resp.StatusCode, string(body))
	}

	var parsed struct {
		Results []struct {
			ID         string            `json:"id"`
			Properties map[string]string `json:"properties"`
			CreatedAt  string            `json:"createdAt"`
			UpdatedAt  string            `json:"updatedAt"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		s.finalizeIngestionRun(ctx, runID, "failed", 0, 0, 0, 1)
		s.completeSourceFeedSync(ctx, feedID, 1, false)
		return nil, err
	}

	ingested, deduped, warnings, errs := 0, 0, 0, 0
	for _, row := range parsed.Results {
		rawPayload, _ := json.Marshal(row)
		h := hashBytes(rawPayload)
		extID := fmt.Sprintf("hubspot-%s-%s", cfg.FeedKind, row.ID)
		var rawObj map[string]any
		_ = json.Unmarshal(rawPayload, &rawObj)
		extra := map[string]any{"hubspot_object": rawObj}
		metaJSON, merr := s.buildRawArtifactMetadataJSON(ctx, *conn, *feed, crm_support.ArtifactKindHubSpotObject, rawPayload, extra)
		if merr != nil {
			errs++
			continue
		}
		rawID, inserted, ierr := insertRawArtifactRow(ctx, s.pool, feedID, runID, crm_support.ArtifactKindHubSpotObject, extID, h, "", metaJSON, nil, nil)
		if ierr != nil {
			errs++
			continue
		}
		if !inserted {
			deduped++
			continue
		}
		title := hubspotDisplayTitle(cfg.FeedKind, row.Properties)
		stage := ""
		switch cfg.FeedKind {
		case "deals":
			stage = row.Properties["dealstage"]
		}
		var createdAt, updatedAt *time.Time
		if row.CreatedAt != "" {
			if t, e := time.Parse(time.RFC3339, row.CreatedAt); e == nil {
				createdAt = &t
			}
		}
		if row.UpdatedAt != "" {
			if t, e := time.Parse(time.RFC3339, row.UpdatedAt); e == nil {
				updatedAt = &t
			}
		}
		if createdAt == nil && row.Properties["createdate"] != "" {
			if ms, e := parseHubSpotMillis(row.Properties["createdate"]); e == nil {
				createdAt = &ms
			}
		}
		if updatedAt == nil && row.Properties["hs_lastmodifieddate"] != "" {
			if ms, e := parseHubSpotMillis(row.Properties["hs_lastmodifieddate"]); e == nil {
				updatedAt = &ms
			}
		}
		customerRef := row.Properties["email"]
		if cfg.FeedKind == "companies" {
			customerRef = row.Properties["domain"]
		}
		norm := crm_support.NormalizedCRMRecord{
			SourceFeedID:         feedID,
			ConnectorFamily:      "crm_support",
			ConnectorType:        "hubspot",
			ExternalObjectID:     row.ID,
			ObjectType:           cfg.FeedKind,
			TitleOrDisplayName:   title,
			StageOrStatus:        stage,
			CreatedAt:            createdAt,
			UpdatedAt:            updatedAt,
			CustomerOrCompanyRef: customerRef,
		}
		normPayload, _ := json.Marshal(norm)
		nh := hashBytes(normPayload)
		tag, qerr := s.pool.Exec(ctx, `
			INSERT INTO normalized_records (raw_artifact_id, source_feed_id, record_type, structured_payload_json, record_hash)
			VALUES ($1,$2,$3,$4,$5)
			ON CONFLICT (source_feed_id, record_hash) DO NOTHING`,
			rawID, feedID, crm_support.RecordTypeCRMRecord, normPayload, nh)
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

func parseHubSpotMillis(s string) (time.Time, error) {
	ms, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	sec := ms / 1000
	nsec := (ms % 1000) * 1e6
	return time.Unix(sec, nsec).UTC(), nil
}
