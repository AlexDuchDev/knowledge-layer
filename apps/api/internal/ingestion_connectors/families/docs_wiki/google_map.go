package docs_wiki

import (
	"time"

	"github.com/google/uuid"
)

// FromGoogleDriveExport maps Google Drive file export fields into NormalizedDocPage (no google SDK import).
func FromGoogleDriveExport(
	sourceFeedID uuid.UUID,
	title, fileID, exportMime, mimeType, body string,
	modifiedRFC3339 string,
	parentFolderIDs []string,
	ownerEmails []string,
	lastModifier string,
	webViewLink string,
	truncated bool,
) NormalizedDocPage {
	var mod *time.Time
	if modifiedRFC3339 != "" {
		if t, err := time.Parse(time.RFC3339, modifiedRFC3339); err == nil {
			mod = &t
		}
	}
	bodyOut := body
	if truncated {
		bodyOut += "\n\n[truncated at connector byte limit]"
	}
	parentRef := ""
	if len(parentFolderIDs) > 0 {
		parentRef = parentFolderIDs[0]
	}
	return NormalizedDocPage{
		SourceFeedID:    sourceFeedID,
		ConnectorFamily: "docs_wiki",
		ConnectorType:   "google_drive",
		Title:           title,
		ExternalRef:     fileID,
		ExternalDocID:   fileID,
		ParentRef:       parentRef,
		ParentRefs:      append([]string(nil), parentFolderIDs...),
		LastModifiedAt:  mod,
		MimeType:        mimeType,
		ExportMime:      exportMime,
		BodyText:        bodyOut,
		WebViewLink:     webViewLink,
		DownstreamHint: map[string]any{
			"owner_emails":          ownerEmails,
			"last_modifying_user":   lastModifier,
			"suggested_entity_type": "ReferenceDocument",
			"default_truth_mode":    "mirrored_authority",
		},
	}
}
