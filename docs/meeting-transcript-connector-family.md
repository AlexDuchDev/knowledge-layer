# Meeting / transcript connector family

Transcript providers and **calendar context** (no transcript in calendar) share the `meeting` family package.

## Record types

| `record_type` | Meaning |
|---------------|---------|
| `meeting_transcript` | `NormalizedMeetingTranscript` (e.g. Fireflies) |
| `calendar_event` | `NormalizedCalendarEvent` (Google Calendar context only). When the event `summary` matches `Project. Topic`, optional fields `parsed_project_title`, `parsed_meeting_topic`, and `title_parse_ok` are set (see `ParseCalendarSummaryProjectTopic` in Go package `meeting`). |

## Fireflies v1

- `connector_config_json`: `fireflies_api_key` (Bearer to Fireflies GraphQL).
- GraphQL query for `transcripts { id title date }` (API may evolve).
- **Security / compliance:** [FIREFLIES_SECURITY.md](./FIREFLIES_SECURITY.md) (tokens, retention, who can see transcript text in KL).

## Google Calendar v1

- `connector_config_json`: `service_account` (JSON object, Calendar readonly scope).
- `external_ref`: calendar id.

## Package

Go: `internal/ingestion_connectors/families/meeting`.
