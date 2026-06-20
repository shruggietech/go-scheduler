package clock

import (
	"sync"
	"time"
)

// FakeClock is a deterministic Clock for tests. Virtual time only moves when
// Advance or Set is called, so timer-driven code can be exercised without real
// sleeps. It is safe for concurrent use.
type FakeClock struct {
	mu     sync.Mutex
	now    time.Time
	wakers []*waker
}

type waker struct {
	deadline time.Time
	ch       chan time.Time
}

// NewFake returns a FakeClock positioned at t.
func NewFake(t time.Time) *FakeClock { return &FakeClock{now: t} }

// Now returns the current virtual time.
func (f *FakeClock) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.now
}

// After returns a channel that fires once virtual time reaches now+d.
func (f *FakeClock) After(d time.Duration) <-chan time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.addLocked(d).ch
}

// NewTimer returns a Timer that fires once virtual time reaches now+d.
func (f *FakeClock) NewTimer(d time.Duration) *Timer {
	f.mu.Lock()
	defer f.mu.Unlock()
	w := f.addLocked(d)
	return &Timer{
		C: w.ch,
		stop: func() bool {
			f.mu.Lock()
			defer f.mu.Unlock()
			return f.removeLocked(w)
		},
		reset: func(nd time.Duration) bool {
			f.mu.Lock()
			defer f.mu.Unlock()
			active := f.removeLocked(w)
			w.deadline = f.now.Add(nd)
			f.wakers = append(f.wakers, w)
			return active
		},
	}
}

// Sleep blocks until virtual time has advanced by at least d. It is intended to
// be called from a goroutine other than the one driving Advance.
func (f *FakeClock) Sleep(d time.Duration) { <-f.After(d) }

// Advance moves virtual time forward by d, firing any wakers that come due.
func (f *FakeClock) Advance(d time.Duration) {
	f.mu.Lock()
	f.now = f.now.Add(d)
	now := f.now
	due := make([]*waker, 0, len(f.wakers))
	kept := f.wakers[:0]
	for _, w := range f.wakers {
		if !w.deadline.After(now) {
			due = append(due, w)
		} else {
			kept = append(kept, w)
		}
	}
	f.wakers = kept
	f.mu.Unlock()

	// Fire outside the lock; channels are buffered (cap 1) so this never blocks.
	for _, w := range due {
		w.ch <- now
	}
}

// addLocked registers a waker for now+d. Must hold f.mu. A non-positive d fires
// immediately.
func (f *FakeClock) addLocked(d time.Duration) *waker {
	w := &waker{deadline: f.now.Add(d), ch: make(chan time.Time, 1)}
	if d <= 0 {
		w.ch <- f.now
		return w
	}
	f.wakers = append(f.wakers, w)
	return w
}

// removeLocked drops w from the pending set, returning true if it was present
// (i.e., had not yet fired). Must hold f.mu.
func (f *FakeClock) removeLocked(w *waker) bool {
	for i, x := range f.wakers {
		if x == w {
			f.wakers = append(f.wakers[:i], f.wakers[i+1:]...)
			return true
		}
	}
	return false
}
