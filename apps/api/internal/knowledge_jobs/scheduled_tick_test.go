package knowledge_jobs

import (
	"testing"
	"time"

	"github.com/robfig/cron/v3"
)

func TestScheduleFiresThisUTCMinute_hourlyOnTheHour(t *testing.T) {
	sched, err := cron.ParseStandard("0 * * * *")
	if err != nil {
		t.Fatal(err)
	}
	// 2024-06-01 14:00:30 UTC — should fire in minute 14:00
	now := time.Date(2024, 6, 1, 14, 0, 30, 0, time.UTC)
	if !ScheduleFiresThisUTCMinute(sched, now) {
		t.Fatal("expected fire at 14:00 UTC minute")
	}
	// 14:01 should not fire for hourly-at-0
	now2 := time.Date(2024, 6, 1, 14, 1, 0, 0, time.UTC)
	if ScheduleFiresThisUTCMinute(sched, now2) {
		t.Fatal("expected no fire at 14:01")
	}
}

func TestScheduleFiresThisUTCMinute_everyFiveMinutes(t *testing.T) {
	sched, err := cron.ParseStandard("*/5 * * * *")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2024, 6, 1, 10, 15, 0, 0, time.UTC)
	if !ScheduleFiresThisUTCMinute(sched, now) {
		t.Fatal("expected fire on */5 at :15")
	}
	now2 := time.Date(2024, 6, 1, 10, 16, 0, 0, time.UTC)
	if ScheduleFiresThisUTCMinute(sched, now2) {
		t.Fatal("expected no fire at :16")
	}
}
