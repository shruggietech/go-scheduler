// Package platform isolates OS-specific behavior behind a small, build-tagged API:
// the system-wide data directory and the "spawn a child process with no visible
// console window" behavior required by the project (the GUI must never leave a
// visible command prompt, and scheduled tasks must not flash console windows).
package platform

import "os/exec"

// DataDir returns the default system-wide directory where the daemon stores its
// database, socket/pipe, and logs. It can be overridden by configuration.
func DataDir() string { return dataDir() }

// HideConsole configures cmd so the spawned process does not open a visible
// console window. It is a no-op on platforms where this is not applicable.
func HideConsole(cmd *exec.Cmd) { hideConsole(cmd) }

// Detach configures cmd so the spawned process keeps running independently of
// the launching process (so a background daemon survives the GUI that started
// it). It is a no-op where not applicable.
func Detach(cmd *exec.Cmd) { detachProcess(cmd) }
