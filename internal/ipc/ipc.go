// Package ipc provides the local transport the daemon and its clients use: a
// Unix domain socket on Unix and a named pipe on Windows. HTTP/JSON is served
// over this transport, so neither a TCP port nor network exposure is required.
package ipc

import "github.com/shruggietech/go-scheduler/internal/config"

// Endpoint resolves the IPC endpoint (socket path or pipe name) from config,
// falling back to the platform default when unset.
func Endpoint(cfg config.Config) string {
	if cfg.IPCPath != "" {
		return cfg.IPCPath
	}
	return defaultEndpoint(cfg)
}
