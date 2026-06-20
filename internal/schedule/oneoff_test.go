package schedule

import (
	"testing"
	"time"

	"github.com/shruggietech/go-scheduler/internal/domain"
)

func TestNewOneOff_StoresUTC(t *testing.T) {
	ny, _ := time.LoadLocation("America/New_York")
	local := time.Date(2026, 8, 4, 9, 0, 0, 0, ny)
	sch := NewOneOff(local)
	if sch.RunAt == nil {
		t.Fatal("RunAt should be set")
	}
	if sch.RunAt.Location() != time.UTC {
		t.Fatalf("RunAt should be UTC, got %v", sch.RunAt.Location())
	}
	if !sch.RunAt.Equal(local) {
		t.Fatalf("RunAt instant changed: %v vs %v", sch.RunAt, local)
	}
}

// IsPastOneOff is the rule the API/CLI use to reject a one-off whose time has
// already passed at creation. Verifying the predicate here keeps the policy
// testable independent of the transport layer.
func TestOneOff_PastDetection(t *testing.T) {
	atPast := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	schedNow := time.Date(2026, 6, 19, 0, 0, 0, 0, time.UTC)
	sch := NewOneOff(atPast)

	_, ok, err := NextRun(sch, "UTC", schedNow)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("a past one-off has no next run; creation should be rejected by the caller")
	}
}

func TestOneOff_CompletedHasNoNext(t *testing.T) {
	at := time.Date(2026, 8, 4, 9, 0, 0, 0, time.UTC)
	sch := domain.Schedule{Kind: domain.ScheduleOneOff, RunAt: &at}
	if _, ok, _ := NextRun(sch, "UTC", at.Add(time.Second)); ok {
		t.Fatal("one-off after its instant must not produce a next run")
	}
}
