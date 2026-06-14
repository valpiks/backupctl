package logger

import (
	"log/slog"
	"os"
)

func New(level string) *slog.Logger {
	return newLogger(os.Stderr, level)
}

func NewCommand(level string) *slog.Logger {
	if level != "debug" {
		level = "warn"
	}

	return newLogger(os.Stderr, level)
}

func newLogger(out *os.File, level string) *slog.Logger {
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

	handler := slog.NewTextHandler(out, &slog.HandlerOptions{
		Level: slogLevel,
	})

	return slog.New(handler)
}
