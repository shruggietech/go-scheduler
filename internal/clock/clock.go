// Package clock provides an injectable time source. The project constitution
// makes this non-negotiable: engine code MUST read time through a Clock so that
// scheduling, catch-up, and DST behavior can be tested deterministically without
// depending on the real wall clock.
package clock

import "time"

// Clock is the minimal time abstraction used throughout the scheduler.
type Clock interface {
	// Now returns the current time.
	Now() time.Time
	// After returns a channel that delivers the current time after at least d.
	After(d time.Duration) <-chan time.Time
	// NewTimer creates a Timer that fires after at least d.
	NewTimer(d time.Duration) *Timer
	// Sleep blocks for at least d.
	Sleep(d time.Duration)
}

// Timer is a clock-agnostic one-shot timer. Its C channel receives the time at
// which the timer fired.
type Timer struct {
	C     <-chan time.Time
	stop  func() bool
	reset func(d time.Duration) bool
}

// Stop prevents the Timer from firing. It returns true if it stops the timer,
// false if the timer has already fired or been stopped.
func (t *Timer) Stop() bool { return t.stop() }

// Reset changes the timer to expire after d. It returns true if the timer was
// active.
func (t *Timer) Reset(d time.Duration) bool { return t.reset(d) }
