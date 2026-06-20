//go:build windows

package platform

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// createNoWindow is the Windows CREATE_NO_WINDOW process-creation flag. It
// prevents the child process from getting its own console window.
const createNoWindow = 0x08000000

func dataDir() string {
	base := os.Getenv("ProgramData")
	if base == "" {
		base = `C:\ProgramData`
	}
	return filepath.Join(base, "goscheduler")
}

func hideConsole(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.HideWindow = true
	cmd.SysProcAttr.CreationFlags |= createNoWindow
}
