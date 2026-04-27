package ingestion_connectors

import (
	"encoding/json"
	"testing"
)

func TestNotionExtractTitle(t *testing.T) {
	raw := json.RawMessage(`{
	  "properties": {
	    "Name": {"type":"title","title":[{"plain_text":"Hello Page"}]}
	  }
	}`)
	var props notionPageProperties
	if err := json.Unmarshal(raw, &props); err != nil {
		t.Fatal(err)
	}
	if got := notionExtractTitle(props); got != "Hello Page" {
		t.Fatal(got)
	}
}

func TestNotionBlockPlainLines_paragraph(t *testing.T) {
	raw := []byte(`{"type":"paragraph","paragraph":{"rich_text":[{"plain_text":"Line A"}]}}`)
	lines := notionBlockPlainLines(raw)
	if len(lines) != 1 || lines[0] != "Line A" {
		t.Fatalf("%#v", lines)
	}
}
