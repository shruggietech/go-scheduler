package schedule

import (
	"testing"
	"time"

	"github.com/shruggietech/go-scheduler/internal/domain"
)

func TestNextRun_Interval(t *testing.T) {
	sch, err := Parse("every 15 minutes", "UTC", now)
	if err != nil {
		t.Fatal(err)
	}
	after := time.Date(2026, 6, 19, 8, 7, 0, 0, time.UTC)
	got, ok, err := NextRun(sch, "UTC", after)
	if err != nil || !ok {
		t.Fatalf("NextRun: ok=%v err=%v", ok, err)
	}
	// Anchored at 08:00, every 15m → next after 08:07 is 08:15.
	if want := time.Date(2026, 6, 19, 8, 15, 0, 0, time.UTC); !got.Equal(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestNextRun_IntervalAnchored(t *testing.T) {
	// "every 15 minutes starting at 09:00" must align to :00/:15/:30/:45 regardless of
	// the evaluation moment — next run after 09:07 is 09:15, not 09:22.
	sch, err := Parse("every 15 minutes starting at 09:00", "UTC", now)
	if err != nil {
		t.Fatal(err)
	}
	after := time.Date(2026, 6, 19, 9, 7, 0, 0, time.UTC)
	got, ok, err := NextRun(sch, "UTC", after)
	if err != nil || !ok {
		t.Fatalf("NextRun: ok=%v err=%v", ok, err)
	}
	if want := time.Date(2026, 6, 19, 9, 15, 0, 0, time.UTC); !got.Equal(want) {
		t.Fatalf("got %v, want %v (anchored to :15)", got, want)
	}
}

func TestNextRun_IntervalUnanchoredUnchanged(t *testing.T) {
	// Without an anchor clause, behavior matches the creation-aligned default (anchor=now=08:00).
	sch, err := Parse("every 15 minutes", "UTC", now)
	if err != nil {
		t.Fatal(err)
	}
	after := time.Date(2026, 6, 19, 8, 7, 0, 0, time.UTC)
	got, ok, err := NextRun(sch, "UTC", after)
	if err != nil || !ok {
		t.Fatalf("NextRun: ok=%v err=%v", ok, err)
	}
	if want := time.Date(2026, 6, 19, 8, 15, 0, 0, time.UTC); !got.Equal(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestNextRun_OrdinalWeekday(t *testing.T) {
	// "3rd Wednesday of every month at 14:00" — in June 2026 that is June 17.
	sch, err := Parse("3rd wednesday monthly at 14:00", "America/New_York", time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	after := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	got, ok, err := NextRun(sch, "America/New_York", after)
	if err != nil || !ok {
		t.Fatalf("NextRun: ok=%v err=%v", ok, err)
	}
	// June 17 2026 14:00 EDT = 18:00 UTC.
	if want := time.Date(2026, 6, 17, 18, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Fatalf("got %v, want %v (3rd Wed June 2026 14:00 EDT)", got, want)
	}
}

func TestNextRun_Weekdays(t *testing.T) {
	// "weekdays at 09:00" — 2026-06-19 is a Friday; next weekday run after Sat is Monday 22nd.
	sch, err := Parse("weekdays at 09:00", "UTC", now)
	if err != nil {
		t.Fatal(err)
	}
	sat := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	got, ok, err := NextRun(sch, "UTC", sat)
	if err != nil || !ok {
		t.Fatalf("NextRun: ok=%v err=%v", ok, err)
	}
	if want := time.Date(2026, 6, 22, 9, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Fatalf("got %v, want Monday %v", got, want)
	}
}

func TestUpcomingRuns(t *testing.T) {
	sch, _ := Parse("every day at 09:00", "UTC", time.Date(2026, 6, 19, 0, 0, 0, 0, time.UTC))
	runs, err := UpcomingRuns(sch, "UTC", time.Date(2026, 6, 19, 0, 0, 0, 0, time.UTC), 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 3 {
		t.Fatalf("want 3 runs, got %d", len(runs))
	}
	for i, r := range runs {
		if r.Hour() != 9 {
			t.Fatalf("run %d not at 09:00 UTC: %v", i, r)
		}
	}
}

func TestNextRun_OneOff(t *testing.T) {
	at := time.Date(2026, 8, 4, 9, 0, 0, 0, time.UTC)
	sch := NewOneOff(at)
	if sch.Kind != domain.ScheduleOneOff {
		t.Fatalf("kind = %v", sch.Kind)
	}
	// Before the time → returns it.
	got, ok, err := NextRun(sch, "UTC", time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC))
	if err != nil || !ok || !got.Equal(at) {
		t.Fatalf("one-off next: got=%v ok=%v err=%v", got, ok, err)
	}
	// After the time → no further run (one-off does not recur).
	_, ok, _ = NextRun(sch, "UTC", time.Date(2026, 8, 5, 0, 0, 0, 0, time.UTC))
	if ok {
		t.Fatal("one-off should not recur after its run time")
	}
}
