package meeting

import (
	"time"

	"github.com/google/uuid"
)

// NormalizedMeetingTranscript is a lightweight transcript summary row (v1).
type NormalizedMeetingTranscript struct {
	SourceFeedID    uuid.UUID  `json:"source_feed_id"`
	ConnectorFamily string     `json:"connector_family"` // meeting
	ConnectorType   string     `json:"connector_type"`   // fireflies
	ExternalRef     string     `json:"external_ref"`
	Title           string     `json:"title,omitempty"`
	StartedAt       *time.Time `json:"started_at,omitempty"`
	BodyText        string     `json:"body_text,omitempty"`
	URLHint         string     `json:"url_hint,omitempty"`
}

// NormalizedCalendarEvent is context-only calendar metadata (no transcript).
type NormalizedCalendarEvent struct {
	SourceFeedID    uuid.UUID  `json:"source_feed_id"`
	ConnectorFamily string     `json:"connector_family"`
	ConnectorType   string     `json:"connector_type"` // google_calendar
	ExternalRef     string     `json:"external_ref"`
	Summary         string     `json:"summary"`
	StartAt         *time.Time `json:"start_at,omitempty"`
	EndAt           *time.Time `json:"end_at,omitempty"`
	HTMLLink        string     `json:"html_link,omitempty"`
	TranscriptHint  string     `json:"transcript_hint,omitempty"` // free-form link to transcript feed
	// ParsedProjectTitle / ParsedMeetingTopic are set when Summary matches "Project. Topic" (see ParseCalendarSummaryProjectTopic).
	ParsedProjectTitle string `json:"parsed_project_title,omitempty"`
	ParsedMeetingTopic string `json:"parsed_meeting_topic,omitempty"`
	TitleParseOK       bool   `json:"title_parse_ok,omitempty"`
}
