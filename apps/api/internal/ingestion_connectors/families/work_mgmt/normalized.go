package work_mgmt

import (
	"time"

	"github.com/google/uuid"
)

// NormalizedWorkItem is a generic issue/card for digest jobs.
type NormalizedWorkItem struct {
	SourceFeedID    uuid.UUID  `json:"source_feed_id"`
	ConnectorFamily string     `json:"connector_family"` // work_mgmt
	ConnectorType   string     `json:"connector_type"`   // jira, trello, asana, linear
	ExternalRef     string     `json:"external_ref"`
	Title           string     `json:"title"`
	BodyText        string     `json:"body_text,omitempty"`
	StatusName      string     `json:"status_name,omitempty"`
	AssigneeRef     string     `json:"assignee_ref,omitempty"`
	CreatorRef      string     `json:"creator_ref,omitempty"`
	ProjectRef      string     `json:"project_ref,omitempty"`
	TeamRef         string     `json:"team_ref,omitempty"`
	CycleRef        string     `json:"cycle_ref,omitempty"`
	Labels          []string   `json:"labels,omitempty"`
	CommentRefs     []string   `json:"comment_refs,omitempty"`
	CreatedAt       *time.Time `json:"created_at,omitempty"`
	UpdatedAt       *time.Time `json:"updated_at,omitempty"`
	URLHint         string     `json:"url_hint,omitempty"`
}
