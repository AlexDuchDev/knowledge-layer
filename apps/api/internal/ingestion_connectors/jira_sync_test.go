package ingestion_connectors

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/knowledgelayer/api/internal/ingestion_connectors/families/docs_wiki"
)

func TestListJiraProjects(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/project/search" {
			t.Fatalf("path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"values":[{"id":"1","key":"KL","name":"Knowledge"}]}`))
	}))
	defer srv.Close()
	cfg := &jiraFeedConfig{SiteBaseURL: srv.URL, Email: "e@x.com", APIToken: "tok"}
	s := &Service{HTTP: srv.Client()}
	list, err := s.ListJiraProjects(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Key != "KL" {
		t.Fatalf("%+v", list)
	}
}

func TestParseJiraFeedConfig(t *testing.T) {
	_, err := ParseJiraFeedConfig([]byte(`{}`))
	if err == nil {
		t.Fatal("expected error")
	}
	_, err = ParseJiraFeedConfig([]byte(`{"jira_site_base_url":"https://x.atlassian.net","jira_email":"a@b.c","jira_api_token":"t"}`))
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidateJiraSourceFeedForActivation(t *testing.T) {
	feed := &SourceFeed{
		ConnectorConfigJSON: json.RawMessage(`{"jira_site_base_url":"https://x.atlassian.net","jira_email":"a@b.c","jira_api_token":"t"}`),
		ExternalRef:         "PROJ",
	}
	if err := ValidateJiraSourceFeedForActivation(feed); err != nil {
		t.Fatal(err)
	}
	feed.ExternalRef = ""
	if ValidateJiraSourceFeedForActivation(feed) == nil {
		t.Fatal("expected error")
	}
}

func TestJiraDescriptionPlain_ADF(t *testing.T) {
	v := map[string]any{
		"content": []any{
			map[string]any{"type": "text", "text": "hello"},
		},
	}
	if got := jiraDescriptionPlain(v); got != "hello" {
		t.Fatalf("%q", got)
	}
}

func TestValidateMicrosoft365SourceFeedForActivation_teamsRef(t *testing.T) {
	raw := json.RawMessage(`{"m365_product":"teams","graph_access_token":"tok"}`)
	feed := &SourceFeed{ConnectorConfigJSON: raw, ExternalRef: "a|b"}
	if err := ValidateMicrosoft365SourceFeedForActivation(feed); err != nil {
		t.Fatal(err)
	}
	feed.ExternalRef = "bad"
	if ValidateMicrosoft365SourceFeedForActivation(feed) == nil {
		t.Fatal("expected error")
	}
}

func TestValidateNotionV1ForActivation(t *testing.T) {
	feed := &SourceFeed{
		ConnectorConfigJSON: json.RawMessage(`{"notion_integration_token":"sec","scope":"page"}`),
		ExternalRef:         "page-uuid",
	}
	token, err := docs_wiki.RequireNotionToken(feed.ConnectorConfigJSON)
	if err != nil {
		t.Fatal(err)
	}
	scopeCfg, err := docs_wiki.ParseNotionConfig(feed.ConnectorConfigJSON)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateNotionV1ForActivation(feed, token, scopeCfg); err != nil {
		t.Fatal(err)
	}
}

func TestValidateNotionV1ForActivation_badScope(t *testing.T) {
	feed := &SourceFeed{
		ConnectorConfigJSON: json.RawMessage(`{"notion_integration_token":"sec","scope":"invalid"}`),
		ExternalRef:         "x",
	}
	token, _ := docs_wiki.RequireNotionToken(feed.ConnectorConfigJSON)
	scopeCfg, _ := docs_wiki.ParseNotionConfig(feed.ConnectorConfigJSON)
	if ValidateNotionV1ForActivation(feed, token, scopeCfg) == nil {
		t.Fatal("expected error")
	}
}
