package integration

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/shruggietech/go-scheduler/internal/domain"
	"github.com/shruggietech/go-scheduler/internal/store"
)

func TestStore_CRUDAndDurability(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	// Group → Schedule → Task → Run → Alert.
	grp := &domain.Group{Name: "Backups", Enabled: true}
	if err := st.CreateGroup(grp); err != nil {
		t.Fatalf("create group: %v", err)
	}
	if grp.ID == "" {
		t.Fatal("expected generated group ID")
	}

	runAt := time.Date(2026, 8, 4, 9, 0, 0, 0, time.UTC)
	sch := &domain.Schedule{Kind: domain.ScheduleOneOff, RunAt: &runAt, HumanSummary: "Once on Aug 4 2026 09:00 UTC"}
	if err := st.CreateSchedule(sch); err != nil {
		t.Fatalf("create schedule: %v", err)
	}

	task := &domain.Task{
		Name: "send-card", GroupID: grp.ID, Command: "/usr/bin/send", Args: []string{"--to", "x"},
		Env: map[string]string{"K": "V"}, Enabled: true, Timezone: "America/New_York",
		ScheduleID: sch.ID, OverlapPolicy: domain.OverlapQueueOne, CatchupPolicy: domain.CatchupOne,
		State: domain.TaskActive,
	}
	if err := st.CreateTask(task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	exit := 0
	now := time.Now().UTC()
	run := &domain.Run{TaskID: task.ID, ScheduledFor: runAt, StartedAt: &now, EndedAt: &now,
		Outcome: domain.OutcomeSuccess, ExitCode: &exit, Trigger: domain.TriggerSchedule}
	if err := st.CreateRun(run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	al := &domain.Alert{TaskID: task.ID, Severity: domain.SeverityWarning, Kind: domain.AlertOverlapQueued, Message: "queued"}
	if err := st.CreateAlert(al); err != nil {
		t.Fatalf("create alert: %v", err)
	}

	// Close and reopen to prove durability across restart.
	if err := st.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	st2, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer st2.Close()

	gotTask, err := st2.GetTask(task.ID)
	if err != nil {
		t.Fatalf("get task after reopen: %v", err)
	}
	if gotTask.Name != "send-card" || gotTask.Timezone != "America/New_York" {
		t.Fatalf("task fields not durable: %+v", gotTask)
	}
	if len(gotTask.Args) != 2 || gotTask.Args[1] != "x" || gotTask.Env["K"] != "V" {
		t.Fatalf("task args/env not durable: %+v", gotTask)
	}

	gotSched, err := st2.GetSchedule(sch.ID)
	if err != nil {
		t.Fatalf("get schedule: %v", err)
	}
	if gotSched.RunAt == nil || !gotSched.RunAt.Equal(runAt) {
		t.Fatalf("schedule run_at not durable: %+v", gotSched)
	}

	runs, err := st2.ListRuns(task.ID, 0)
	if err != nil || len(runs) != 1 {
		t.Fatalf("expected 1 run, got %d (err %v)", len(runs), err)
	}
	if runs[0].Outcome != domain.OutcomeSuccess || runs[0].ExitCode == nil || *runs[0].ExitCode != 0 {
		t.Fatalf("run fields not durable: %+v", runs[0])
	}

	alerts, err := st2.ListAlerts(true)
	if err != nil || len(alerts) != 1 {
		t.Fatalf("expected 1 unacked alert, got %d (err %v)", len(alerts), err)
	}
}

func TestStore_NotFoundAndDeletes(t *testing.T) {
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()

	if _, err := st.GetTask("missing"); err != store.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if err := st.DeleteTask("missing"); err != store.ErrNotFound {
		t.Fatalf("expected ErrNotFound on delete, got %v", err)
	}

	g := &domain.Group{Name: "G", Enabled: true}
	_ = st.CreateGroup(g)
	if err := st.SetGroupEnabled(g.ID, false); err != nil {
		t.Fatalf("disable group: %v", err)
	}
	got, _ := st.GetGroup(g.ID)
	if got.Enabled {
		t.Fatal("group should be disabled")
	}
}
