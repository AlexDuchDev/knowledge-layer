package ingestion_connectors

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"

	"github.com/knowledgelayer/api/internal/ingestion_connectors/families/meeting"
)

type googleCalendarFeedConfig struct {
	ServiceAccount json.RawMessage `json:"service_account"`
}

// ValidateGoogleCalendarSourceFeedForActivation validates service account + calendar id.
func ValidateGoogleCalendarSourceFeedForActivation(feed *SourceFeed) error {
	if feed == nil {
		return errors.New("google_calendar: nil feed")
	}
	_, err := parseGoogleCalendarFeedConfig(feed.ConnectorConfigJSON)
	if err != nil {
		return err
	}
	if strings.TrimSpace(feed.ExternalRef) == "" {
		return errors.New("google_calendar: external_ref required (calendar id)")
	}
	return nil
}

func parseGoogleCalendarFeedConfig(raw json.RawMessage) (*googleCalendarFeedConfig, error) {
	var c googleCalendarFeedConfig
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, err
	}
	if len(c.ServiceAccount) == 0 {
		return nil, errors.New("google_calendar: service_account required in connector_config_json")
	}
	return &c, nil
}

// SyncGoogleCalendar lists recent events for context linking (no transcript body).
func (s *Service) SyncGoogleCalendar(ctx context.Context, feedID uuid.UUID) (*IngestionRun, error) {
	feed, err := s.GetSourceFeed(ctx, feedID)
	if err != nil {
		return nil, err
	}
	conn, err := s.GetConnector(ctx, feed.ConnectorID)
	if err != nil {
		return nil, err
	}
	if conn.Type != "google_calendar" {
		return nil, fmt.Errorf("connector is %s, not google_calendar", conn.Type)
	}
	if feed.Status != "active" {
		return nil, errors.New("source feed must be active")
	}
	cfg, err := parseGoogleCalendarFeedConfig(feed.ConnectorConfigJSON)
	if err != nil {
		return nil, err
	}
	calID := strings.TrimSpace(feed.ExternalRef)
	if calID == "" {
		return nil, errors.New("google_calendar: external_ref required (calendar id)")
	}

	creds, err := google.CredentialsFromJSON(ctx, cfg.ServiceAccount, calendar.CalendarReadonlyScope)
	if err != nil {
		return nil, fmt.Errorf("google calendar credentials: %w", err)
	}
	client := oauth2.NewClient(ctx, creds.TokenSource)
	svc, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	runID, err := s.startIngestionRun(ctx, feedID)
	if err != nil {
		return nil, err
	}

	tMin := time.Now().UTC().Add(-168 * time.Hour).Format(time.RFC3339)
	evs, err := svc.Events.List(calID).TimeMin(tMin).MaxResults(25).SingleEvents(true).OrderBy("startTime").Context(ctx).Do()
	if err != nil {
		s.finalizeIngestionRun(ctx, runID, "failed", 0, 0, 0, 1)
		s.completeSourceFeedSync(ctx, feedID, 1, false)
		return nil, err
	}

	ingested := 0
	deduped := 0
	warnings := 0
	errs := 0

	for _, e := range evs.Items {
		if e == nil || e.Id == "" {
			continue
		}
		rawPayload, _ := json.Marshal(e)
		h := hashBytes(rawPayload)
		extID := "gcal-" + e.Id
		var rawObj map[string]any
		_ = json.Unmarshal(rawPayload, &rawObj)
		extra := map[string]any{"google_calendar_event": rawObj}
		metaJSON, merr := s.buildRawArtifactMetadataJSON(ctx, *conn, *feed, meeting.ArtifactKindCalEvent, rawPayload, extra)
		if merr != nil {
			errs++
			continue
		}
		rawID, inserted, ierr := insertRawArtifactRow(ctx, s.pool, feedID, runID, "google_calendar_event", extID, h, "", metaJSON, nil, nil)
		if ierr != nil {
			errs++
			continue
		}
		if !inserted {
			deduped++
			continue
		}
		start, end := gcalEventTimes(e)
		norm := meeting.NormalizedCalendarEvent{
			SourceFeedID:    feedID,
			ConnectorFamily: "meeting",
			ConnectorType:   "google_calendar",
			ExternalRef:     e.Id,
			Summary:         e.Summary,
			StartAt:         start,
			EndAt:           end,
			HTMLLink:        e.HtmlLink,
		}
		normPayload, _ := json.Marshal(norm)
		nh := hashBytes(normPayload)
		tag, qerr := s.pool.Exec(ctx, `
			INSERT INTO normalized_records (raw_artifact_id, source_feed_id, record_type, structured_payload_json, record_hash)
			VALUES ($1,$2,$3,$4,$5)
			ON CONFLICT (source_feed_id, record_hash) DO NOTHING`,
			rawID, feedID, meeting.RecordTypeCalendarEvent, normPayload, nh)
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

func gcalEventTimes(e *calendar.Event) (*time.Time, *time.Time) {
	var start, end *time.Time
	if e.Start != nil && e.Start.DateTime != "" {
		if t, err := time.Parse(time.RFC3339, e.Start.DateTime); err == nil {
			start = &t
		}
	} else if e.Start != nil && e.Start.Date != "" {
		if t, err := time.Parse("2006-01-02", e.Start.Date); err == nil {
			start = &t
		}
	}
	if e.End != nil && e.End.DateTime != "" {
		if t, err := time.Parse(time.RFC3339, e.End.DateTime); err == nil {
			end = &t
		}
	} else if e.End != nil && e.End.Date != "" {
		if t, err := time.Parse("2006-01-02", e.End.Date); err == nil {
			end = &t
		}
	}
	return start, end
}
