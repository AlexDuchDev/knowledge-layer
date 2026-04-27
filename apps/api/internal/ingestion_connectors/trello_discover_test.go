package ingestion_connectors

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type trelloMockTransport struct{}

func (trelloMockTransport) RoundTrip(*http.Request) (*http.Response, error) {
	body := `[{"id":"b1","name":"Roadmap","url":"https://trello.com/b/b1"}]`
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}, nil
}

func TestListTrelloBoards(t *testing.T) {
	cfg := &trelloFeedConfig{APIKey: "k", Token: "t"}
	s := &Service{HTTP: &http.Client{Transport: trelloMockTransport{}}}
	list, err := s.ListTrelloBoards(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != "b1" || list[0].Name != "Roadmap" {
		t.Fatalf("%+v", list)
	}
}
