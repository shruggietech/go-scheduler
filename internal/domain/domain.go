// Package domain defines the core entities of the scheduler (Group, Task,
// Schedule, Run, Alert) and their enumerations. Entities are plain data with
// UTC timestamps; persistence lives in internal/store and behavior in the
// engine. Keeping the types in one low-level package avoids import cycles
// between the task, schedule, store, and engine packages.
package domain

import "time"

// ---- Enumerations -------------------------------------------------------

// TaskState is the lifecycle state of a Task.
type TaskState string

const (
	TaskActive    TaskState = "active"
	TaskCompleted TaskState = "completed" // one-off ran, or recurrence exhausted
	TaskDisabled  TaskState = "disabled"
)

// OverlapPolicy controls what happens when a task is still running at its next
// trigger time.
type OverlapPolicy string

const (
	OverlapQueueOne        OverlapPolicy = "queue_one" // default: queue exactly one pending run
	OverlapSkip            OverlapPolicy = "skip"       // skip the new trigger
	OverlapAllowConcurrent OverlapPolicy = "allow_concurrent"
)

// CatchupPolicy controls catch-up behavior after downtime.
type CatchupPolicy string

const (
	CatchupOne  CatchupPolicy = "one"  // default: one catch-up run if any were missed
	CatchupNone CatchupPolicy = "none" // never catch up
)

// ScheduleKind distinguishes the timing model of a Schedule.
type ScheduleKind string

const (
	ScheduleOneOff    ScheduleKind = "one_off"
	ScheduleRecurring ScheduleKind = "recurring"
	ScheduleEvent     ScheduleKind = "event"
)

// RunOutcome is the result of a single execution.
type RunOutcome string

const (
	OutcomeSuccess  RunOutcome = "success"
	OutcomeFailure  RunOutcome = "failure"
	OutcomeSkipped  RunOutcome = "skipped"
	OutcomeCaughtUp RunOutcome = "caught_up"
	OutcomeQueued   RunOutcome = "queued"
)

// RunTrigger records what caused a run.
type RunTrigger string

const (
	TriggerSchedule RunTrigger = "schedule"
	TriggerEvent    RunTrigger = "event"
	TriggerCatchup  RunTrigger = "catchup"
	TriggerManual   RunTrigger = "manual"
)

// AlertSeverity and AlertKind classify surfaced conditions.
type AlertSeverity string

const (
	SeverityInfo    AlertSeverity = "info"
	SeverityWarning AlertSeverity = "warning"
	SeverityError   AlertSeverity = "error"
)

type AlertKind string

const (
	AlertOverlapQueued AlertKind = "overlap_queued"
	AlertRunFailed     AlertKind = "run_failed"
	AlertMissedRun     AlertKind = "missed_run"
	AlertService       AlertKind = "service"
)

// ---- Entities -----------------------------------------------------------

// Group is a named container forming a nested hierarchy. ParentID is empty for
// top-level groups. Disabling cascades to descendants and their tasks.
type Group struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	ParentID  string    `json:"parent_id,omitempty"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Task is a unit of work with a schedule, timezone, and execution policies.
type Task struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	GroupID       string            `json:"group_id,omitempty"`
	Command       string            `json:"command"`
	Args          []string          `json:"args,omitempty"`
	WorkingDir    string            `json:"working_dir,omitempty"`
	Env           map[string]string `json:"env,omitempty"`
	RunAs         string            `json:"run_as,omitempty"`
	Enabled       bool              `json:"enabled"`
	Timezone      string            `json:"timezone"`
	ScheduleID    string            `json:"schedule_id"`
	OverlapPolicy OverlapPolicy     `json:"overlap_policy"`
	CatchupPolicy CatchupPolicy     `json:"catchup_policy"`
	State         TaskState         `json:"state"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

// Schedule is the timing definition for a task. Exactly one of (RRULE+Anchor),
// RunAt, or TriggerID is populated, matching Kind. All times are UTC.
type Schedule struct {
	ID           string       `json:"id"`
	Kind         ScheduleKind `json:"kind"`
	RRULE        string       `json:"rrule,omitempty"`
	Anchor       *time.Time   `json:"anchor,omitempty"`
	RunAt        *time.Time   `json:"run_at,omitempty"`
	TriggerID    string       `json:"trigger_id,omitempty"`
	HumanSummary string       `json:"human_summary"`
}

// Run is a single execution record (append-only history).
type Run struct {
	ID           string     `json:"id"`
	TaskID       string     `json:"task_id"`
	ScheduledFor time.Time  `json:"scheduled_for"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	EndedAt      *time.Time `json:"ended_at,omitempty"`
	Outcome      RunOutcome `json:"outcome"`
	ExitCode     *int       `json:"exit_code,omitempty"`
	Output       string     `json:"output,omitempty"`
	Trigger      RunTrigger `json:"trigger"`
}

// Alert is a surfaced condition shown in the GUI and reflected in logs.
type Alert struct {
	ID           string        `json:"id"`
	TaskID       string        `json:"task_id,omitempty"`
	Severity     AlertSeverity `json:"severity"`
	Kind         AlertKind     `json:"kind"`
	Message      string        `json:"message"`
	CreatedAt    time.Time     `json:"created_at"`
	Acknowledged bool          `json:"acknowledged"`
}
