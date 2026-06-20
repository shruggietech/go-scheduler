package clock

import "time"

// realClock is the production Clock backed by the standard library.
type realClock struct{}

// NewReal returns a Clock backed by the real wall clock.
func NewReal() Clock { return realClock{} }

func (realClock) Now() time.Time { return time.Now() }

func (realClock) After(d time.Duration) <-chan time.Time { return time.After(d) }

func (realClock) NewTimer(d time.Duration) *Timer {
	t := time.NewTimer(d)
	return &Timer{C: t.C, stop: t.Stop, reset: t.Reset}
}

func (realClock) Sleep(d time.Duration) { time.Sleep(d) }
