package ingestion_connectors

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestValidateConfluenceSourceFeedForActivation(t *testing.T) {
	raw := json.RawMessage(`{"confluence_base_url":"https://x.atlassian.net/wiki","confluence_auth":"tok","confluence_feed_kind":"space"}`)
	feed := &SourceFeed{ConnectorConfigJSON: raw, ExternalRef: "ENG"}
	if err := ValidateConfluenceSourceFeedForActivation(feed); err != nil {
		t.Fatal(err)
	}
	feed.ExternalRef = ""
	if ValidateConfluenceSourceFeedForActivation(feed) == nil {
		t.Fatal("expected error")
	}
}

func TestValidateConfluenceSourceFeedForActivation_pageCollection(t *testing.T) {
	raw := json.RawMessage(`{"confluence_base_url":"https://x.atlassian.net/wiki","confluence_auth":"tok","confluence_feed_kind":"page_collection"}`)
	feed := &SourceFeed{ConnectorConfigJSON: raw, ExternalRef: "1, 2"}
	if err := ValidateConfluenceSourceFeedForActivation(feed); err != nil {
		t.Fatal(err)
	}
	feed.ExternalRef = ","
	if ValidateConfluenceSourceFeedForActivation(feed) == nil {
		t.Fatal("expected error")
	}
}

func TestConfluencePageParsed_links(t *testing.T) {
	raw := `{"id":"99","title":"Hello","type":"page","_links":{"webui":"/spaces/ENG/pages/99"}}`
	var p confluencePageParsed
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		t.Fatal(err)
	}
	if p.Links.WebUI != "/spaces/ENG/pages/99" || p.Title != "Hello" {
		t.Fatalf("%+v", p)
	}
}

func TestConfluenceGET_401(t *testing.T) {
	s := &Service{
		HTTP: &http.Client{
			Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: 401,
					Body:       io.NopCloser(strings.NewReader(`{"message":"no"}`)),
					Header:     make(http.Header),
				}, nil
			}),
		},
	}
	_, code, err := s.confluenceGET(context.Background(), "https://example.test/wiki/rest/api/content", "bad")
	if err != nil || code != 401 {
		t.Fatalf("code=%d err=%v", code, err)
	}
}
