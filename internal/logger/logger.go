package logger

import (
	"log/slog"
	"os"
)

func New(level string, format ...string) *slog.Logger {
	return newLogger(os.Stderr, level, firstFormat(format))
}

func NewCommand(level string, format ...string) *slog.Logger {
	if level != "debug" {
		level = "warn"
	}

	return newLogger(os.Stderr, level, firstFormat(format))
}

func newLogger(out *os.File, level string, format string) *slog.Logger {
	var slogLevel slog.Level

	switch level {
	case "debug":
		slogLevel = slog.LevelDebug
	case "warn":
		slogLevel = slog.LevelWarn
	case "error":
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: slogLevel,
	}

	if format == "json" {
		return slog.New(slog.NewJSONHandler(out, opts))
	}

	handler := slog.NewTextHandler(out, opts)

	return slog.New(handler)
}

func firstFormat(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
