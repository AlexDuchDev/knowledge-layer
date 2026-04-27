// Package microsoft365 defines shared Microsoft 365 connector family metadata.
package microsoft365

// RecordTypeMailMessage is a normalized Graph mail row (v1).
const RecordTypeMailMessage = "m365_mail_message"

// RecordTypeTeamsMessage is a normalized Teams channel message row (v1).
const RecordTypeTeamsMessage = "m365_teams_message"

// RecordTypeCloudFile is OneDrive / SharePoint file metadata + optional extracted text (v1).
const RecordTypeCloudFile = "m365_cloud_file"

// RecordTypeCalendarEvent is Microsoft 365 calendar event context (not transcript).
const RecordTypeCalendarEvent = "m365_calendar_event"

const ArtifactKindGraphJSON = "microsoft365_graph_json"

// M365 file/calendar raw artifact types.
const (
	ArtifactM365FileMetadata   = "m365_file_metadata"
	ArtifactM365FileReference  = "m365_file_reference"
	ArtifactM365FileText       = "m365_file_text"
	ArtifactM365FolderContext  = "m365_folder_context"
	ArtifactM365CalendarEvent  = "m365_calendar_event"
	ArtifactM365MailMessage    = "m365_mail_message"
	ArtifactM365MailBody       = "m365_mail_body"
	ArtifactM365AttachmentMeta = "m365_attachment_metadata"
)
