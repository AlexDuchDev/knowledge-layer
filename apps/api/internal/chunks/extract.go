package chunks

import (
	"encoding/json"
	"strings"
)

// SourceTypeEntityBody is the legacy chunk source: an entity's title+summary+body.
const SourceTypeEntityBody = "entity_body"

// SourceTypeNormalizedRecord is a chunk rooted in a normalized_record (chat
// message, doc page, meeting transcript, etc.) before any entity exists.
const SourceTypeNormalizedRecord = "normalized_record"

// extractTextFromNormalizedRecord pulls the operator-relevant prose out of a
// normalized_record's structured_payload_json. The mapping is intentionally
// per-record-type and conservative: only fields a human would expect to be
// retrievable end up in the chunked text. Provider-specific control fields
// (channel ids, mime types, timestamps) are excluded so they do not pollute
// embedding signal or surface in /search results.
//
// Unknown record_type returns ("", false) and the caller skips chunking;
// pre-mapping that record type is then a follow-up that lands in the registry
// alongside whatever family added it. Callers should NOT panic on miss —
// "unknown record_type" is the documented case when a new connector lands
// before a chunk-extraction reviewer sees it.
func extractTextFromNormalizedRecord(recordType string, payload json.RawMessage) (string, bool) {
	if len(payload) == 0 {
		return "", false
	}
	var p map[string]any
	if err := json.Unmarshal(payload, &p); err != nil {
		return "", false
	}
	switch recordType {
	case "chat_message":
		// Slack / Telegram / Mattermost / Microsoft Teams individual message.
		// Author + channel ref live elsewhere; the prose is text_body.
		return cleanText(p["text_body"]), true
	case "chat_thread":
		// A chat_thread normalises an aggregate of messages; concatenate them.
		return joinMessages(p["messages"]), true
	case "docs_page":
		// Notion / Confluence / Microsoft Docs — title + full body.
		// title goes first so a search for the title rank-orders correctly.
		t := cleanText(p["title"])
		b := cleanText(p["body_text"])
		return joinNonEmpty([]string{t, b}, "\n\n"), true
	case "meeting_transcript":
		// Fireflies / Google Meet recordings. body_text is the joined transcript;
		// some adapters also produce title + summary fields.
		t := cleanText(p["title"])
		s := cleanText(p["summary"])
		b := cleanText(p["body_text"])
		return joinNonEmpty([]string{t, s, b}, "\n\n"), true
	case "calendar_event":
		// Google Calendar / Microsoft 365. Title + parsed topic + parsed project,
		// no body — the human signal is in summary + parsed metadata.
		s := cleanText(p["summary"])
		topic := cleanText(p["parsed_meeting_topic"])
		proj := cleanText(p["parsed_project_title"])
		return joinNonEmpty([]string{s, topic, proj}, " — "), true
	case "email_message":
		// Gmail / M365 mail.
		subj := cleanText(p["subject"])
		body := cleanText(p["body_text"])
		return joinNonEmpty([]string{subj, body}, "\n\n"), true
	case "m365_mail_message":
		subj := cleanText(p["subject"])
		body := cleanText(p["body_preview"])
		return joinNonEmpty([]string{subj, body}, "\n\n"), true
	case "m365_teams_message":
		return cleanText(p["text_body"]), true
	case "m365_cloud_file":
		t := cleanText(p["title"])
		b := cleanText(p["body_text"])
		return joinNonEmpty([]string{t, b}, "\n\n"), true
	case "m365_calendar_event":
		s := cleanText(p["subject"])
		body := cleanText(p["body_preview"])
		return joinNonEmpty([]string{s, body}, "\n\n"), true
	case "work_item":
		// Linear / Jira / Asana / Trello issue or card.
		t := cleanText(p["title"])
		desc := cleanText(p["description"])
		return joinNonEmpty([]string{t, desc}, "\n\n"), true
	case "support_ticket", "support_conversation":
		// Zendesk / Intercom / HubSpot ticket.
		t := cleanText(p["subject"])
		desc := cleanText(p["description"])
		body := cleanText(p["body_text"])
		return joinNonEmpty([]string{t, desc, body}, "\n\n"), true
	case "crm_record":
		t := cleanText(p["title"])
		desc := cleanText(p["description"])
		notes := cleanText(p["notes"])
		return joinNonEmpty([]string{t, desc, notes}, "\n\n"), true
	case "google_drive_document":
		// Pre-existing reference_document_map.go path also creates an entity from
		// google_drive_document. After v0.3.0 we ALSO chunk the normalized record
		// so it's retrievable before the entity-create job has run. The eventual
		// entity-rooted chunks de-duplicate via the entity_search_projection.
		t := cleanText(p["title"])
		b := cleanText(p["body_text"])
		return joinNonEmpty([]string{t, b}, "\n\n"), true
	default:
		return "", false
	}
}

// cleanText pulls a string from an arbitrary JSON value and trims it. Returns
// empty string for nil / non-string. NOT type-strict on purpose — a number that
// somehow leaked into a text field is rendered, not silently dropped.
func cleanText(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case json.Number:
		return strings.TrimSpace(t.String())
	default:
		// Render anything else as JSON; rare path, mostly a safety net for
		// unexpected payload shapes from a buggy connector.
		b, err := json.Marshal(v)
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(b))
	}
}

// joinMessages flattens a payload's "messages": [...] array into prose. Each
// message is rendered as "<author>: <text>" when both are present, or just
// the text otherwise. Used for chat_thread record type aggregation.
func joinMessages(v any) string {
	arr, ok := v.([]any)
	if !ok {
		return ""
	}
	var lines []string
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		text := cleanText(m["text"])
		if text == "" {
			text = cleanText(m["text_body"])
		}
		if text == "" {
			continue
		}
		author := cleanText(m["author"])
		if author == "" {
			author = cleanText(m["author_ref"])
		}
		if author != "" {
			lines = append(lines, author+": "+text)
		} else {
			lines = append(lines, text)
		}
	}
	return strings.Join(lines, "\n")
}

// joinNonEmpty joins fragments with a separator, dropping empty ones. Avoids
// runs like "\n\n\n\n" when only one of (title, summary, body) is present.
func joinNonEmpty(parts []string, sep string) string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if strings.TrimSpace(p) == "" {
			continue
		}
		out = append(out, strings.TrimSpace(p))
	}
	return strings.Join(out, sep)
}
