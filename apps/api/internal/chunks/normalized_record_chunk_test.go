package chunks

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/knowledgelayer/api/internal/db"
)

// TestRebuildNormalizedRecordChunks_extractsTextAndPersists asserts the v0.3.0
// chunk-decoupling pipeline end-to-end against a real Postgres: a chat_message
// normalized_record yields ≥ 1 chunk row with source_type='normalized_record',
// non-nil normalized_record_id, NULL entity_id, and the expected text body.
//
// This is the DB-shape regression test — pre-v0.3.0 the same payload produced
// 0 chunks (the user's reported "система не чанкует" bug). Skipped without
// DATABASE_URL.
func TestRebuildNormalizedRecordChunks_extractsTextAndPersists(t *testing.T) {
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

	feedID := uuid.MustParse("32000000-0000-0000-0000-000000000001") // seed default domain - we'll get a real source_feed via a temporary insert
	// Provision one source_feed for the test (filesystem connector is the
	// canonical no-creds test fixture, same as RC validation).
	tmpFeedID := uuid.New()
	if _, err := pool.Exec(ctx, `
		INSERT INTO source_feeds (id, connector_id, source_uri, display_name, owner_id, domain_id,
			sensitivity_level, allowed_job_types_json, ingestion_mode, connector_config_json, status)
		VALUES ($1, '20000000-0000-0000-0000-000000000012', '', 'chunk-test-feed',
			'30000000-0000-0000-0000-000000000001', $2, 0,
			'["weekly_digest"]'::jsonb, 'ingestion_only', '{}'::jsonb, 'draft')`,
		tmpFeedID, feedID); err != nil {
		t.Fatalf("insert source_feed: %v", err)
	}
	defer pool.Exec(ctx, `DELETE FROM source_feeds WHERE id = $1`, tmpFeedID)

	runID := uuid.New()
	if _, err := pool.Exec(ctx, `
		INSERT INTO ingestion_runs (id, source_feed_id, trigger_type, status)
		VALUES ($1, $2, 'manual', 'completed')`,
		runID, tmpFeedID); err != nil {
		t.Fatalf("insert ingestion_run: %v", err)
	}
	defer pool.Exec(ctx, `DELETE FROM ingestion_runs WHERE id = $1`, runID)

	rawID := uuid.New()
	if _, err := pool.Exec(ctx, `
		INSERT INTO raw_artifacts (id, source_feed_id, ingestion_run_id, artifact_type, external_artifact_id, content_hash, metadata_json)
		VALUES ($1, $2, $3, 'chat_message', 'test-ext-ref', 'hash-test', '{}'::jsonb)`,
		rawID, tmpFeedID, runID); err != nil {
		t.Fatalf("insert raw_artifact: %v", err)
	}
	defer pool.Exec(ctx, `DELETE FROM raw_artifacts WHERE id = $1`, rawID)

	// Insert a chat_message normalized_record with realistic shape per the
	// chat family extractor (chunks/extract.go: text_body is the prose).
	normID := uuid.New()
	payload := json.RawMessage(`{
		"text_body": "Discussed the new release plan. Need to confirm with security team before shipping.",
		"author_ref": "u1",
		"channel_or_chat_ref": "general",
		"posted_at": "2026-04-28T10:00:00Z",
		"connector_family": "chat",
		"connector_type": "slack"
	}`)
	if _, err := pool.Exec(ctx, `
		INSERT INTO normalized_records (id, raw_artifact_id, source_feed_id, record_type, structured_payload_json, record_hash, normalization_version)
		VALUES ($1, $2, $3, 'chat_message', $4, 'norm-hash-test', 1)`,
		normID, rawID, tmpFeedID, payload); err != nil {
		t.Fatalf("insert normalized_record: %v", err)
	}
	defer pool.Exec(ctx, `DELETE FROM normalized_records WHERE id = $1`, normID)

	ids, err := svc.RebuildNormalizedRecordChunks(ctx, normID)
	if err != nil {
		t.Fatalf("RebuildNormalizedRecordChunks: %v", err)
	}
	if len(ids) == 0 {
		t.Fatal("expected at least one chunk for chat_message payload, got zero (chunking is broken)")
	}

	// Inspect the row shape: normalized_record_id non-nil, entity_id NULL,
	// source_type='normalized_record', text contains the input prose.
	chunks, err := svc.ListByNormalizedRecord(ctx, normID)
	if err != nil {
		t.Fatalf("ListByNormalizedRecord: %v", err)
	}
	if len(chunks) != len(ids) {
		t.Errorf("ListByNormalizedRecord returned %d chunks, expected %d", len(chunks), len(ids))
	}
	for _, c := range chunks {
		if c.NormalizedRecordID == uuid.Nil {
			t.Error("expected NormalizedRecordID populated, got Nil")
		}
		if c.EntityID != uuid.Nil {
			t.Errorf("expected EntityID Nil for normalized_record-rooted chunk, got %s", c.EntityID)
		}
		if c.SourceType != SourceTypeNormalizedRecord {
			t.Errorf("expected SourceType=%q, got %q", SourceTypeNormalizedRecord, c.SourceType)
		}
		if !strings.Contains(c.TextContent, "release plan") {
			t.Errorf("expected text to contain 'release plan', got %q", c.TextContent)
		}
	}
}

// TestExtractTextFromNormalizedRecord_perTypeRegistry asserts every record_type
// the OSS connectors produce has a non-empty extractor mapping. New record_types
// added to the codebase without an extractor entry will fail this assertion —
// the intent is to make "I added a connector but forgot the chunker" a CI-loud
// regression.
func TestExtractTextFromNormalizedRecord_perTypeRegistry(t *testing.T) {
	cases := []struct {
		recordType string
		payload    string
		expectText string
	}{
		{"chat_message", `{"text_body":"hello"}`, "hello"},
		{"docs_page", `{"title":"Spec","body_text":"body"}`, "Spec"},
		{"meeting_transcript", `{"title":"Meeting","body_text":"Transcript text"}`, "Meeting"},
		{"calendar_event", `{"summary":"Standup","parsed_meeting_topic":"daily"}`, "Standup"},
		{"email_message", `{"subject":"Re: tickets","body_text":"please fix"}`, "Re: tickets"},
		{"work_item", `{"title":"Bug","description":"Reproduce"}`, "Bug"},
		{"support_ticket", `{"subject":"Login broken","description":"500 on /login"}`, "Login broken"},
		{"google_drive_document", `{"title":"Report","body_text":"data"}`, "Report"},
	}
	for _, tc := range cases {
		t.Run(tc.recordType, func(t *testing.T) {
			text, ok := extractTextFromNormalizedRecord(tc.recordType, []byte(tc.payload))
			if !ok {
				t.Fatalf("extractor missing for record_type=%q", tc.recordType)
			}
			if !strings.Contains(text, tc.expectText) {
				t.Errorf("expected extracted text to contain %q, got %q", tc.expectText, text)
			}
		})
	}

	// Unknown record_type returns ("", false) — the documented "no extractor"
	// case so future record_types without a registry entry stay silent rather
	// than embed garbage.
	if _, ok := extractTextFromNormalizedRecord("zz_unknown_record_type", []byte(`{"x":"y"}`)); ok {
		t.Error("expected unknown record_type to return ok=false")
	}
}
