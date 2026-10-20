package logging

import (
	"fmt"
	"io"
	"log/slog"
)

// New returns a logger that writes to w. format is "text" or "json",
// and level is "debug", "info", "warn", or "error".
func New(w io.Writer, format, level string) (*slog.Logger, error) {
	var l slog.Level
	if err := l.UnmarshalText([]byte(level)); err != nil {
		return nil, fmt.Errorf("log level: %w", err)
	}

	opts := &slog.HandlerOptions{Level: l}

	switch format {
	case "text":
		return slog.New(slog.NewTextHandler(w, opts)), nil
	case "json":
		return slog.New(slog.NewJSONHandler(w, opts)), nil
	default:
		return nil, fmt.Errorf("log format: unknown format %q", format)
	}
}
