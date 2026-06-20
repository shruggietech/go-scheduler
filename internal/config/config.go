// Package config defines the single configuration schema for the scheduler,
// its defaults, and fail-fast validation. Per the constitution's UX-consistency
// principle, invalid configuration is rejected at startup with an actionable
// message naming the offending field.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/shruggietech/go-scheduler/internal/platform"
)

// Config is the complete, documented configuration for the daemon. All fields
// have defaults (see Default); a config file may override any subset.
type Config struct {
	// DataDir is the directory holding the database, IPC endpoint, and logs.
	DataDir string `json:"data_dir"`
	// IPCPath is the Unix socket path (Unix) or named-pipe name (Windows) the
	// daemon listens on. Empty means "derive from DataDir / platform default".
	IPCPath string `json:"ipc_path"`
	// AdminGroup is the OS group permitted to manage the daemon over IPC.
	AdminGroup string `json:"admin_group"`
	// DefaultTimezone is the IANA zone applied to tasks that do not pin one.
	// "Local" resolves to the host's local timezone.
	DefaultTimezone string `json:"default_timezone"`
	// LogLevel is one of: debug, info, warn, error.
	LogLevel string `json:"log_level"`
	// LogFormat is one of: json, text.
	LogFormat string `json:"log_format"`
	// OutputCapBytes bounds captured stdout/stderr per run.
	OutputCapBytes int `json:"output_cap_bytes"`
	// WorkerPoolSize bounds concurrent task executions.
	WorkerPoolSize int `json:"worker_pool_size"`
}

// Default returns the built-in configuration.
func Default() Config {
	dir := platform.DataDir()
	return Config{
		DataDir:         dir,
		IPCPath:         "", // resolved by the IPC layer from DataDir when empty
		AdminGroup:      "goschedadmin",
		DefaultTimezone: "Local",
		LogLevel:        "info",
		LogFormat:       "json",
		OutputCapBytes:  1 << 20, // 1 MiB
		WorkerPoolSize:  16,
	}
}

// DBPath returns the database file path derived from DataDir.
func (c Config) DBPath() string { return filepath.Join(c.DataDir, "goscheduler.db") }

var (
	validLevels  = map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	validFormats = map[string]bool{"json": true, "text": true}
)

// Validate checks the configuration and returns the first problem found, naming
// the offending field. It is intentionally strict (fail fast at startup).
func (c Config) Validate() error {
	if c.DataDir == "" {
		return fmt.Errorf("config: data_dir must not be empty")
	}
	if !validLevels[c.LogLevel] {
		return fmt.Errorf("config: log_level %q is invalid (want one of debug, info, warn, error)", c.LogLevel)
	}
	if !validFormats[c.LogFormat] {
		return fmt.Errorf("config: log_format %q is invalid (want json or text)", c.LogFormat)
	}
	if c.OutputCapBytes <= 0 {
		return fmt.Errorf("config: output_cap_bytes must be positive, got %d", c.OutputCapBytes)
	}
	if c.WorkerPoolSize <= 0 {
		return fmt.Errorf("config: worker_pool_size must be positive, got %d", c.WorkerPoolSize)
	}
	if err := validateTimezone(c.DefaultTimezone); err != nil {
		return err
	}
	return nil
}

// validateTimezone ensures the zone is resolvable. "Local" and "" are accepted
// as "use the host local zone".
func validateTimezone(tz string) error {
	if tz == "" || tz == "Local" {
		return nil
	}
	if _, err := time.LoadLocation(tz); err != nil {
		return fmt.Errorf("config: default_timezone %q is not a valid IANA zone: %w", tz, err)
	}
	return nil
}

// Load reads a JSON config file layered over Default and validates the result.
// A missing path returns the validated defaults (not an error), so the daemon
// can run with zero configuration.
func Load(path string) (Config, error) {
	cfg := Default()
	if path != "" {
		data, err := os.ReadFile(path)
		switch {
		case err == nil:
			if err := json.Unmarshal(data, &cfg); err != nil {
				return Config{}, fmt.Errorf("config: parsing %s: %w", path, err)
			}
		case os.IsNotExist(err):
			// fall through to validated defaults
		default:
			return Config{}, fmt.Errorf("config: reading %s: %w", path, err)
		}
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
