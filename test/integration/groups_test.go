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

// TestGroups_NestingAndCascade covers US3: a 3-level group hierarchy, and that
// disabling an ancestor group stops its tasks from being scheduled (and
// re-enabling restores them) — without mutating the task's own enabled flag.
func TestGroups_NestingAndCascade(t *testing.T) {
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	// Backups -> Database -> (task lives under Database). 3 levels.
	backups := &domain.Group{Name: "Backups", Enabled: true}
	if err := st.CreateGroup(backups); err != nil {
		t.Fatal(err)
	}
	database := &domain.Group{Name: "Database", ParentID: backups.ID, Enabled: true}
	if err := st.CreateGroup(database); err != nil {
		t.Fatal(err)
	}
	nightly := &domain.Group{Name: "Nightly", ParentID: database.ID, Enabled: true}
	if err := st.CreateGroup(nightly); err != nil {
		t.Fatal(err)
	}

	// Parent must exist; cycles rejected.
	if err := st.CreateGroup(&domain.Group{Name: "bad", ParentID: "nope"}); err == nil {
		t.Fatal("creating a group under a non-existent parent should fail")
	}
	if err := st.SetGroupParent(backups.ID, nightly.ID); err != store.ErrCycle {
		t.Fatalf("reparenting an ancestor under its descendant should be ErrCycle, got %v", err)
	}

	base := time.Date(2026, 6, 19, 8, 0, 0, 0, time.UTC)
	sch := &domain.Schedule{Kind: domain.ScheduleRecurring, RRULE: "FREQ=HOURLY;INTERVAL=2", Anchor: &base}
	if err := st.CreateSchedule(sch); err != nil {
		t.Fatal(err)
	}
	task := &domain.Task{
		Name: "dump", Command: "x", Enabled: true, Timezone: "UTC", GroupID: nightly.ID,
		ScheduleID: sch.ID, OverlapPolicy: domain.OverlapQueueOne, CatchupPolicy: domain.CatchupNone,
		State: domain.TaskActive,
	}
	if err := st.CreateTask(task); err != nil {
		t.Fatal(err)
	}

	// Disable the top group → the deeply-nested task must not run.
	if err := st.SetGroupEnabled(backups.ID, false); err != nil {
		t.Fatal(err)
	}

	fc := clock.NewFake(base)
	ran := make(chan domain.Run, 4)
	eng := engine.New(st, fc, recordingRunner{}, quietLogger(), 2)
	eng.SetOnRun(func(r domain.Run) { ran <- r })
	ctx, cancel := context.WithCancel(context.Background())
	go func() { _ = eng.Start(ctx) }()

	// With the ancestor disabled there is nothing to schedule, so the engine
	// should not arm a timer at all.
	time.Sleep(150 * time.Millisecond)
	if w := fc.Waiters(); w != 0 {
		t.Fatalf("expected no scheduled runs while ancestor group disabled, got %d waiters", w)
	}
	fc.Advance(4 * time.Hour)
	select {
	case r := <-ran:
		t.Fatalf("task ran despite disabled ancestor group: %+v", r)
	case <-time.After(150 * time.Millisecond):
	}

	// Re-enable the top group and reload → the task schedules again.
	if err := st.SetGroupEnabled(backups.ID, true); err != nil {
		t.Fatal(err)
	}
	eng.Reload()
	waitWaiter(t, fc)
	fc.Advance(2 * time.Hour)
	select {
	case <-ran:
	case <-time.After(2 * time.Second):
		t.Fatal("task did not resume after re-enabling its ancestor group")
	}
	cancel()

	// The task's own enabled flag was never mutated by group cascade.
	got, _ := st.GetTask(task.ID)
	if !got.Enabled {
		t.Fatal("group cascade should not mutate the task's own enabled flag")
	}
}
