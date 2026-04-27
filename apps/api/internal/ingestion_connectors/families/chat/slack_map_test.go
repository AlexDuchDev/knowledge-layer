package chat

import (
	"testing"

	"github.com/google/uuid"
)

func TestFromSlackMessage_threadAndTime(t *testing.T) {
	id := uuid.MustParse("10000000-0000-0000-0000-000000000001")
	m := FromSlackMessage(id, "C1", "1234567890.000001", "U9", "hello", "1234567890.000001")
	if m.ExternalThreadID != "" {
		t.Fatal("root should not set thread id")
	}
	if m.PostedAt == nil || m.PostedAt.UTC().Unix() != 1234567890 {
		t.Fatalf("time %v", m.PostedAt)
	}
	m2 := FromSlackMessage(id, "C1", "1234567890.000002", "U9", "reply", "1234567890.000001")
	if m2.ExternalThreadID != "1234567890.000001" {
		t.Fatal(m2.ExternalThreadID)
	}
}

func TestSlackTsUnixSeconds(t *testing.T) {
	sec, ok := slackTsUnixSeconds("not-a-ts")
	if ok || sec != 0 {
		t.Fatal()
	}
	sec, ok = slackTsUnixSeconds("42.99")
	if !ok || sec != 42 {
		t.Fatal(sec, ok)
	}
}

func TestFromSlackMessage_emptyTs(t *testing.T) {
	id := uuid.New()
	m := FromSlackMessage(id, "C1", "", "U1", "x", "")
	if m.PostedAt != nil {
		t.Fatal()
	}
}
