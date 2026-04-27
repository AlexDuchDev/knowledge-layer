package microsoft365

import (
	"time"

	"github.com/google/uuid"
)

// NormalizedCloudFile is OneDrive / SharePoint item metadata (v1).
type NormalizedCloudFile struct {
	SourceFeedID       uuid.UUID  `json:"source_feed_id"`
	ConnectorFamily    string     `json:"connector_family"` // microsoft365
	ConnectorType      string     `json:"connector_type"`   // onedrive | sharepoint
	ExternalFileID     string     `json:"external_file_id"`
	TitleOrFileName    string     `json:"title_or_file_name"`
	MimeType           string     `json:"mime_type,omitempty"`
	Owner              string     `json:"owner,omitempty"`
	PathOrContainerRef string     `json:"path_or_container_ref,omitempty"`
	ModifiedAt         *time.Time `json:"modified_at,omitempty"`
	StorageReference   string     `json:"storage_reference,omitempty"` // driveItem id or webUrl
	ExtractedText      string     `json:"extracted_text,omitempty"`
}

// NormalizedM365CalendarEvent is Graph calendar event context for jobs/correlation.
type NormalizedM365CalendarEvent struct {
	SourceFeedID    uuid.UUID  `json:"source_feed_id"`
	ConnectorFamily string     `json:"connector_family"`
	ConnectorType   string     `json:"connector_type"` // outlook_calendar
	ExternalEventID string     `json:"external_event_id"`
	Title           string     `json:"title"`
	Organizer       string     `json:"organizer,omitempty"`
	Participants    []string   `json:"participants,omitempty"`
	StartTime       *time.Time `json:"start_time,omitempty"`
	EndTime         *time.Time `json:"end_time,omitempty"`
	CalendarRef     string     `json:"calendar_ref,omitempty"`
	MeetingRef      string     `json:"meeting_ref,omitempty"` // onlineMeeting joinUrl or id if present
}
