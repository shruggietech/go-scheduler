//go:build !windows

package platform

import (
	"os/exec"
	"runtime"
	"syscall"
)

func dataDir() string {
	if runtime.GOOS == "darwin" {
		return "/Library/Application Support/goscheduler"
	}
	return "/var/lib/goscheduler"
}

// hideConsole is a no-op on Unix: child processes do not create console windows.
func hideConsole(_ *exec.Cmd) {}

// detachProcess starts the child in a new session so it is not tied to the
// launcher's controlling terminal and survives the launcher exiting.
func detachProcess(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setsid = true
}
