package logger

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "logs", "backupctl.log")

	log, closeLog, err := NewFile("info", path)
	if err != nil {
		t.Fatalf("NewFile() error = %v", err)
	}

	log.Info("scheduler started", "job", "job_1")

	if err := closeLog(); err != nil {
		t.Fatalf("closeLog() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	got := string(data)
	if !strings.Contains(got, "scheduler started") {
		t.Fatalf("log file = %q, want scheduler message", got)
	}

	if !strings.Contains(got, "job_1") {
		t.Fatalf("log file = %q, want job attribute", got)
	}
}

func TestNewFileLevelFiltering(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "backupctl.log")

	log, closeLog, err := NewFile("error", path)
	if err != nil {
		t.Fatalf("NewFile() error = %v", err)
	}

	log.Info("should be filtered")
	log.ErrorContext(context.Background(), "should be written")

	if err := closeLog(); err != nil {
		t.Fatalf("closeLog() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	got := string(data)
	if strings.Contains(got, "should be filtered") {
		t.Fatalf("log file = %q, info log should be filtered", got)
	}

	if !strings.Contains(got, "should be written") {
		t.Fatalf("log file = %q, error log missing", got)
	}
}
