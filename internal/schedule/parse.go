package schedule

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/teambition/rrule-go"

	"github.com/shruggietech/go-scheduler/internal/domain"
)

// Parse turns a human-readable schedule phrase into a recurring Schedule with an
// RRULE, anchor, and plain-language summary. It never requires cron syntax.
//
// Supported forms (case-insensitive):
//
//	every <N> <unit>            e.g. "every 15 minutes", "every 30s", "every 2 hours"
//	every <unit>               e.g. "every day", "every week"
//	... [at <time>]            day-or-coarser rules accept a time-of-day
//	weekdays|weekends [at ...]  e.g. "weekdays at 09:00"
//	every <weekday> [at ...]    e.g. "every monday at 9am"
//	<ordinal> <weekday> monthly e.g. "3rd wednesday monthly at 14:00", "last friday of the month"
func Parse(input, tzName string, now time.Time) (domain.Schedule, error) {
	s := strings.ToLower(strings.TrimSpace(input))
	if s == "" {
		return domain.Schedule{}, fmt.Errorf("schedule: empty schedule expression")
	}

	if sch, ok, err := parseOrdinal(s); ok || err != nil {
		return finish(sch, tzName, now, err)
	}
	if sch, ok, err := parseDayset(s); ok || err != nil {
		return finish(sch, tzName, now, err)
	}
	if sch, ok, err := parseEveryWeekday(s); ok || err != nil {
		return finish(sch, tzName, now, err)
	}
	if sch, ok, err := parseInterval(s); ok || err != nil {
		return finish(sch, tzName, now, err)
	}
	return domain.Schedule{}, fmt.Errorf("schedule: could not understand %q (try forms like \"every 15 minutes\", \"weekdays at 09:00\", \"3rd wednesday monthly at 14:00\")", input)
}

// finish validates the constructed RRULE, sets anchor/kind, and returns.
func finish(sch domain.Schedule, _ string, now time.Time, err error) (domain.Schedule, error) {
	if err != nil {
		return domain.Schedule{}, err
	}
	if _, perr := rrule.StrToROption(sch.RRULE); perr != nil {
		return domain.Schedule{}, fmt.Errorf("schedule: built invalid rule %q: %w", sch.RRULE, perr)
	}
	sch.Kind = domain.ScheduleRecurring
	anchor := now.UTC()
	sch.Anchor = &anchor
	return sch, nil
}

var (
	reInterval = regexp.MustCompile(`^every\s+(?:(\d+)\s*)?(second|seconds|sec|secs|s|minute|minutes|min|mins|m|hour|hours|hr|hrs|h|day|days|d|week|weeks|w)(?:\s+at\s+(.+))?$`)
	reDayset   = regexp.MustCompile(`^(weekdays|weekends)(?:\s+at\s+(.+))?$`)
	reEveryDay = regexp.MustCompile(`^every\s+(monday|tuesday|wednesday|thursday|friday|saturday|sunday)(?:\s+at\s+(.+))?$`)
	reOrdinal  = regexp.MustCompile(`^(1st|2nd|3rd|4th|5th|last|first|second|third|fourth|fifth)\s+(monday|tuesday|wednesday|thursday|friday|saturday|sunday)\s+(?:of\s+(?:the|each|every)\s+month|monthly)(?:\s+at\s+(.+))?$`)
)

var weekdayCode = map[string]string{
	"monday": "MO", "tuesday": "TU", "wednesday": "WE", "thursday": "TH",
	"friday": "FR", "saturday": "SA", "sunday": "SU",
}

var weekdayTitle = map[string]string{
	"monday": "Monday", "tuesday": "Tuesday", "wednesday": "Wednesday", "thursday": "Thursday",
	"friday": "Friday", "saturday": "Saturday", "sunday": "Sunday",
}

var ordinalNum = map[string]int{
	"1st": 1, "first": 1, "2nd": 2, "second": 2, "3rd": 3, "third": 3,
	"4th": 4, "fourth": 4, "5th": 5, "fifth": 5, "last": -1,
}

func parseInterval(s string) (domain.Schedule, bool, error) {
	m := reInterval.FindStringSubmatch(s)
	if m == nil {
		return domain.Schedule{}, false, nil
	}
	n := 1
	if m[1] != "" {
		var err error
		if n, err = strconv.Atoi(m[1]); err != nil || n < 1 {
			return domain.Schedule{}, true, fmt.Errorf("schedule: invalid interval %q", m[1])
		}
	}
	freq, unitName, subDaily := unitToFreq(m[2])
	tod := strings.TrimSpace(m[3])
	if subDaily && tod != "" {
		return domain.Schedule{}, true, fmt.Errorf("schedule: %q does not support an 'at <time>' clause", m[2])
	}

	parts := []string{"FREQ=" + freq, "INTERVAL=" + strconv.Itoa(n)}
	summary := "Every " + plural(n, unitName)
	if !subDaily {
		h, mi, withTime, err := maybeTime(tod)
		if err != nil {
			return domain.Schedule{}, true, err
		}
		if withTime {
			parts = append(parts, byTime(h, mi)...)
			summary += " at " + clock(h, mi)
		}
	}
	return domain.Schedule{RRULE: strings.Join(parts, ";"), HumanSummary: summary}, true, nil
}

func parseDayset(s string) (domain.Schedule, bool, error) {
	m := reDayset.FindStringSubmatch(s)
	if m == nil {
		return domain.Schedule{}, false, nil
	}
	var byday, label string
	if m[1] == "weekdays" {
		byday, label = "MO,TU,WE,TH,FR", "Every weekday"
	} else {
		byday, label = "SA,SU", "Every weekend day"
	}
	parts := []string{"FREQ=WEEKLY", "BYDAY=" + byday}
	h, mi, withTime, err := maybeTime(strings.TrimSpace(m[2]))
	if err != nil {
		return domain.Schedule{}, true, err
	}
	if withTime {
		parts = append(parts, byTime(h, mi)...)
		label += " at " + clock(h, mi)
	}
	return domain.Schedule{RRULE: strings.Join(parts, ";"), HumanSummary: label}, true, nil
}

func parseEveryWeekday(s string) (domain.Schedule, bool, error) {
	m := reEveryDay.FindStringSubmatch(s)
	if m == nil {
		return domain.Schedule{}, false, nil
	}
	parts := []string{"FREQ=WEEKLY", "BYDAY=" + weekdayCode[m[1]]}
	label := "Every " + weekdayTitle[m[1]]
	h, mi, withTime, err := maybeTime(strings.TrimSpace(m[2]))
	if err != nil {
		return domain.Schedule{}, true, err
	}
	if withTime {
		parts = append(parts, byTime(h, mi)...)
		label += " at " + clock(h, mi)
	}
	return domain.Schedule{RRULE: strings.Join(parts, ";"), HumanSummary: label}, true, nil
}

func parseOrdinal(s string) (domain.Schedule, bool, error) {
	m := reOrdinal.FindStringSubmatch(s)
	if m == nil {
		return domain.Schedule{}, false, nil
	}
	n := ordinalNum[m[1]]
	day := weekdayCode[m[2]]
	sign := "+"
	if n < 0 {
		sign = ""
	}
	parts := []string{"FREQ=MONTHLY", fmt.Sprintf("BYDAY=%s%d%s", sign, n, day)}
	label := fmt.Sprintf("The %s %s of every month", ordinalWord(n), weekdayTitle[m[2]])
	h, mi, withTime, err := maybeTime(strings.TrimSpace(m[3]))
	if err != nil {
		return domain.Schedule{}, true, err
	}
	if withTime {
		parts = append(parts, byTime(h, mi)...)
		label += " at " + clock(h, mi)
	}
	return domain.Schedule{RRULE: strings.Join(parts, ";"), HumanSummary: label}, true, nil
}

// ---- helpers ------------------------------------------------------------

func unitToFreq(u string) (freq, name string, subDaily bool) {
	switch u {
	case "second", "seconds", "sec", "secs", "s":
		return "SECONDLY", "second", true
	case "minute", "minutes", "min", "mins", "m":
		return "MINUTELY", "minute", true
	case "hour", "hours", "hr", "hrs", "h":
		return "HOURLY", "hour", true
	case "day", "days", "d":
		return "DAILY", "day", false
	case "week", "weeks", "w":
		return "WEEKLY", "week", false
	}
	return "", "", false
}

func byTime(h, mi int) []string {
	return []string{"BYHOUR=" + strconv.Itoa(h), "BYMINUTE=" + strconv.Itoa(mi), "BYSECOND=0"}
}

// maybeTime parses an optional time-of-day clause. Returns withTime=false when
// the clause is empty.
func maybeTime(s string) (h, mi int, withTime bool, err error) {
	if s == "" {
		return 0, 0, false, nil
	}
	h, mi, ok := parseTimeOfDay(s)
	if !ok {
		return 0, 0, false, fmt.Errorf("schedule: invalid time-of-day %q (try 09:00, 9:00 AM, 9am)", s)
	}
	return h, mi, true, nil
}

var reTOD = regexp.MustCompile(`^(\d{1,2})(?::(\d{2}))?\s*(am|pm)?$`)

// parseTimeOfDay accepts "14:00", "9:00", "9:00 am", "9am", "9".
func parseTimeOfDay(s string) (h, mi int, ok bool) {
	m := reTOD.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return 0, 0, false
	}
	h, _ = strconv.Atoi(m[1])
	if m[2] != "" {
		mi, _ = strconv.Atoi(m[2])
	}
	switch m[3] {
	case "am":
		if h == 12 {
			h = 0
		}
	case "pm":
		if h != 12 {
			h += 12
		}
	}
	if h > 23 || mi > 59 {
		return 0, 0, false
	}
	return h, mi, true
}

func plural(n int, unit string) string {
	if n == 1 {
		return unit
	}
	return strconv.Itoa(n) + " " + unit + "s"
}

func clock(h, mi int) string { return fmt.Sprintf("%02d:%02d", h, mi) }

func ordinalWord(n int) string {
	switch n {
	case 1:
		return "1st"
	case 2:
		return "2nd"
	case 3:
		return "3rd"
	case 4:
		return "4th"
	case 5:
		return "5th"
	case -1:
		return "last"
	}
	return strconv.Itoa(n) + "th"
}
