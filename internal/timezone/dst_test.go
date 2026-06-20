package timezone

import (
	"testing"
	"time"
)

func mustLoad(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := Resolve(name)
	if err != nil {
		t.Fatalf("resolve %s: %v", name, err)
	}
	return loc
}

func TestResolve_LocalAndNamed(t *testing.T) {
	if loc, _ := Resolve("Local"); loc != time.Local {
		t.Fatal("Local should map to time.Local")
	}
	if loc, _ := Resolve(""); loc != time.Local {
		t.Fatal("empty should map to time.Local")
	}
	if _, err := Resolve("Mars/Phobos"); err == nil {
		t.Fatal("invalid zone should error")
	}
}

// US spring-forward 2026: 2026-03-08 02:00 -> 03:00 in America/New_York.
// A 02:30 task should run at the next valid instant, 03:00 EDT (07:00 UTC).
func TestWallTime_SpringForwardNextValid(t *testing.T) {
	ny := mustLoad(t, "America/New_York")
	got := WallTime(ny, 2026, time.March, 8, 2, 30, 0).UTC()
	want := time.Date(2026, time.March, 8, 7, 0, 0, 0, time.UTC) // 03:00 EDT
	if !got.Equal(want) {
		t.Fatalf("spring-forward: got %v, want %v (03:00 EDT)", got, want)
	}
}

// US fall-back 2026: 2026-11-01 02:00 -> 01:00. 01:30 occurs twice.
// We want the FIRST occurrence: 01:30 EDT (05:30 UTC), not 01:30 EST (06:30 UTC).
func TestWallTime_FallBackFirstOccurrence(t *testing.T) {
	ny := mustLoad(t, "America/New_York")
	got := WallTime(ny, 2026, time.November, 1, 1, 30, 0).UTC()
	want := time.Date(2026, time.November, 1, 5, 30, 0, 0, time.UTC) // 01:30 EDT
	if !got.Equal(want) {
		t.Fatalf("fall-back: got %v, want %v (01:30 EDT, first occurrence)", got, want)
	}
}

// A normal (non-DST) time resolves to itself.
func TestWallTime_NormalTime(t *testing.T) {
	ny := mustLoad(t, "America/New_York")
	got := WallTime(ny, 2026, time.June, 19, 9, 0, 0).UTC()
	want := time.Date(2026, time.June, 19, 13, 0, 0, 0, time.UTC) // 09:00 EDT
	if !got.Equal(want) {
		t.Fatalf("normal time: got %v, want %v", got, want)
	}
}
