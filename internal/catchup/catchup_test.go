package catchup

import (
	"testing"
	"time"

	"github.com/shruggietech/go-scheduler/internal/domain"
)

func hourly(anchor time.Time) domain.Schedule {
	return domain.Schedule{Kind: domain.ScheduleRecurring, RRULE: "FREQ=HOURLY;INTERVAL=1", Anchor: &anchor}
}

func TestEvaluate_MissedRunTriggersCatchup(t *testing.T) {
	anchor := time.Date(2026, 6, 19, 8, 0, 0, 0, time.UTC)
	sch := hourly(anchor)
	last := time.Date(2026, 6, 19, 9, 0, 0, 0, time.UTC)  // last run at 09:00
	now := time.Date(2026, 6, 19, 12, 30, 0, 0, time.UTC) // 3+ hours later (downtime)

	dec, err := Evaluate(sch, "UTC", last, true, domain.CatchupOne, now)
	if err != nil {
		t.Fatal(err)
	}
	if !dec.ShouldCatchUp {
		t.Fatal("expected catch-up after missing runs during downtime")
	}
	if want := time.Date(2026, 6, 19, 10, 0, 0, 0, time.UTC); !dec.FirstMissed.Equal(want) {
		t.Fatalf("first missed = %v, want %v", dec.FirstMissed, want)
	}
}

func TestEvaluate_NoMissWhenNextIsFuture(t *testing.T) {
	anchor := time.Date(2026, 6, 19, 8, 0, 0, 0, time.UTC)
	sch := hourly(anchor)
	last := time.Date(2026, 6, 19, 9, 0, 0, 0, time.UTC)
	now := time.Date(2026, 6, 19, 9, 30, 0, 0, time.UTC) // before the next (10:00)

	dec, _ := Evaluate(sch, "UTC", last, true, domain.CatchupOne, now)
	if dec.ShouldCatchUp {
		t.Fatal("no catch-up expected when the next run is still in the future")
	}
}

func TestEvaluate_PolicyNone(t *testing.T) {
	anchor := time.Date(2026, 6, 19, 8, 0, 0, 0, time.UTC)
	sch := hourly(anchor)
	last := time.Date(2026, 6, 19, 9, 0, 0, 0, time.UTC)
	now := time.Date(2026, 6, 19, 15, 0, 0, 0, time.UTC) // many missed

	dec, _ := Evaluate(sch, "UTC", last, true, domain.CatchupNone, now)
	if dec.ShouldCatchUp {
		t.Fatal("policy 'none' must never catch up")
	}
}

func TestEvaluate_NoPriorRun(t *testing.T) {
	anchor := time.Date(2026, 6, 19, 8, 0, 0, 0, time.UTC)
	sch := hourly(anchor)
	now := time.Date(2026, 6, 19, 15, 0, 0, 0, time.UTC)

	dec, _ := Evaluate(sch, "UTC", time.Time{}, false, domain.CatchupOne, now)
	if dec.ShouldCatchUp {
		t.Fatal("a never-run task has nothing to catch up")
	}
}
