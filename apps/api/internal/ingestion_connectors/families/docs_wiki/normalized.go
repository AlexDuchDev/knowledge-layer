package docs_wiki

import (
	"time"

	"github.com/google/uuid"
)

// NormalizedDocPage is the canonical docs/wiki shape for digest and downstream jobs.
type NormalizedDocPage struct {
	SourceFeedID    uuid.UUID      `json:"source_feed_id"`
	ConnectorFamily string         `json:"connector_family"` // docs_wiki
	ConnectorType   string         `json:"connector_type"`   // notion, google_drive, confluence, ...
	Title           string         `json:"title"`
	ExternalRef     string         `json:"external_ref"`
	ExternalDocID   string         `json:"external_doc_id,omitempty"` // same as ExternalRef when applicable; explicit for Confluence/page ids
	ParentRef       string         `json:"parent_ref,omitempty"`
	ParentRefs      []string       `json:"parent_refs,omitempty"`
	SpaceRef        string         `json:"space_ref,omitempty"` // Confluence space key/id
	LastModifiedAt  *time.Time     `json:"last_modified_at,omitempty"`
	OwnerRef        string         `json:"owner_ref,omitempty"`
	EditorRef       string         `json:"editor_ref,omitempty"`
	Labels          []string       `json:"labels,omitempty"`
	MimeType        string         `json:"mime_type,omitempty"`
	ExportMime      string         `json:"export_mime,omitempty"`
	BodyText        string         `json:"body_text"`
	WebViewLink     string         `json:"web_view_link,omitempty"`
	DownstreamHint  map[string]any `json:"downstream_hint,omitempty"`
}
