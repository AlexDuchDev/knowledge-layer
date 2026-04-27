# Microsoft 365 connector family

One connector type **`microsoft_365`** with feed-level product selection.

## Record types

| `record_type` | Meaning |
|---------------|---------|
| `m365_mail_message` | Outlook mail (`NormalizedMailMessage`) |
| `m365_teams_message` | Teams channel message (`NormalizedTeamsMessage`) |
| `m365_cloud_file` | OneDrive / SharePoint file metadata (`NormalizedCloudFile`) |
| `m365_calendar_event` | Calendar event context (`NormalizedM365CalendarEvent`; not a transcript) |

## Configuration

- `connector_config_json`:
  - `m365_product`: `outlook` | `teams` | `onedrive` | `sharepoint` | `calendar`
  - `graph_access_token`: delegated Microsoft Graph bearer token (v1 stores token in feed config; production should move to secret ref / token refresh).
  - **Files:** `m365_files_scope`: `folder` | `library` | `subtree` | `search`; for `search`, `m365_search_query` is required; optional `m365_search_max_results` (capped).
  - **Outlook mail:** optional `mail_folder_id`, `shared_mailbox_upn`, `mail_filter_query` (OData `$filter`).
  - **Calendar:** optional `time_window_hours` (default 168, max 720).

## External ref

- **Outlook:** optional; used with folder / shared mailbox fields above.
- **Teams:** `team_id|channel_id` (pipe-separated).
- **OneDrive / SharePoint:** `me|root` (default drive root), `driveId|root`, or `driveId|itemId` for listing children; ignored when `m365_files_scope` is `search`.
- **Calendar:** empty or `primary` for default calendar, or a specific calendar id.

## Package

Go: `internal/ingestion_connectors/families/microsoft365`.
