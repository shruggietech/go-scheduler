// Package catchup decides whether a task missed scheduled runs during downtime
// and should perform a single catch-up run. The decision is pure (given the
// task's schedule, last run, and policy); the engine performs the dispatch at
// startup. Per the spec, at most one catch-up run occurs per task regardless of
// how many occurrences were missed, after which normal scheduling resumes.
package catchup

import (
	"time"

	"github.com/shruggietech/go-scheduler/internal/domain"
	"github.com/shruggietech/go-scheduler/internal/schedule"
)

// Decision is the outcome of evaluating catch-up for a task.
type Decision struct {
	ShouldCatchUp bool
	// FirstMissed is the first scheduled occurrence that was missed (for the
	// alert message); the catch-up run itself executes at startup time.
	FirstMissed time.Time
}

// Evaluate reports whether a task should perform a catch-up run. It returns
// no-catch-up when the policy is not "one", when the task has no prior run
// (nothing to be "behind" on), or when the next scheduled occurrence after the
// last run is still in the future.
func Evaluate(sch domain.Schedule, tzName string, lastScheduled time.Time, hasPrior bool, policy domain.CatchupPolicy, now time.Time) (Decision, error) {
	if policy != domain.CatchupOne || !hasPrior {
		return Decision{}, nil
	}
	next, ok, err := schedule.NextRun(sch, tzName, lastScheduled)
	if err != nil {
		return Decision{}, err
	}
	if ok && !next.After(now) {
		// The occurrence after the last run is at or before now → it was missed.
		return Decision{ShouldCatchUp: true, FirstMissed: next}, nil
	}
	return Decision{}, nil
}
