package autostart

import (
	"context"
	"errors"
	"testing"
	"time"
)

func init() { pollInterval = 5 * time.Millisecond } // speed up tests

func TestEnsureRunning_AlreadyReachable(t *testing.T) {
	spawned := 0
	err := EnsureRunning(context.Background(),
		func(context.Context) error { return nil },
		func() error { spawned++; return nil },
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if spawned != 0 {
		t.Fatal("should not spawn when the daemon is already reachable")
	}
}

func TestEnsureRunning_SpawnsThenBecomesReady(t *testing.T) {
	pings := 0
	ping := func(context.Context) error {
		pings++
		if pings <= 2 { // initial check + first poll fail, then ready
			return errors.New("down")
		}
		return nil
	}
	spawned := 0
	err := EnsureRunning(context.Background(), ping, func() error { spawned++; return nil })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if spawned != 1 {
		t.Fatalf("expected exactly one spawn, got %d", spawned)
	}
}

func TestEnsureRunning_SpawnError(t *testing.T) {
	want := errors.New("no binary")
	err := EnsureRunning(context.Background(),
		func(context.Context) error { return errors.New("down") },
		func() error { return want },
	)
	if !errors.Is(err, want) {
		t.Fatalf("expected spawn error, got %v", err)
	}
}

func TestEnsureRunning_Timeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	err := EnsureRunning(ctx,
		func(context.Context) error { return errors.New("always down") },
		func() error { return nil },
	)
	if err == nil {
		t.Fatal("expected a timeout error when the daemon never becomes ready")
	}
}
