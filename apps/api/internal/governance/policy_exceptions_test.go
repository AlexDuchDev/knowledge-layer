package governance

import (
	"testing"
	"time"
)

func TestEffectiveExceptionStatus(t *testing.T) {
	now := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	if got := effectiveExceptionStatus("revoked", nil, now); got != "revoked" {
		t.Fatalf("revoked: got %q", got)
	}
	if got := effectiveExceptionStatus("active", &past, now); got != "expired" {
		t.Fatalf("past expiry: got %q", got)
	}
	if got := effectiveExceptionStatus("active", &future, now); got != "active" {
		t.Fatalf("future expiry: got %q", got)
	}
	if got := effectiveExceptionStatus("active", nil, now); got != "active" {
		t.Fatalf("no expiry: got %q", got)
	}
}
