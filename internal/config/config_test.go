package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault_IsValid(t *testing.T) {
	if err := Default().Validate(); err != nil {
		t.Fatalf("default config should be valid, got: %v", err)
	}
}

func TestValidate_RejectsBadFields(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Config)
		field  string
	}{
		{"empty data dir", func(c *Config) { c.DataDir = "" }, "data_dir"},
		{"bad log level", func(c *Config) { c.LogLevel = "loud" }, "log_level"},
		{"bad log format", func(c *Config) { c.LogFormat = "xml" }, "log_format"},
		{"zero output cap", func(c *Config) { c.OutputCapBytes = 0 }, "output_cap_bytes"},
		{"negative workers", func(c *Config) { c.WorkerPoolSize = -1 }, "worker_pool_size"},
		{"bad timezone", func(c *Config) { c.DefaultTimezone = "Mars/Phobos" }, "default_timezone"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Default()
			tt.mutate(&c)
			err := c.Validate()
			if err == nil {
				t.Fatalf("expected validation error for %s", tt.name)
			}
			if !contains(err.Error(), tt.field) {
				t.Fatalf("error %q should name field %q", err, tt.field)
			}
		})
	}
}

func TestValidate_AcceptsLocalAndNamedZones(t *testing.T) {
	for _, tz := range []string{"", "Local", "America/New_York", "UTC"} {
		c := Default()
		c.DefaultTimezone = tz
		if err := c.Validate(); err != nil {
			t.Fatalf("timezone %q should be valid: %v", tz, err)
		}
	}
}

func TestLoad_MissingFileReturnsDefaults(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "does-not-exist.json"))
	if err != nil {
		t.Fatalf("missing file should yield defaults, got: %v", err)
	}
	if cfg.LogLevel != "info" {
		t.Fatalf("expected default log level, got %q", cfg.LogLevel)
	}
}

func TestLoad_OverlaysAndValidates(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{"log_level":"debug","worker_pool_size":4}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if cfg.LogLevel != "debug" || cfg.WorkerPoolSize != 4 {
		t.Fatalf("overlay not applied: %+v", cfg)
	}
	// Unspecified fields keep defaults.
	if cfg.LogFormat != "json" {
		t.Fatalf("expected default log_format json, got %q", cfg.LogFormat)
	}
}

func TestLoad_InvalidOverlayRejected(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{"log_format":"xml"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("expected invalid overlay to be rejected")
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
