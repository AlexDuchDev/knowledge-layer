package ingestion_connectors

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/knowledgelayer/api/internal/db"
	"github.com/knowledgelayer/api/internal/ingestion_connectors/families/chat"
	"github.com/knowledgelayer/api/internal/ingestion_connectors/families/docs_wiki"
	"github.com/knowledgelayer/api/internal/ingestion_connectors/families/meeting"
)

func TestProcessQueuedRawArtifact_normalizesSupportedFamilies(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}

	ctx := context.Background()
	if err := db.MigrateUp(dsn); err != nil {
		t.Fatal(err)
	}
	pool, err := db.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	svc := NewService(pool, nil)

	admin := uuid.MustParse("30000000-0000-0000-0000-000000000001")
	domain := uuid.MustParse("32000000-0000-0000-0000-000000000001")

	connTelegram := uuid.MustParse("20000000-0000-0000-0000-000000000001")
	connSlack := uuid.MustParse("20000000-0000-0000-0000-000000000003")
	connDrive := uuid.MustParse("20000000-0000-0000-0000-000000000002")
	connNotion := uuid.MustParse("20000000-0000-0000-0000-000000000004")
	connConfluence := uuid.MustParse("20000000-0000-0000-0000-00000000000c")
	connGoogleCalendar := uuid.MustParse("20000000-0000-0000-0000-000000000008")
	connFireflies := uuid.MustParse("20000000-0000-0000-0000-000000000007")
	connMattermost := uuid.MustParse("20000000-0000-0000-0000-000000000013")

	insertSourceFeed := func(connID uuid.UUID) uuid.UUID {
		feedID := uuid.New()
		_, err := pool.Exec(ctx, `
			INSERT INTO source_feeds (
				id, connector_id, source_uri, display_name, owner_id, domain_id,
				sensitivity_level, allowed_job_types_json, ingestion_mode, connector_config_json, status
			) VALUES ($1,$2,'', 'test-feed', $3,$4, 0, '["weekly_digest"]'::jsonb, 'ingestion_only', '{}'::jsonb, 'active')`,
			feedID, connID, admin, domain,
		)
		if err != nil {
			t.Fatal(err)
		}
		return feedID
	}

	insertIngestionRun := func(feedID uuid.UUID) uuid.UUID {
		runID := uuid.New()
		_, err := pool.Exec(ctx, `
			INSERT INTO ingestion_runs (id, source_feed_id, trigger_type, status, started_at)
			VALUES ($1,$2,'manual','completed',now())`, runID, feedID,
		)
		if err != nil {
			t.Fatal(err)
		}
		return runID
	}

	insertRawArtifact := func(feedID, runID uuid.UUID, artifactType string, externalID string, metadata any) (rawID uuid.UUID) {
		rawID = uuid.New()
		metaBytes, err := json.Marshal(metadata)
		if err != nil {
			t.Fatal(err)
		}

		srcCreatedAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

		_, err = pool.Exec(ctx, `
			INSERT INTO raw_artifacts (
				id, source_feed_id, ingestion_run_id, artifact_type, external_artifact_id,
				storage_uri, content_hash, metadata_json, source_created_at, source_author_ref
			) VALUES ($1,$2,$3,$4,$5,'', $6, $7::jsonb, $8, NULL)`,
			rawID, feedID, runID, artifactType, externalID,
			fmt.Sprintf("hash-%s-%s", artifactType, rawID.String()),
			string(metaBytes),
			srcCreatedAt,
		)
		if err != nil {
			t.Fatal(err)
		}
		return rawID
	}

	getNormalized := func(rawID uuid.UUID) (count int, recordType string, payload json.RawMessage) {
		err := pool.QueryRow(ctx, `SELECT COUNT(1) FROM normalized_records WHERE raw_artifact_id=$1`, rawID).Scan(&count)
		if err != nil {
			t.Fatal(err)
		}
		err = pool.QueryRow(ctx, `SELECT record_type, structured_payload_json FROM normalized_records WHERE raw_artifact_id=$1`, rawID).
			Scan(&recordType, &payload)
		if err != nil {
			t.Fatal(err)
		}
		return count, recordType, payload
	}

	assertNormalizedOnce := func(rawID uuid.UUID) {
		// First run should create exactly one normalized record.
		if err := svc.ProcessQueuedRawArtifact(ctx, rawID); err != nil {
			t.Fatalf("ProcessQueuedRawArtifact(rawID=%s): %v", rawID, err)
		}
		count1, _, _ := getNormalized(rawID)
		if count1 != 1 {
			t.Fatalf("expected normalized_records count=1 for rawID=%s, got %d", rawID, count1)
		}

		// Second run must be idempotent.
		if err := svc.ProcessQueuedRawArtifact(ctx, rawID); err != nil {
			t.Fatalf("ProcessQueuedRawArtifact(idempotent rawID=%s): %v", rawID, err)
		}
		count2, _, _ := getNormalized(rawID)
		if count2 != 1 {
			t.Fatalf("expected normalized_records count to stay=1 for rawID=%s, got %d", rawID, count2)
		}
	}

	// 1) Chat family
	{
		feedID := insertSourceFeed(connTelegram)
		runID := insertIngestionRun(feedID)
		rawID := insertRawArtifact(feedID, runID, "telegram_update", "tg-1", map[string]any{
			"telegram_update": map[string]any{
				"update_id": 1,
				"message": map[string]any{
					"message_id": 10,
					"date":       int64(1710000000),
					"text":       "hi",
					"chat": map[string]any{
						"id": int64(123),
					},
				},
			},
		})
		assertNormalizedOnce(rawID)
		count, recordType, payload := getNormalized(rawID)
		if count != 1 {
			t.Fatal("expected exactly one normalized row")
		}
		if recordType != chat.RecordTypeChatMessage {
			t.Fatalf("telegram_update record_type: expected %q, got %q", chat.RecordTypeChatMessage, recordType)
		}
		var msg chat.NormalizedChatMessage
		_ = json.Unmarshal(payload, &msg)
		if msg.TextBody != "hi" {
			t.Fatalf("telegram_update text: expected %q, got %q", "hi", msg.TextBody)
		}
		if msg.ExternalMessageID != "10" {
			t.Fatalf("telegram_update external_message_id: expected %q, got %q", "10", msg.ExternalMessageID)
		}
	}
	{
		feedID := insertSourceFeed(connSlack)
		runID := insertIngestionRun(feedID)
		rawID := insertRawArtifact(feedID, runID, "slack_message", "slack-CHAN-1700000000.00", map[string]any{
			"slack_message": map[string]any{
				"type":        "message",
				"user":        "u1",
				"text":        "hello",
				"ts":          "1700000000.00",
				"thread_ts":   "1700000000.00",
				"reply_count": 0,
			},
		})
		assertNormalizedOnce(rawID)
		_, recordType, payload := getNormalized(rawID)
		if recordType != chat.RecordTypeChatMessage {
			t.Fatalf("slack_message record_type: expected %q, got %q", chat.RecordTypeChatMessage, recordType)
		}
		var msg chat.NormalizedChatMessage
		_ = json.Unmarshal(payload, &msg)
		if msg.TextBody != "hello" {
			t.Fatalf("slack_message text: expected %q, got %q", "hello", msg.TextBody)
		}
		if msg.ChannelOrChatRef != "CHAN" {
			t.Fatalf("slack_message channel_ref: expected %q, got %q", "CHAN", msg.ChannelOrChatRef)
		}
	}
	{
		feedID := insertSourceFeed(connMattermost)
		runID := insertIngestionRun(feedID)
		rawID := insertRawArtifact(feedID, runID, "mattermost_post", "mm-chan1:post1", map[string]any{
			"mattermost_post": map[string]any{
				"id":         "post1",
				"create_at":  1700000000000,
				"user_id":    "u1",
				"message":    "hello mm",
				"root_id":    "",
				"channel_id": "chan1",
			},
		})
		assertNormalizedOnce(rawID)
		_, recordType, payload := getNormalized(rawID)
		if recordType != chat.RecordTypeChatMessage {
			t.Fatalf("mattermost_post record_type: expected %q, got %q", chat.RecordTypeChatMessage, recordType)
		}
		var msg chat.NormalizedChatMessage
		_ = json.Unmarshal(payload, &msg)
		if msg.ConnectorType != "mattermost" || msg.TextBody != "hello mm" {
			t.Fatalf("mattermost_post payload mismatch: %+v", msg)
		}
		if msg.ChannelOrChatRef != "chan1" {
			t.Fatalf("mattermost channel_ref: expected %q, got %q", "chan1", msg.ChannelOrChatRef)
		}
	}

	// 2) Docs / files family
	{
		feedID := insertSourceFeed(connDrive)
		runID := insertIngestionRun(feedID)
		rawID := insertRawArtifact(feedID, runID, "google_drive_file", "file-1", map[string]any{
			"file_id":       "file-1",
			"name":          "Doc",
			"mime_type":     "application/vnd.google-apps.document",
			"export_mime":   "text/plain",
			"parents":       []string{"parent1"},
			"owners":        []string{"a@b.c"},
			"last_modifier": "owner",
			"web_view_link": "https://example.com/doc",
			"truncated":     false,
			"content_text":  "body text",
			"connector":     "google_drive",
		})
		assertNormalizedOnce(rawID)
		_, recordType, payload := getNormalized(rawID)
		if recordType != docs_wiki.RecordTypeDocsPage {
			t.Fatalf("google_drive_file record_type: expected %q, got %q", docs_wiki.RecordTypeDocsPage, recordType)
		}
		var doc docs_wiki.NormalizedDocPage
		_ = json.Unmarshal(payload, &doc)
		if doc.Title != "Doc" || doc.BodyText != "body text" || doc.ExternalRef != "file-1" {
			t.Fatalf("google_drive_file payload mismatch: %+v", doc)
		}
	}
	{
		feedID := insertSourceFeed(connNotion)
		runID := insertIngestionRun(feedID)
		rawID := insertRawArtifact(feedID, runID, "notion_page", "notion-page-1", map[string]any{
			"notion_page": map[string]any{
				"page_id": "page-1",
				"title":   "N title",
				"body":    "N body",
			},
		})
		assertNormalizedOnce(rawID)
		_, recordType, payload := getNormalized(rawID)
		if recordType != docs_wiki.RecordTypeDocsPage {
			t.Fatalf("notion_page record_type: expected %q, got %q", docs_wiki.RecordTypeDocsPage, recordType)
		}
		var doc docs_wiki.NormalizedDocPage
		_ = json.Unmarshal(payload, &doc)
		if doc.Title != "N title" || doc.BodyText != "N body" || doc.ExternalRef != "page-1" {
			t.Fatalf("notion_page payload mismatch: %+v", doc)
		}
	}
	{
		feedID := insertSourceFeed(connConfluence)
		runID := insertIngestionRun(feedID)

		rawID := insertRawArtifact(feedID, runID, docs_wiki.ArtifactConfluencePage, "confluence-c1", map[string]any{
			"confluence_page": map[string]any{
				"id":    "c1",
				"type":  "page",
				"title": "C title",
				"body": map[string]any{
					"storage": map[string]any{
						"value":          "C body",
						"representation": "storage",
					},
				},
				"version": map[string]any{
					"when": "2026-01-01T00:00:00Z",
				},
				"history": map[string]any{
					"lastUpdated": map[string]any{
						"when": "2026-01-02T00:00:00Z",
						"by": map[string]any{
							"username":    "editor",
							"displayName": "Editor",
						},
					},
				},
				"metadata": map[string]any{
					"labels": map[string]any{
						"results": []map[string]any{
							{"name": "team"},
						},
					},
				},
				"ancestors": []map[string]any{
					{"id": "anc1"},
				},
				"_links": map[string]any{
					"webui": "https://example.com/c1",
				},
			},
		})
		assertNormalizedOnce(rawID)
		_, recordType, payload := getNormalized(rawID)
		if recordType != docs_wiki.RecordTypeDocsPage {
			t.Fatalf("confluence_page record_type: expected %q, got %q", docs_wiki.RecordTypeDocsPage, recordType)
		}
		var doc docs_wiki.NormalizedDocPage
		_ = json.Unmarshal(payload, &doc)
		if doc.Title != "C title" || doc.BodyText != "C body" || doc.ExternalRef != "c1" {
			t.Fatalf("confluence_page payload mismatch: %+v", doc)
		}
	}

	// 3) Calendar / meeting family
	{
		feedID := insertSourceFeed(connGoogleCalendar)
		runID := insertIngestionRun(feedID)
		rawID := insertRawArtifact(feedID, runID, "google_calendar_event", "evt-1", map[string]any{
			"google_calendar_event": map[string]any{
				"id":       "evt1",
				"summary":  "My Project. Weekly sync",
				"htmlLink": "https://example.com/evt1",
				"start": map[string]any{
					"dateTime": "2026-01-01T10:00:00Z",
				},
				"end": map[string]any{
					"dateTime": "2026-01-01T11:00:00Z",
				},
			},
		})
		assertNormalizedOnce(rawID)
		_, recordType, payload := getNormalized(rawID)
		if recordType != meeting.RecordTypeCalendarEvent {
			t.Fatalf("google_calendar_event record_type: expected %q, got %q", meeting.RecordTypeCalendarEvent, recordType)
		}
		var ev meeting.NormalizedCalendarEvent
		_ = json.Unmarshal(payload, &ev)
		if ev.ExternalRef != "evt1" || ev.Summary != "My Project. Weekly sync" {
			t.Fatalf("google_calendar_event payload mismatch: %+v", ev)
		}
		if !ev.TitleParseOK || ev.ParsedProjectTitle != "My Project" || ev.ParsedMeetingTopic != "Weekly sync" {
			t.Fatalf("google_calendar_event parse fields: %+v", ev)
		}
	}
	{
		feedID := insertSourceFeed(connFireflies)
		runID := insertIngestionRun(feedID)
		rawID := insertRawArtifact(feedID, runID, "fireflies_transcript", "tr-1", map[string]any{
			"fireflies_transcript": map[string]any{
				"id":    "tr1",
				"title": "Transcript title",
				"date":  "2026-01-01T00:00:00Z",
			},
		})
		assertNormalizedOnce(rawID)
		_, recordType, payload := getNormalized(rawID)
		if recordType != meeting.RecordTypeTranscript {
			t.Fatalf("fireflies_transcript record_type: expected %q, got %q", meeting.RecordTypeTranscript, recordType)
		}
		var tr meeting.NormalizedMeetingTranscript
		_ = json.Unmarshal(payload, &tr)
		if tr.ExternalRef != "tr1" || tr.Title != "Transcript title" {
			t.Fatalf("fireflies_transcript payload mismatch: %+v", tr)
		}
	}
}
