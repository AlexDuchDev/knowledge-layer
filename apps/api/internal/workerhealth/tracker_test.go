package workerhealth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hibiken/asynq"
)

func TestTracker_MarkAndSnapshot(t *testing.T) {
	tr := NewTracker()
	if got := tr.Snapshot(); len(got) != 0 {
		t.Fatalf("expected empty snapshot, got %v", got)
	}
	tr.Mark("task:a")
	tr.Mark("task:b")
	snap := tr.Snapshot()
	if len(snap) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(snap))
	}
	if snap["task:a"].IsZero() || snap["task:b"].IsZero() {
		t.Fatalf("expected non-zero timestamps, got %v", snap)
	}
}

func TestTracker_Wrap_OnlyMarksOnSuccess(t *testing.T) {
	tr := NewTracker()
	okHandler := func(context.Context, *asynq.Task) error { return nil }
	failHandler := func(context.Context, *asynq.Task) error { return errors.New("boom") }

	wrappedOK := tr.Wrap("task:ok", okHandler)
	wrappedFail := tr.Wrap("task:fail", failHandler)

	if err := wrappedOK(context.Background(), nil); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if err := wrappedFail(context.Background(), nil); err == nil {
		t.Fatal("expected error from wrapped failing handler")
	}

	snap := tr.Snapshot()
	if _, ok := snap["task:ok"]; !ok {
		t.Fatalf("expected task:ok marked after success, got %v", snap)
	}
	if _, ok := snap["task:fail"]; ok {
		t.Fatalf("expected task:fail NOT marked after failure (would mask stuck workers), got %v", snap)
	}
}

func TestTracker_NilSafe(t *testing.T) {
	var tr *Tracker
	tr.Mark("task:x")
	if got := tr.Snapshot(); got != nil {
		t.Fatalf("expected nil snapshot from nil tracker, got %v", got)
	}
}

func TestBuildSnapshot_OmitsUnconfiguredSections(t *testing.T) {
	cfg := Config{
		WorkerName: "testworker",
		StartedAt:  time.Now().Add(-2 * time.Second).UTC(),
	}
	snap := BuildSnapshot(context.Background(), cfg)
	if snap.Worker != "testworker" {
		t.Fatalf("worker name: %q", snap.Worker)
	}
	if snap.Database != "unconfigured" {
		t.Fatalf("expected database=unconfigured, got %q", snap.Database)
	}
	if snap.Redis != "" {
		t.Fatalf("expected redis empty when client nil, got %q", snap.Redis)
	}
	if snap.Queues != nil {
		t.Fatalf("expected nil queues when inspector nil, got %v", snap.Queues)
	}
	if snap.LastByTask != nil {
		t.Fatalf("expected nil last-by-task when tracker nil, got %v", snap.LastByTask)
	}
	if snap.UptimeSec < 1 {
		t.Fatalf("uptime should be at least 1s, got %d", snap.UptimeSec)
	}
}
