//go:build !windows

package ipc

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/shruggietech/go-scheduler/internal/config"
)

func defaultEndpoint(cfg config.Config) string {
	return filepath.Join(cfg.DataDir, "goschedd.sock")
}

// Listen creates a Unix domain socket listener, removing any stale socket file
// left by a previous run.
func Listen(endpoint string) (net.Listener, error) {
	if err := os.MkdirAll(filepath.Dir(endpoint), 0o755); err != nil {
		return nil, fmt.Errorf("ipc: create socket dir: %w", err)
	}
	if err := os.Remove(endpoint); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("ipc: remove stale socket: %w", err)
	}
	l, err := net.Listen("unix", endpoint)
	if err != nil {
		return nil, fmt.Errorf("ipc: listen %s: %w", endpoint, err)
	}
	return l, nil
}

// DialContext connects to the Unix domain socket endpoint.
func DialContext(ctx context.Context, endpoint string) (net.Conn, error) {
	var d net.Dialer
	return d.DialContext(ctx, "unix", endpoint)
}
