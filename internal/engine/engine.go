// Package engine is the scheduling core: a single timer-driven loop computes
// the next run for each active task, dispatches due runs through a bounded
// worker pool, and records history. Time is read through an injected Clock so
// behavior is deterministic under test. Overlap handling lives in overlap.go.
package engine

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/shruggietech/go-scheduler/internal/catchup"
	"github.com/shruggietech/go-scheduler/internal/clock"
	"github.com/shruggietech/go-scheduler/internal/domain"
	"github.com/shruggietech/go-scheduler/internal/schedule"
	"github.com/shruggietech/go-scheduler/internal/store"
)

// Runner executes a task and returns its Run record. The executor implements it;
// tests inject a fake.
type Runner interface {
	Run(ctx context.Context, task domain.Task, scheduledFor time.Time, trigger domain.RunTrigger) domain.Run
}

// taskCtx caches a task with its schedule for next-run computation.
type taskCtx struct {
	task domain.Task
	sch  domain.Schedule
}

// Engine schedules and dispatches task runs.
type Engine struct {
	store  *store.Store
	clk    clock.Clock
	runner Runner
	log    *slog.Logger
	sem    chan struct{} // bounded worker pool

	mu      sync.Mutex
	tasks   map[string]taskCtx   // active tasks by ID
	next    map[string]time.Time // next scheduled run (UTC) by task ID
	running map[string]bool
	queued  map[string]time.Time // queued pending run's scheduled time, by task ID

	reload       chan struct{}
	runCtx       context.Context
	runWG        sync.WaitGroup // tracks in-flight runs for graceful drain
	onRun        func(domain.Run)
	onCompletion func(sourceTaskID string, outcome domain.RunOutcome, eventKey string, now time.Time)
	onStartup    func()
	cycleCh      chan time.Time // optional test observation of processed cycles
}

// New constructs an Engine. workers bounds concurrent task executions.
func New(st *store.Store, clk clock.Clock, runner Runner, log *slog.Logger, workers int) *Engine {
	if workers <= 0 {
		workers = 1
	}
	return &Engine{
		store:   st,
		clk:     clk,
		runner:  runner,
		log:     log,
		sem:     make(chan struct{}, workers),
		tasks:   map[string]taskCtx{},
		next:    map[string]time.Time{},
		running: map[string]bool{},
		queued:  map[string]time.Time{},
		reload:  make(chan struct{}, 1),
	}
}

// SetOnRun registers a callback invoked after each run is recorded (used for
// alerts/event streaming and for test synchronization).
func (e *Engine) SetOnRun(f func(domain.Run)) { e.onRun = f }

// SetCompletionHook registers a callback invoked after a run completes with a
// success/failure outcome (used to fire event triggers). eventKey is the run ID.
func (e *Engine) SetCompletionHook(f func(sourceTaskID string, outcome domain.RunOutcome, eventKey string, now time.Time)) {
	e.onCompletion = f
}

// SetStartupHook registers a callback invoked once when the loop starts, after
// the run context is established (used for at-least-once trigger recovery).
func (e *Engine) SetStartupHook(f func()) { e.onStartup = f }

// FireEvent dispatches a target task as an event-triggered run, honoring its
// overlap policy.
func (e *Engine) FireEvent(targetTaskID string) {
	task, err := e.store.GetTask(targetTaskID)
	if err != nil {
		e.log.Error("engine: fire event target", "task", targetTaskID, "err", err)
		return
	}
	e.dispatch(task, e.clk.Now(), domain.TriggerEvent)
}

// enableCycleObservation wires a channel the loop signals after each processing
// cycle. Test-only.
func (e *Engine) enableCycleObservation() <-chan time.Time {
	e.cycleCh = make(chan time.Time, 64)
	return e.cycleCh
}

// Reload asks the loop to recompute schedules from the store (call after tasks
// change). Non-blocking and coalesced.
func (e *Engine) Reload() {
	select {
	case e.reload <- struct{}{}:
	default:
	}
}

// Start runs the scheduling loop until ctx is cancelled, then drains in-flight
// runs. It blocks; run it in a goroutine.
func (e *Engine) Start(ctx context.Context) error {
	e.runCtx = ctx
	e.recompute(e.clk.Now())
	e.runCatchup(e.clk.Now())
	if e.onStartup != nil {
		e.onStartup()
	}
	for {
		d, has := e.untilNext(e.clk.Now())
		var wake <-chan time.Time
		var timer *clock.Timer
		if has {
			timer = e.clk.NewTimer(d)
			wake = timer.C
		}
		select {
		case <-ctx.Done():
			if timer != nil {
				timer.Stop()
			}
			e.runWG.Wait()
			return ctx.Err()
		case <-e.reload:
			if timer != nil {
				timer.Stop()
			}
			e.recompute(e.clk.Now())
		case now := <-wake:
			e.runDue(now)
			if e.cycleCh != nil {
				e.cycleCh <- now
			}
		}
	}
}

// recompute rebuilds the active task set and their next run times from the store.
func (e *Engine) recompute(now time.Time) {
	tasks, err := e.store.ListTasks("", string(domain.TaskActive))
	if err != nil {
		e.log.Error("engine: list tasks", "err", err)
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.tasks = map[string]taskCtx{}
	newNext := map[string]time.Time{}
	for _, task := range tasks {
		if !task.Enabled {
			continue
		}
		// A task is ineligible if any ancestor group is disabled (cascade).
		if ok, err := e.store.GroupChainEnabled(task.GroupID); err == nil && !ok {
			continue
		}
		sch, err := e.store.GetSchedule(task.ScheduleID)
		if err != nil {
			e.log.Error("engine: get schedule", "task", task.ID, "err", err)
			continue
		}
		e.tasks[task.ID] = taskCtx{task: task, sch: sch}
		if n, ok, err := schedule.NextRun(sch, task.Timezone, now); err == nil && ok {
			newNext[task.ID] = n
		}
	}
	e.next = newNext
}

// runCatchup performs one catch-up run per eligible task that missed scheduled
// runs during downtime. The catch-up run is recorded at `now` (so a subsequent
// restart does not re-trigger it) and honors the task's overlap policy via
// dispatch. Normal scheduling (computed in recompute) resumes afterward.
func (e *Engine) runCatchup(now time.Time) {
	e.mu.Lock()
	tasks := make([]taskCtx, 0, len(e.tasks))
	for _, tc := range e.tasks {
		tasks = append(tasks, tc)
	}
	e.mu.Unlock()

	for _, tc := range tasks {
		runs, err := e.store.ListRuns(tc.task.ID, 1)
		if err != nil || len(runs) == 0 {
			continue // never run → nothing to catch up
		}
		dec, err := catchup.Evaluate(tc.sch, tc.task.Timezone, runs[0].ScheduledFor, true, tc.task.CatchupPolicy, now)
		if err != nil {
			e.log.Error("engine: catchup evaluate", "task", tc.task.ID, "err", err)
			continue
		}
		if !dec.ShouldCatchUp {
			continue
		}
		e.log.Warn("missed run(s) during downtime; performing one catch-up",
			"task", tc.task.ID, "name", tc.task.Name, "first_missed", dec.FirstMissed)
		e.raiseAlert(tc.task.ID, domain.SeverityWarning, domain.AlertMissedRun,
			"missed run(s) during downtime; running one catch-up")
		e.dispatch(tc.task, now, domain.TriggerCatchup)
	}
}

// untilNext returns the duration until the earliest scheduled run, and whether
// any run is scheduled.
func (e *Engine) untilNext(now time.Time) (time.Duration, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	var earliest time.Time
	has := false
	for _, t := range e.next {
		if !has || t.Before(earliest) {
			earliest, has = t, true
		}
	}
	if !has {
		return 0, false
	}
	if d := earliest.Sub(now); d > 0 {
		return d, true
	}
	return 0, true
}

// runDue dispatches every task whose next run is at or before now and advances
// its schedule.
func (e *Engine) runDue(now time.Time) {
	e.mu.Lock()
	due := make([]string, 0)
	for id, t := range e.next {
		if !t.After(now) {
			due = append(due, id)
		}
	}
	e.mu.Unlock()

	for _, id := range due {
		e.mu.Lock()
		tc, ok := e.tasks[id]
		scheduledFor := e.next[id]
		e.mu.Unlock()
		if !ok {
			continue
		}

		e.dispatch(tc.task, scheduledFor, domain.TriggerSchedule)

		// Advance (or retire one-off) using the schedule.
		if tc.sch.Kind == domain.ScheduleOneOff {
			e.completeOneOff(id)
			continue
		}
		if n, ok, err := schedule.NextRun(tc.sch, tc.task.Timezone, scheduledFor); err == nil && ok {
			e.mu.Lock()
			e.next[id] = n
			e.mu.Unlock()
		} else {
			e.mu.Lock()
			delete(e.next, id)
			e.mu.Unlock()
		}
	}
}

// completeOneOff marks a one-off task completed and removes it from scheduling.
func (e *Engine) completeOneOff(taskID string) {
	e.mu.Lock()
	delete(e.next, taskID)
	delete(e.tasks, taskID)
	e.mu.Unlock()
	if err := e.store.SetTaskState(taskID, domain.TaskCompleted); err != nil {
		e.log.Error("engine: complete one-off", "task", taskID, "err", err)
	}
}

// launch runs a task through the worker pool and records the result.
func (e *Engine) launch(task domain.Task, scheduledFor time.Time, trigger domain.RunTrigger) {
	e.runWG.Add(1)
	go func() {
		defer e.runWG.Done()
		e.sem <- struct{}{}
		defer func() { <-e.sem }()

		run := e.runner.Run(e.runCtx, task, scheduledFor, trigger)
		e.recordRun(run)
		e.finish(task)
	}()
}

// recordRun persists a run, raises a failure alert when needed, and notifies
// the onRun callback.
func (e *Engine) recordRun(run domain.Run) {
	if err := e.store.CreateRun(&run); err != nil {
		e.log.Error("engine: record run", "task", run.TaskID, "err", err)
	}
	if run.Outcome == domain.OutcomeFailure {
		e.raiseAlert(run.TaskID, domain.SeverityError, domain.AlertRunFailed, "task run failed")
	}
	if e.onRun != nil {
		e.onRun(run)
	}
	// Fire event triggers on real completions (not queued/skipped markers).
	if e.onCompletion != nil && (run.Outcome == domain.OutcomeSuccess || run.Outcome == domain.OutcomeFailure) {
		e.onCompletion(run.TaskID, run.Outcome, run.ID, e.clk.Now())
	}
}

// finish marks a task no longer running and dispatches any queued pending run.
func (e *Engine) finish(task domain.Task) {
	e.mu.Lock()
	e.running[task.ID] = false
	qFor, queued := e.queued[task.ID]
	delete(e.queued, task.ID)
	e.mu.Unlock()

	if queued {
		e.mu.Lock()
		e.running[task.ID] = true
		e.mu.Unlock()
		e.launch(task, qFor, domain.TriggerSchedule)
	}
}

// raiseAlert stores an alert and logs it.
func (e *Engine) raiseAlert(taskID string, sev domain.AlertSeverity, kind domain.AlertKind, msg string) {
	a := domain.Alert{TaskID: taskID, Severity: sev, Kind: kind, Message: msg}
	if err := e.store.CreateAlert(&a); err != nil {
		e.log.Error("engine: create alert", "err", err)
	}
}
