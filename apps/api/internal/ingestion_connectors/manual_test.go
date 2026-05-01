package ingestion_connectors

import (
	"strings"
	"testing"
)

func TestParseYouTubeID(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"dQw4w9WgXcQ", "dQw4w9WgXcQ", false},
		{"https://www.youtube.com/watch?v=dQw4w9WgXcQ", "dQw4w9WgXcQ", false},
		{"https://youtu.be/dQw4w9WgXcQ", "dQw4w9WgXcQ", false},
		{"https://www.youtube.com/embed/dQw4w9WgXcQ", "dQw4w9WgXcQ", false},
		{"https://www.youtube.com/shorts/dQw4w9WgXcQ", "dQw4w9WgXcQ", false},
		{"https://www.youtube.com/watch?v=dQw4w9WgXcQ&list=ABC", "dQw4w9WgXcQ", false},
		{"", "", true},
		{"not-a-url", "", true},
		{"https://example.com", "", true},
	}
	for _, c := range cases {
		got, err := ParseYouTubeID(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("ParseYouTubeID(%q) expected error, got %q", c.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseYouTubeID(%q) error: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("ParseYouTubeID(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestExtractManualBytes_Plain(t *testing.T) {
	res, err := ExtractManualBytes("notes.txt", "text/plain", []byte("Hello world\nLine 2"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Title != "notes.txt" {
		t.Errorf("Title = %q, want notes.txt", res.Title)
	}
	if !strings.Contains(res.BodyText, "Hello world") {
		t.Errorf("BodyText missing input: %q", res.BodyText)
	}
}

func TestExtractManualBytes_HTML(t *testing.T) {
	body := []byte(`<html><head><title>My page</title></head><body><p>Hi <script>evil()</script>there</p></body></html>`)
	res, err := ExtractManualBytes("page.html", "text/html", body)
	if err != nil {
		t.Fatal(err)
	}
	if res.Title != "My page" {
		t.Errorf("Title = %q, want 'My page'", res.Title)
	}
	if strings.Contains(res.BodyText, "evil") {
		t.Errorf("BodyText leaked script: %q", res.BodyText)
	}
	if !strings.Contains(res.BodyText, "Hi") || !strings.Contains(res.BodyText, "there") {
		t.Errorf("BodyText missing prose: %q", res.BodyText)
	}
}

func TestExtractManualBytes_UnknownMime(t *testing.T) {
	res, err := ExtractManualBytes("blob.bin", "application/octet-stream", []byte{0xff, 0xfe, 0xfd})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Warnings) == 0 {
		t.Errorf("expected unsupported-mime warning, got none")
	}
	if res.BodyText != "" {
		t.Errorf("expected empty body for unknown mime, got %q", res.BodyText)
	}
}

func TestDetectManualMimeType(t *testing.T) {
	cases := map[string]string{
		"report.pdf":     "application/pdf",
		"contract.docx":  "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"notes.md":       "text/markdown",
		"data.csv":       "text/csv",
		"snapshot.json":  "application/json",
		"index.html":     "text/html",
		"plain.txt":      "text/plain",
	}
	for name, want := range cases {
		got := DetectManualMimeType(name, []byte{})
		if got != want {
			t.Errorf("DetectManualMimeType(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestManualSnippet(t *testing.T) {
	body := strings.Repeat("intro ", 50) + "the magic word is XYLOPHONE here. " + strings.Repeat("outro ", 50)

	t.Run("centers on match with ellipses", func(t *testing.T) {
		s := manualSnippet(body, "XYLOPHONE", 80)
		if !strings.Contains(s, "XYLOPHONE") {
			t.Fatalf("snippet missing match: %q", s)
		}
		if !strings.HasPrefix(s, "…") {
			t.Errorf("expected leading ellipsis when match is mid-doc: %q", s)
		}
		if !strings.HasSuffix(s, "…") {
			t.Errorf("expected trailing ellipsis when match is mid-doc: %q", s)
		}
		if utf8Len(s) > 100 {
			t.Errorf("snippet too long: %d chars (%q)", utf8Len(s), s)
		}
	})

	t.Run("short body returned as is", func(t *testing.T) {
		short := "Hello world"
		got := manualSnippet(short, "world", 240)
		if got != short {
			t.Errorf("snippet=%q, want %q", got, short)
		}
	})

	t.Run("query missing falls back to leading slice", func(t *testing.T) {
		s := manualSnippet(body, "absent-word", 40)
		if strings.Contains(s, "magic") {
			t.Errorf("fallback should not have hit the body middle: %q", s)
		}
		if utf8Len(s) > 60 {
			t.Errorf("fallback snippet too long: %q", s)
		}
	})
}

func utf8Len(s string) int {
	n := 0
	for range s {
		n++
	}
	return n
}

func TestParseManualCollectionConfig(t *testing.T) {
	raw, err := MarshalManualCollectionConfig(ManualCollectionConfig{Label: "X", Description: "Y"})
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := ParseManualCollectionConfig(raw)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Label != "X" || cfg.Description != "Y" {
		t.Errorf("roundtrip mismatch: %+v", cfg)
	}
	// Empty input must return zero, no error.
	cfg2, err := ParseManualCollectionConfig(nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg2.Label != "" {
		t.Errorf("expected zero, got %+v", cfg2)
	}
}
