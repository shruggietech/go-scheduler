package schedule

import (
	"strings"
	"testing"
	"time"
)

var now = time.Date(2026, 6, 19, 8, 0, 0, 0, time.UTC)

func TestParse_Forms(t *testing.T) {
	tests := []struct {
		input       string
		wantTokens  []string // every token must appear in the RRULE
		wantSummary string
	}{
		{"every 15 minutes", []string{"FREQ=MINUTELY", "INTERVAL=15"}, "Every 15 minutes"},
		{"every 30s", []string{"FREQ=SECONDLY", "INTERVAL=30"}, "Every 30 seconds"},
		{"every 2 hours", []string{"FREQ=HOURLY", "INTERVAL=2"}, "Every 2 hours"},
		{"every day at 09:00", []string{"FREQ=DAILY", "INTERVAL=1", "BYHOUR=9", "BYMINUTE=0"}, "Every day at 09:00"},
		{"every 3 days", []string{"FREQ=DAILY", "INTERVAL=3"}, "Every 3 days"},
		{"weekdays at 9:00 AM", []string{"FREQ=WEEKLY", "BYDAY=MO,TU,WE,TH,FR", "BYHOUR=9"}, "Every weekday at 09:00"},
		{"every monday at 9am", []string{"FREQ=WEEKLY", "BYDAY=MO", "BYHOUR=9"}, "Every Monday at 09:00"},
		{"3rd wednesday monthly at 14:00", []string{"FREQ=MONTHLY", "BYDAY=+3WE", "BYHOUR=14"}, "The 3rd Wednesday of every month at 14:00"},
		{"last friday of the month", []string{"FREQ=MONTHLY", "BYDAY=-1FR"}, "The last Friday of every month"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			sch, err := Parse(tt.input, "UTC", now)
			if err != nil {
				t.Fatalf("Parse(%q) error: %v", tt.input, err)
			}
			for _, want := range tt.wantTokens {
				if !strings.Contains(sch.RRULE, want) {
					t.Fatalf("RRULE %q missing token %q", sch.RRULE, want)
				}
			}
			if sch.HumanSummary != tt.wantSummary {
				t.Fatalf("summary = %q, want %q", sch.HumanSummary, tt.wantSummary)
			}
		})
	}
}

func TestParse_Rejects(t *testing.T) {
	for _, bad := range []string{"", "soon", "every banana", "every 15 minutes at 09:00", "3rd wednesday monthly at 99:99"} {
		if _, err := Parse(bad, "UTC", now); err == nil {
			t.Fatalf("expected error for %q", bad)
		}
	}
}

func TestParse_TimeOfDayVariants(t *testing.T) {
	for _, in := range []string{"every day at 14:00", "every day at 2:00 PM", "every day at 2pm"} {
		sch, err := Parse(in, "UTC", now)
		if err != nil {
			t.Fatalf("Parse(%q): %v", in, err)
		}
		if !strings.Contains(sch.RRULE, "BYHOUR=14") {
			t.Fatalf("%q did not yield 14:00, got %q", in, sch.RRULE)
		}
	}
}
