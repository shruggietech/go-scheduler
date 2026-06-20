// Package timezone resolves IANA timezones and converts intended local
// wall-clock times into concrete instants, applying the project's Daylight
// Saving Time rules: a time that falls in a skipped hour (spring-forward) runs
// at the next valid instant; a time in a repeated hour (fall-back) runs once,
// on the first occurrence. Storage and scheduling use UTC throughout.
package timezone

import (
	"fmt"
	"time"
)

// Resolve returns the *time.Location for an IANA name. "Local" and "" map to the
// host's local zone.
func Resolve(name string) (*time.Location, error) {
	if name == "" || name == "Local" {
		return time.Local, nil
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return nil, fmt.Errorf("timezone: %q is not a valid IANA zone: %w", name, err)
	}
	return loc, nil
}

// WallTime resolves an intended local wall-clock time (y/mo/d h:mi:s in loc) to
// a concrete instant located in loc, applying the DST rules described in the
// package doc. Callers convert the result to UTC for storage/dispatch.
func WallTime(loc *time.Location, y int, mo time.Month, d, h, mi, s int) time.Time {
	// Spring-forward: if the requested wall time does not exist, advance minute
	// by minute (using UTC calendar arithmetic to handle rollovers) until we
	// reach the first valid wall time — the next valid instant.
	base := time.Date(y, mo, d, h, mi, s, 0, time.UTC)
	for add := 0; add < 24*60; add++ {
		ref := base.Add(time.Duration(add) * time.Minute)
		cand := time.Date(ref.Year(), ref.Month(), ref.Day(), ref.Hour(), ref.Minute(), ref.Second(), 0, loc)
		if wallMatches(cand, ref) {
			return firstOccurrence(loc, cand)
		}
	}
	return firstOccurrence(loc, time.Date(y, mo, d, h, mi, s, 0, loc))
}

// firstOccurrence ensures that for an ambiguous wall time (fall-back), we return
// the earlier of the two instants. If shifting one hour earlier yields the same
// wall-clock reading, that earlier instant is the first occurrence.
func firstOccurrence(loc *time.Location, t time.Time) time.Time {
	earlier := t.Add(-time.Hour)
	if earlier.In(loc).Hour() == t.Hour() && earlier.In(loc).Minute() == t.Minute() {
		return earlier
	}
	return t
}

func wallMatches(cand, ref time.Time) bool {
	return cand.Year() == ref.Year() && cand.Month() == ref.Month() && cand.Day() == ref.Day() &&
		cand.Hour() == ref.Hour() && cand.Minute() == ref.Minute()
}
