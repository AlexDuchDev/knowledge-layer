package ingestion_connectors

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"google.golang.org/api/drive/v3"

	"github.com/knowledgelayer/api/internal/ingestion_connectors/families/docs_wiki"
)

func TestParseGoogleDriveConfig(t *testing.T) {
	t.Run("missing folder", func(t *testing.T) {
		_, err := parseGoogleDriveConfig([]byte(`{"service_account":{}}`))
		if err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("missing service account", func(t *testing.T) {
		_, err := parseGoogleDriveConfig([]byte(`{"folder_id":"abc"}`))
		if err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("defaults max files", func(t *testing.T) {
		cfg, err := parseGoogleDriveConfig([]byte(`{"folder_id":"f1","service_account":{"type":"service_account"}}`))
		if err != nil {
			t.Fatal(err)
		}
		if cfg.MaxFilesPerSync != defaultMaxDriveFiles {
			t.Fatalf("max files %d", cfg.MaxFilesPerSync)
		}
	})
	t.Run("caps max files", func(t *testing.T) {
		cfg, err := parseGoogleDriveConfig([]byte(`{"folder_id":"f1","service_account":{"type":"service_account"},"max_files_per_sync":9999}`))
		if err != nil {
			t.Fatal(err)
		}
		if cfg.MaxFilesPerSync != maxDriveFilesCap {
			t.Fatalf("got %d", cfg.MaxFilesPerSync)
		}
	})
}

func TestBuildNormalizedDriveRecord(t *testing.T) {
	f := &drive.File{
		Id:           "file-1",
		Name:         "Spec",
		MimeType:     "application/vnd.google-apps.document",
		ModifiedTime: "2024-01-02T15:04:05Z",
		Parents:      []string{"folder-9"},
		WebViewLink:  "https://docs.google.com/document/d/file-1",
		LastModifyingUser: &drive.User{
			EmailAddress: "editor@example.com",
		},
	}
	feedID := uuid.MustParse("10000000-0000-0000-0000-000000000001")
	b, rt, err := buildNormalizedDriveRecord(feedID, f, "text/plain", []byte("hello"), false)
	if err != nil {
		t.Fatal(err)
	}
	if rt != docs_wiki.RecordTypeDocsPage {
		t.Fatalf("record type %s", rt)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	if m["external_ref"] != "file-1" {
		t.Fatalf("external_ref %v", m["external_ref"])
	}
	if m["connector_family"] != "docs_wiki" {
		t.Fatalf("connector_family %v", m["connector_family"])
	}
}
