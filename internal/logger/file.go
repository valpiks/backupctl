package logger

import (
	"log/slog"
	"os"
	"path/filepath"
)

func NewFile(level string, path string) (*slog.Logger, func() error, error) {
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

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, nil, err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, nil, err
	}

	handler := slog.NewTextHandler(file, &slog.HandlerOptions{
		Level: slogLevel,
	})

	return slog.New(handler), file.Close, nil
}
