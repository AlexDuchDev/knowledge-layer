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

	"github.com/knowledgelayer/api/internal/ingestion_connectors/families/microsoft365"
)

// m365DriveChildrenGraphPath returns the Graph path segment (with leading slash) for listing drive children, or an error.
func m365DriveChildrenGraphPath(externalRef string) (string, error) {
	ref := strings.TrimSpace(externalRef)
	if ref == "" {
		ref = "me|root"
	}
	parts := strings.Split(ref, "|")
	if len(parts) != 2 {
		return "", errors.New("microsoft_365 files: external_ref must be me|root or driveId|itemId or driveId|root")
	}
	a, b := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	if a == "" || b == "" {
		return "", errors.New("microsoft_365 files: external_ref segments must be non-empty")
	}
	if strings.EqualFold(a, "me") && strings.EqualFold(b, "root") {
		return "/me/drive/root/children", nil
	}
	if strings.EqualFold(b, "root") {
		return fmt.Sprintf("/drives/%s/root/children", url.PathEscape(a)), nil
	}
	return fmt.Sprintf("/drives/%s/items/%s/children", url.PathEscape(a), url.PathEscape(b)), nil
}

func (s *Service) syncM365OneDriveSharePoint(ctx context.Context, feedID, runID uuid.UUID, feed *SourceFeed, conn *Connector, cfg *m365FeedConfig) (*IngestionRun, error) {
	connectorType := cfg.Product
	scope := strings.ToLower(strings.TrimSpace(cfg.M365FilesScope))
	if scope == "" {
		scope = "folder"
	}

	var listURL string
	if scope == "search" {
		q := strings.TrimSpace(cfg.M365SearchQuery)
		maxTop := cfg.M365SearchMaxResults
		if maxTop <= 0 {
			maxTop = 10
		}
		if maxTop > m365FilesMaxItems {
			maxTop = m365FilesMaxItems
		}
		// Escape single quotes for Graph search(q='...') path segment.
		safe := strings.ReplaceAll(q, "'", "''")
		listURL = fmt.Sprintf("https://graph.microsoft.com/v1.0/me/drive/root/search(q='%s')?$top=%d", safe, maxTop)
	} else {
		path, err := m365DriveChildrenGraphPath(feed.ExternalRef)
		if err != nil {
			s.finalizeIngestionRun(ctx, runID, "failed", 0, 0, 0, 1)
			s.completeSourceFeedSync(ctx, feedID, 1, false)
			return nil, err
		}
		listURL = "https://graph.microsoft.com/v1.0" + path + fmt.Sprintf("?$top=%d", m365FilesMaxItems)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, listURL, nil)
	if err != nil {
		s.finalizeIngestionRun(ctx, runID, "failed", 0, 0, 0, 1)
		s.completeSourceFeedSync(ctx, feedID, 1, false)
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.AccessToken)
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
		return nil, fmt.Errorf("graph drive list: status %d: %s", resp.StatusCode, string(body))
	}

	var parsed struct {
		Value []m365DriveItem `json:"value"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		s.finalizeIngestionRun(ctx, runID, "failed", 0, 0, 0, 1)
		s.completeSourceFeedSync(ctx, feedID, 1, false)
		return nil, err
	}

	ingested, deduped, warnings, errs := 0, 0, 0, 0
	for _, it := range parsed.Value {
		rawPayload, _ := json.Marshal(it)
		h := hashBytes(rawPayload)
		extID := "m365-file-" + it.ID
		var rawObj map[string]any
		_ = json.Unmarshal(rawPayload, &rawObj)
		extra := map[string]any{"m365_drive_item": rawObj}
		artifactType := microsoft365.ArtifactM365FileMetadata
		if it.Folder != nil {
			artifactType = microsoft365.ArtifactM365FolderContext
		}
		metaJSON, merr := s.buildRawArtifactMetadataJSON(ctx, *conn, *feed, microsoft365.ArtifactKindGraphJSON, rawPayload, extra)
		if merr != nil {
			errs++
			continue
		}
		rawID, inserted, ierr := insertRawArtifactRow(ctx, s.pool, feedID, runID, artifactType, extID, h, "", metaJSON, nil, nil)
		if ierr != nil {
			errs++
			continue
		}
		if !inserted {
			deduped++
			continue
		}
		if it.File == nil {
			continue
		}
		var mod *time.Time
		if it.LastModifiedDateTime != "" {
			if t, err := time.Parse(time.RFC3339, it.LastModifiedDateTime); err == nil {
				mod = &t
			}
		}
		owner := strings.TrimSpace(it.CreatedBy.User.Mail)
		if owner == "" {
			owner = strings.TrimSpace(it.CreatedBy.User.Email)
		}
		pathRef := ""
		if it.ParentReference.Path != "" {
			pathRef = it.ParentReference.Path
		}
		norm := microsoft365.NormalizedCloudFile{
			SourceFeedID:       feedID,
			ConnectorFamily:    "microsoft365",
			ConnectorType:      connectorType,
			ExternalFileID:     it.ID,
			TitleOrFileName:    it.Name,
			MimeType:           it.File.MimeType,
			Owner:              owner,
			PathOrContainerRef: pathRef,
			ModifiedAt:         mod,
			StorageReference:   it.WebURL,
		}
		normPayload, _ := json.Marshal(norm)
		nh := hashBytes(normPayload)
		tag, qerr := s.pool.Exec(ctx, `
			INSERT INTO normalized_records (raw_artifact_id, source_feed_id, record_type, structured_payload_json, record_hash)
			VALUES ($1,$2,$3,$4,$5)
			ON CONFLICT (source_feed_id, record_hash) DO NOTHING`,
			rawID, feedID, microsoft365.RecordTypeCloudFile, normPayload, nh)
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

type m365DriveItem struct {
	ID                   string `json:"id"`
	Name                 string `json:"name"`
	WebURL               string `json:"webUrl"`
	LastModifiedDateTime string `json:"lastModifiedDateTime"`
	File                 *struct {
		MimeType string `json:"mimeType"`
	} `json:"file,omitempty"`
	Folder *struct {
		ChildCount int `json:"childCount"`
	} `json:"folder,omitempty"`
	ParentReference struct {
		Path    string `json:"path"`
		DriveID string `json:"driveId"`
	} `json:"parentReference"`
	CreatedBy struct {
		User struct {
			Email string `json:"email"`
			Mail  string `json:"mail"`
		} `json:"user"`
	} `json:"createdBy"`
}

func (s *Service) syncM365Calendar(ctx context.Context, feedID, runID uuid.UUID, feed *SourceFeed, conn *Connector, cfg *m365FeedConfig) (*IngestionRun, error) {
	hours := cfg.TimeWindowHours
	if hours <= 0 {
		hours = 168
	}
	if hours > 720 {
		hours = 720
	}
	start := time.Now().UTC().Add(-time.Duration(hours) * time.Hour)
	end := time.Now().UTC().Add(24 * time.Hour)
	startS := start.Format(time.RFC3339)
	endS := end.Format(time.RFC3339)

	calRef := strings.TrimSpace(feed.ExternalRef)
	var listURL string
	q := url.Values{}
	q.Set("startDateTime", startS)
	q.Set("endDateTime", endS)
	q.Set("$top", fmt.Sprintf("%d", m365CalendarMaxItems))
	if calRef == "" || strings.EqualFold(calRef, "primary") {
		listURL = "https://graph.microsoft.com/v1.0/me/calendar/calendarView?" + q.Encode()
	} else {
		listURL = fmt.Sprintf("https://graph.microsoft.com/v1.0/me/calendars/%s/calendarView?%s", url.PathEscape(calRef), q.Encode())
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, listURL, nil)
	if err != nil {
		s.finalizeIngestionRun(ctx, runID, "failed", 0, 0, 0, 1)
		s.completeSourceFeedSync(ctx, feedID, 1, false)
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.AccessToken)
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
		return nil, fmt.Errorf("graph calendar: status %d: %s", resp.StatusCode, string(body))
	}

	var parsed struct {
		Value []struct {
			ID        string         `json:"id"`
			Subject   string         `json:"subject"`
			Start     m365DateTimeTZ `json:"start"`
			End       m365DateTimeTZ `json:"end"`
			Organizer struct {
				EmailAddress struct {
					Address string `json:"address"`
				} `json:"emailAddress"`
			} `json:"organizer"`
			Attendees []struct {
				EmailAddress struct {
					Address string `json:"address"`
				} `json:"emailAddress"`
			} `json:"attendees"`
			OnlineMeeting *struct {
				JoinURL string `json:"joinUrl"`
			} `json:"onlineMeeting"`
		} `json:"value"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		s.finalizeIngestionRun(ctx, runID, "failed", 0, 0, 0, 1)
		s.completeSourceFeedSync(ctx, feedID, 1, false)
		return nil, err
	}

	calendarRef := calRef
	if calendarRef == "" {
		calendarRef = "primary"
	}

	ingested, deduped, warnings, errs := 0, 0, 0, 0
	for _, ev := range parsed.Value {
		rawPayload, _ := json.Marshal(ev)
		h := hashBytes(rawPayload)
		extID := "m365-cal-" + ev.ID
		var rawObj map[string]any
		_ = json.Unmarshal(rawPayload, &rawObj)
		extra := map[string]any{"m365_calendar_event": rawObj}
		metaJSON, merr := s.buildRawArtifactMetadataJSON(ctx, *conn, *feed, microsoft365.ArtifactKindGraphJSON, rawPayload, extra)
		if merr != nil {
			errs++
			continue
		}
		rawID, inserted, ierr := insertRawArtifactRow(ctx, s.pool, feedID, runID, microsoft365.ArtifactM365CalendarEvent, extID, h, "", metaJSON, nil, nil)
		if ierr != nil {
			errs++
			continue
		}
		if !inserted {
			deduped++
			continue
		}
		var startT, endT *time.Time
		if t, err := m365ParseGraphDateTime(ev.Start); err == nil {
			startT = &t
		}
		if t, err := m365ParseGraphDateTime(ev.End); err == nil {
			endT = &t
		}
		participants := make([]string, 0, len(ev.Attendees))
		for _, a := range ev.Attendees {
			if addr := strings.TrimSpace(a.EmailAddress.Address); addr != "" {
				participants = append(participants, addr)
			}
		}
		meetingRef := ""
		if ev.OnlineMeeting != nil && ev.OnlineMeeting.JoinURL != "" {
			meetingRef = ev.OnlineMeeting.JoinURL
		}
		norm := microsoft365.NormalizedM365CalendarEvent{
			SourceFeedID:    feedID,
			ConnectorFamily: "microsoft365",
			ConnectorType:   "outlook_calendar",
			ExternalEventID: ev.ID,
			Title:           ev.Subject,
			Organizer:       strings.TrimSpace(ev.Organizer.EmailAddress.Address),
			Participants:    participants,
			StartTime:       startT,
			EndTime:         endT,
			CalendarRef:     calendarRef,
			MeetingRef:      meetingRef,
		}
		normPayload, _ := json.Marshal(norm)
		nh := hashBytes(normPayload)
		tag, qerr := s.pool.Exec(ctx, `
			INSERT INTO normalized_records (raw_artifact_id, source_feed_id, record_type, structured_payload_json, record_hash)
			VALUES ($1,$2,$3,$4,$5)
			ON CONFLICT (source_feed_id, record_hash) DO NOTHING`,
			rawID, feedID, microsoft365.RecordTypeCalendarEvent, normPayload, nh)
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

type m365DateTimeTZ struct {
	DateTime string `json:"dateTime"`
	TimeZone string `json:"timeZone"`
}

func m365ParseGraphDateTime(dt m365DateTimeTZ) (time.Time, error) {
	if strings.TrimSpace(dt.DateTime) == "" {
		return time.Time{}, errors.New("empty")
	}
	loc := time.UTC
	if tz := strings.TrimSpace(dt.TimeZone); tz != "" {
		if l, err := time.LoadLocation(tz); err == nil {
			loc = l
		}
	}
	s := dt.DateTime
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.0000000",
		"2006-01-02T15:04:05.000000",
		"2006-01-02T15:04:05",
	}
	var lastErr error
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, s, loc); err == nil {
			return t.UTC(), nil
		} else {
			lastErr = err
		}
	}
	return time.Time{}, lastErr
}
