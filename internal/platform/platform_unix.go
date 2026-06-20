//go:build !windows

package platform

import (
	"os/exec"
	"runtime"
)

func dataDir() string {
	if runtime.GOOS == "darwin" {
		return "/Library/Application Support/goscheduler"
	}
	return "/var/lib/goscheduler"
}

// hideConsole is a no-op on Unix: child processes do not create console windows.
func hideConsole(_ *exec.Cmd) {}
