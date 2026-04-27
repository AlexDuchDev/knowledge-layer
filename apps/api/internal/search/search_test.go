package search

import "testing"

func TestTrustLine(t *testing.T) {
	s := trustLine("derived", "draft", "unknown")
	if s == "" {
		t.Fatal("empty trust line")
	}
}
