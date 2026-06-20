package server

import (
	"net/http"
	"time"

	"github.com/shruggietech/go-scheduler/internal/domain"
	"github.com/shruggietech/go-scheduler/internal/schedule"
)

// Occurrence is a single calendar entry — either a past run or a computed future
// scheduled run.
type Occurrence struct {
	TaskID   string            `json:"task_id"`
	TaskName string            `json:"task_name"`
	Time     time.Time         `json:"time"`
	Kind     string            `json:"kind"` // "past" | "scheduled"
	Outcome  domain.RunOutcome `json:"outcome,omitempty"`
}

// CalendarResponse is returned by GET /v1/calendar.
type CalendarResponse struct {
	From        time.Time    `json:"from"`
	To          time.Time    `json:"to"`
	Occurrences []Occurrence `json:"occurrences"`
}

const maxOccurrencesPerTask = 500

func (s *Server) handleCalendar(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC()
	from := parseTimeParam(r.URL.Query().Get("from"), now)
	to := parseTimeParam(r.URL.Query().Get("to"), now.Add(7*24*time.Hour))
	if to.Before(from) {
		writeError(w, http.StatusBadRequest, CodeValidation, "to", "'to' must be after 'from'")
		return
	}

	tasks, err := s.store.ListTasks("", "")
	if err != nil {
		s.internal(w, err)
		return
	}

	occ := make([]Occurrence, 0)
	for _, task := range tasks {
		// Past runs in the window.
		runs, err := s.store.ListRuns(task.ID, 0)
		if err != nil {
			s.internal(w, err)
			return
		}
		for _, run := range runs {
			if !run.ScheduledFor.Before(from) && !run.ScheduledFor.After(to) {
				occ = append(occ, Occurrence{
					TaskID: task.ID, TaskName: task.Name, Time: run.ScheduledFor,
					Kind: "past", Outcome: run.Outcome,
				})
			}
		}

		// Future scheduled occurrences in the window (active tasks only).
		if task.State != domain.TaskActive || !task.Enabled {
			continue
		}
		sch, err := s.store.GetSchedule(task.ScheduleID)
		if err != nil {
			continue
		}
		start := from
		if now.After(start) {
			start = now
		}
		upcoming, err := schedule.UpcomingRuns(sch, task.Timezone, start, maxOccurrencesPerTask)
		if err != nil {
			continue
		}
		for _, t := range upcoming {
			if t.After(to) {
				break
			}
			occ = append(occ, Occurrence{TaskID: task.ID, TaskName: task.Name, Time: t, Kind: "scheduled"})
		}
	}

	writeJSON(w, http.StatusOK, CalendarResponse{From: from, To: to, Occurrences: occ})
}

func parseTimeParam(s string, def time.Time) time.Time {
	if s == "" {
		return def
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC()
	}
	return def
}
