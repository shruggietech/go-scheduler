package clock

import (
	"testing"
	"time"
)

var base = time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)

func TestFakeClock_NowAndAdvance(t *testing.T) {
	c := NewFake(base)
	if !c.Now().Equal(base) {
		t.Fatalf("Now() = %v, want %v", c.Now(), base)
	}
	c.Advance(90 * time.Minute)
	if got, want := c.Now(), base.Add(90*time.Minute); !got.Equal(want) {
		t.Fatalf("after Advance Now() = %v, want %v", got, want)
	}
}

func TestFakeClock_After_FiresOnAdvance(t *testing.T) {
	c := NewFake(base)
	ch := c.After(time.Hour)
	select {
	case <-ch:
		t.Fatal("After fired before time advanced")
	default:
	}
	c.Advance(time.Hour)
	select {
	case got := <-ch:
		if want := base.Add(time.Hour); !got.Equal(want) {
			t.Fatalf("After delivered %v, want %v", got, want)
		}
	default:
		t.Fatal("After did not fire after advancing past deadline")
	}
}

func TestFakeClock_After_NonPositiveFiresImmediately(t *testing.T) {
	c := NewFake(base)
	select {
	case <-c.After(0):
	default:
		t.Fatal("After(0) should fire immediately")
	}
}

func TestFakeClock_Timer_StopPreventsFire(t *testing.T) {
	c := NewFake(base)
	tm := c.NewTimer(time.Hour)
	if !tm.Stop() {
		t.Fatal("Stop on active timer should return true")
	}
	c.Advance(2 * time.Hour)
	select {
	case <-tm.C:
		t.Fatal("stopped timer should not fire")
	default:
	}
	if tm.Stop() {
		t.Fatal("Stop on already-stopped timer should return false")
	}
}

func TestFakeClock_Timer_Reset(t *testing.T) {
	c := NewFake(base)
	tm := c.NewTimer(time.Hour)
	c.Advance(30 * time.Minute)
	if !tm.Reset(2 * time.Hour) { // was still active
		t.Fatal("Reset on active timer should return true")
	}
	c.Advance(time.Hour) // now at +90m; new deadline is +30m+2h = +150m, not yet
	select {
	case <-tm.C:
		t.Fatal("timer fired before reset deadline")
	default:
	}
	c.Advance(90 * time.Minute) // now at +180m, past +150m
	select {
	case <-tm.C:
	default:
		t.Fatal("timer did not fire after reset deadline")
	}
}

func TestFakeClock_Sleep_UnblocksWhenAdvanced(t *testing.T) {
	c := NewFake(base)
	done := make(chan struct{})
	go func() {
		c.Sleep(time.Hour)
		close(done)
	}()
	// Give the sleeper time to register its waker, then advance.
	waitForWakers(c, 1)
	c.Advance(time.Hour)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Sleep did not unblock after advancing")
	}
}

// waitForWakers spins until n wakers are registered (real-time bounded), so the
// Sleep goroutine has registered before we advance.
func waitForWakers(c *FakeClock, n int) {
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		c.mu.Lock()
		got := len(c.wakers)
		c.mu.Unlock()
		if got >= n {
			return
		}
		time.Sleep(time.Millisecond)
	}
}
