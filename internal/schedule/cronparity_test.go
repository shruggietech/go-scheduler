package schedule

import (
	"testing"
	"time"
)

// TestCronParity demonstrates SC-002: anything a typical cron expression can say
// is expressible here in human-readable terms, producing the same run times.
// Each case pairs a familiar cron pattern with the human phrase users would type
// and asserts the computed next run matches the cron-equivalent instant.
func TestCronParity(t *testing.T) {
	utcAnchor := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		cron    string // for documentation/traceability
		human   string
		after   time.Time
		wantUTC time.Time
	}{
		{
			cron:    "*/15 * * * *",
			human:   "every 15 minutes",
			after:   time.Date(2026, 6, 1, 0, 7, 0, 0, time.UTC),
			wantUTC: time.Date(2026, 6, 1, 0, 15, 0, 0, time.UTC),
		},
		{
			cron:    "0 9 * * *",
			human:   "every day at 09:00",
			after:   time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
			wantUTC: time.Date(2026, 6, 2, 9, 0, 0, 0, time.UTC),
		},
		{
			cron:    "0 9 * * 1-5", // weekdays at 09:00; June 1 2026 is a Monday
			human:   "weekdays at 09:00",
			after:   time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
			wantUTC: time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC),
		},
		{
			cron:    "0 14 * * 3", // every Wednesday at 14:00; first after June 1 is June 3
			human:   "every wednesday at 14:00",
			after:   time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
			wantUTC: time.Date(2026, 6, 3, 14, 0, 0, 0, time.UTC),
		},
		{
			cron:    "0 0 * * *",
			human:   "every day at 00:00",
			after:   time.Date(2026, 6, 1, 1, 0, 0, 0, time.UTC),
			wantUTC: time.Date(2026, 6, 2, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, c := range cases {
		t.Run(c.cron, func(t *testing.T) {
			sch, err := Parse(c.human, "UTC", utcAnchor)
			if err != nil {
				t.Fatalf("Parse(%q): %v", c.human, err)
			}
			got, ok, err := NextRun(sch, "UTC", c.after)
			if err != nil || !ok {
				t.Fatalf("NextRun ok=%v err=%v", ok, err)
			}
			if !got.Equal(c.wantUTC) {
				t.Fatalf("cron %q via %q: got %v, want %v", c.cron, c.human, got, c.wantUTC)
			}
		})
	}
}
