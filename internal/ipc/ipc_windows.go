//go:build windows

package ipc

import (
	"context"
	"fmt"
	"net"

	"github.com/Microsoft/go-winio"

	"github.com/shruggietech/go-scheduler/internal/config"
)

func defaultEndpoint(_ config.Config) string {
	return `\\.\pipe\goschedd`
}

// Listen creates a Windows named-pipe listener.
func Listen(endpoint string) (net.Listener, error) {
	l, err := winio.ListenPipe(endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("ipc: listen pipe %s: %w", endpoint, err)
	}
	return l, nil
}

// DialContext connects to the named-pipe endpoint.
func DialContext(ctx context.Context, endpoint string) (net.Conn, error) {
	return winio.DialPipeContext(ctx, endpoint)
}
