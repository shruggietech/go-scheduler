//go:build windows

package ipc

import (
	"context"
	"fmt"
	"net"

	"github.com/Microsoft/go-winio"

	"github.com/shruggietech/go-schedule/internal/config"
)

func defaultEndpoint(_ config.Config) string {
	return `\\.\pipe\goschedd`
}

// pipeSDDL is the security descriptor applied to the IPC named pipe. When the
// daemon runs as a Windows service (LocalSystem), the pipe's default ACL would
// only admit SYSTEM/Administrators, so a normal per-user GUI is denied. This
// SDDL grants:
//   - SYSTEM (SY) and built-in Administrators (BA): full control;
//   - Authenticated Users (AU): generic read/write, so the logged-in user's GUI
//     and CLI can connect even when launched without elevation (a non-elevated
//     admin token has its Administrators SID as deny-only, so AU is required).
//
// Trade-off: any authenticated local user can manage the scheduler. That suits a
// single-user desktop install; a multi-user/locked-down deployment should narrow
// AU to a dedicated admin group (see config.AdminGroup) in a future change.
const pipeSDDL = "D:P(A;;GA;;;SY)(A;;GA;;;BA)(A;;GRGW;;;AU)"

// Listen creates a Windows named-pipe listener with an ACL that lets the
// interactive user reach a service-hosted (LocalSystem) daemon.
func Listen(endpoint string) (net.Listener, error) {
	l, err := winio.ListenPipe(endpoint, &winio.PipeConfig{SecurityDescriptor: pipeSDDL})
	if err != nil {
		return nil, fmt.Errorf("ipc: listen pipe %s: %w", endpoint, err)
	}
	return l, nil
}

// DialContext connects to the named-pipe endpoint.
func DialContext(ctx context.Context, endpoint string) (net.Conn, error) {
	return winio.DialPipeContext(ctx, endpoint)
}
