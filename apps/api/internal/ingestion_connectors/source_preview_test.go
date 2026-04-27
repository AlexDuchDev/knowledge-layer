package ingestion_connectors

import (
	"context"
	"testing"
)

func TestPreviewRequiresDraft(t *testing.T) {
	s := &Service{}
	feed := &SourceFeed{Status: "active"}
	conn := &Connector{Type: "telegram"}
	_, err := s.PreviewSourceFeed(context.Background(), feed, conn)
	if err == nil {
		t.Fatal("expected error for non-draft feed")
	}
}
