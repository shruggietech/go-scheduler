package integration

import (
	"context"
	"testing"
	"time"

	"github.com/shruggietech/go-scheduler/internal/clock"
	"github.com/shruggietech/go-scheduler/internal/domain"
	"github.com/shruggietech/go-scheduler/internal/engine"
	"github.com/shruggietech/go-scheduler/internal/store"
)

// seedTaskWithRun creates an hourly task with one prior run at lastRun, so the
// engine has a baseline to detect missed runs against.
func seedTaskWithRun(t *testing.T, st *store.Store, policy domain.CatchupPolicy, anchor, lastRun time.Time) *domain.Task {
	t.Helper()
	sch := &domain.Schedule{Kind: domain.ScheduleRecurring, RRULE: "FREQ=HOURLY;INTERVAL=1", Anchor: &anchor}
	if err := st.CreateSchedule(sch); err != nil {
		t.Fatal(err)
	}
	task := &domain.Task{
		Name: "job", Command: "x", Enabled: true, Timezone: "UTC", ScheduleID: sch.ID,
		OverlapPolicy: domain.OverlapQueueOne, CatchupPolicy: policy, State: domain.TaskActive,
	}
	if err := st.CreateTask(task); err != nil {
		t.Fatal(err)
	}
	end := lastRun
	if err := st.CreateRun(&domain.Run{TaskID: task.ID, ScheduledFor: lastRun, EndedAt: &end,
		Outcome: domain.OutcomeSuccess, Trigger: domain.TriggerSchedule}); err != nil {
		t.Fatal(err)
	}
	return task
}

func countTrigger(t *testing.T, st *store.Store, taskID string, trig domain.RunTrigger) int {
	t.Helper()
	runs, err := st.ListRuns(taskID, 0)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, r := range runs {
		if r.Trigger == trig {
			n++
		}
	}
	return n
}

// TestCatchup_OneRunThenResume covers US5: after downtime that missed several
// runs, exactly one catch-up run occurs and then normal scheduling resumes.
func TestCatchup_OneRunThenResume(t *testing.T) {
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	anchor := time.Date(2026, 6, 19, 8, 0, 0, 0, time.UTC)
	lastRun := time.Date(2026, 6, 19, 9, 0, 0, 0, time.UTC)
	// Daemon "restarts" at 12:30 — runs at 10:00, 11:00, 12:00 were missed.
	startNow := time.Date(2026, 6, 19, 12, 30, 0, 0, time.UTC)

	task := seedTaskWithRun(t, st, domain.CatchupOne, anchor, lastRun)

	fc := clock.NewFake(startNow)
	ran := make(chan domain.Run, 16)
	eng := engine.New(st, fc, recordingRunner{}, quietLogger(), 4)
	eng.SetOnRun(func(r domain.Run) { ran <- r })
	ctx, cancel := context.WithCancel(context.Background())
	go func() { _ = eng.Start(ctx) }()

	// Exactly one catch-up run should fire at startup.
	r := waitSignal(t, ran, "catch-up run")
	if r.Trigger != domain.TriggerCatchup {
		t.Fatalf("first run should be a catch-up, got trigger %s", r.Trigger)
	}

	// Give the engine a moment; ensure no second catch-up.
	time.Sleep(150 * time.Millisecond)
	if n := countTrigger(t, st, task.ID, domain.TriggerCatchup); n != 1 {
		t.Fatalf("expected exactly 1 catch-up run, got %d", n)
	}

	// Normal scheduling resumes: next run is at 13:00 (after startNow).
	waitWaiter(t, fc)
	fc.Advance(40 * time.Minute) // -> 13:10, past 13:00
	r2 := waitSignal(t, ran, "resumed scheduled run")
	if r2.Trigger != domain.TriggerSchedule {
		t.Fatalf("resumed run should be a normal scheduled run, got %s", r2.Trigger)
	}
	if want := time.Date(2026, 6, 19, 13, 0, 0, 0, time.UTC); !r2.ScheduledFor.Equal(want) {
		t.Fatalf("resumed run scheduled_for = %v, want %v", r2.ScheduledFor, want)
	}
	cancel()
}

// TestCatchup_PolicyNone confirms a task with catchup=none performs no catch-up.
func TestCatchup_PolicyNone(t *testing.T) {
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	anchor := time.Date(2026, 6, 19, 8, 0, 0, 0, time.UTC)
	lastRun := time.Date(2026, 6, 19, 9, 0, 0, 0, time.UTC)
	startNow := time.Date(2026, 6, 19, 12, 30, 0, 0, time.UTC)
	task := seedTaskWithRun(t, st, domain.CatchupNone, anchor, lastRun)

	fc := clock.NewFake(startNow)
	eng := engine.New(st, fc, recordingRunner{}, quietLogger(), 4)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = eng.Start(ctx) }()

	time.Sleep(200 * time.Millisecond)
	if n := countTrigger(t, st, task.ID, domain.TriggerCatchup); n != 0 {
		t.Fatalf("policy none should produce 0 catch-up runs, got %d", n)
	}
}
