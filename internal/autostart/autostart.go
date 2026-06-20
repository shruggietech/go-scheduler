// Package autostart lets the GUI (or any client) ensure the daemon is running:
// if it cannot be reached, it launches goschedd as a detached, windowless
// background process and waits until it becomes reachable. This gives the GUI a
// zero-configuration experience — opening it just works, without first
// installing a service. A previously running daemon (e.g. the installed service)
// is detected via the health check and reused; the single-instance lock in the
// daemon prevents a second one from starting.
package autostart

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/shruggietech/go-scheduler/internal/platform"
)

// pollInterval is how often EnsureRunning re-checks readiness after spawning.
var pollInterval = 200 * time.Millisecond

// DaemonPath returns the goschedd binary path, expected next to the current
// executable.
func DaemonPath() (string, error) {
	self, err := os.Executable()
	if err != nil {
		return "", err
	}
	name := "goschedd"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	p := filepath.Join(filepath.Dir(self), name)
	if _, err := os.Stat(p); err != nil {
		return "", fmt.Errorf("autostart: daemon binary not found next to the GUI (looked for %s)", p)
	}
	return p, nil
}

// SpawnDaemon launches goschedd (located next to the current executable) as a
// detached, windowless background process that outlives the caller.
func SpawnDaemon() error {
	execPath, err := DaemonPath()
	if err != nil {
		return err
	}
	cmd := exec.Command(execPath)
	platform.HideConsole(cmd) // no console window
	platform.Detach(cmd)      // survives the GUI exiting
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("autostart: start daemon: %w", err)
	}
	return cmd.Process.Release()
}

// EnsureRunning makes the daemon reachable. If ping succeeds it returns
// immediately (a daemon — possibly the installed service — is already running).
// Otherwise it calls spawn and polls ping until the daemon is ready or ctx is
// done.
func EnsureRunning(ctx context.Context, ping func(context.Context) error, spawn func() error) error {
	if ping(ctx) == nil {
		return nil
	}
	if err := spawn(); err != nil {
		return err
	}
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("autostart: daemon did not become ready in time: %w", ctx.Err())
		case <-ticker.C:
			if ping(ctx) == nil {
				return nil
			}
		}
	}
}
