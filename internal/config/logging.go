package config

import (
	"io"
	"log/slog"
)

// NewLogger builds a structured slog.Logger from the configuration. Per the
// constitution's UX-consistency principle, logging is structured with
// consistent levels and fields across all components.
func NewLogger(c Config, w io.Writer) *slog.Logger {
	opts := &slog.HandlerOptions{Level: parseLevel(c.LogLevel)}
	var h slog.Handler
	if c.LogFormat == "text" {
		h = slog.NewTextHandler(w, opts)
	} else {
		h = slog.NewJSONHandler(w, opts)
	}
	return slog.New(h)
}

func parseLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
