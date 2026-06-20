// Package schedule converts human-readable scheduling intent into a stored
// representation (RFC 5545 RRULE for recurrence, or a single instant for
// one-off) and computes concrete next-run times in UTC. Cron-style syntax is
// never exposed to users; RRULE is an internal detail.
package schedule

import (
	"fmt"
	"time"

	"github.com/teambition/rrule-go"

	"github.com/shruggietech/go-scheduler/internal/domain"
	"github.com/shruggietech/go-scheduler/internal/timezone"
)

// NewOneOff builds a one-off schedule that fires once at runAt (stored UTC).
func NewOneOff(runAt time.Time) domain.Schedule {
	u := runAt.UTC()
	return domain.Schedule{
		Kind:         domain.ScheduleOneOff,
		RunAt:        &u,
		HumanSummary: "Once at " + u.Format("2006-01-02 15:04 MST"),
	}
}

// NextRun returns the next run instant (UTC) strictly after `after` for the
// schedule evaluated in timezone tzName. The bool is false when there is no
// further run (exhausted one-off, or event schedule with no time component).
func NextRun(sch domain.Schedule, tzName string, after time.Time) (time.Time, bool, error) {
	switch sch.Kind {
	case domain.ScheduleOneOff:
		if sch.RunAt == nil {
			return time.Time{}, false, fmt.Errorf("schedule: one-off missing run_at")
		}
		if sch.RunAt.After(after) {
			return sch.RunAt.UTC(), true, nil
		}
		return time.Time{}, false, nil
	case domain.ScheduleEvent:
		return time.Time{}, false, nil
	case domain.ScheduleRecurring:
		return nextRecurring(sch, tzName, after)
	default:
		return time.Time{}, false, fmt.Errorf("schedule: unknown kind %q", sch.Kind)
	}
}

func nextRecurring(sch domain.Schedule, tzName string, after time.Time) (time.Time, bool, error) {
	loc, err := timezone.Resolve(tzName)
	if err != nil {
		return time.Time{}, false, err
	}
	opt, err := rrule.StrToROption(sch.RRULE)
	if err != nil {
		return time.Time{}, false, fmt.Errorf("schedule: parse rrule %q: %w", sch.RRULE, err)
	}
	anchor := after
	if sch.Anchor != nil {
		anchor = *sch.Anchor
	}
	opt.Dtstart = anchor.In(loc)

	r, err := rrule.NewRRule(*opt)
	if err != nil {
		return time.Time{}, false, fmt.Errorf("schedule: build rrule: %w", err)
	}
	occ := r.After(after.In(loc), false)
	if occ.IsZero() {
		return time.Time{}, false, nil
	}

	// For day-or-coarser frequencies at a fixed time-of-day, apply the DST rules
	// (next-valid / first-occurrence). Sub-daily frequencies use the raw instant.
	switch opt.Freq {
	case rrule.YEARLY, rrule.MONTHLY, rrule.WEEKLY, rrule.DAILY:
		norm := timezone.WallTime(loc, occ.Year(), occ.Month(), occ.Day(), occ.Hour(), occ.Minute(), occ.Second())
		return norm.UTC(), true, nil
	default:
		return occ.UTC(), true, nil
	}
}

// UpcomingRuns returns up to n future run instants (UTC) after `after`.
func UpcomingRuns(sch domain.Schedule, tzName string, after time.Time, n int) ([]time.Time, error) {
	var out []time.Time
	cursor := after
	for i := 0; i < n; i++ {
		next, ok, err := NextRun(sch, tzName, cursor)
		if err != nil {
			return nil, err
		}
		if !ok {
			break
		}
		out = append(out, next)
		cursor = next
	}
	return out, nil
}
