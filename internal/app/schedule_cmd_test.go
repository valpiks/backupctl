package app

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/valpiks/backupctl/internal/scheduler"
)

func TestScheduleCommandCreatesJob(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)

	configPath := createTestConfig(t, tmp)
	cmd := newScheduleCommand(CLIOptions{})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--interval", "24h", "--config", configPath, "--format", "custom"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !strings.Contains(out.String(), "Job created") {
		t.Fatalf("output = %q, want job created", out.String())
	}

	store := scheduler.NewJSONStore(filepath.Join(tmp, ".backupctl", "jobs.json"))
	jobs, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(jobs) != 1 {
		t.Fatalf("jobs len = %d, want 1", len(jobs))
	}

	if jobs[0].Interval != "24h" {
		t.Fatalf("job Interval = %q, want 24h", jobs[0].Interval)
	}

	if jobs[0].Format != "custom" {
		t.Fatalf("job Format = %q, want custom", jobs[0].Format)
	}
}

func TestScheduleCommandValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "missing schedule",
			args:    []string{},
			wantErr: "either --cron or --interval is required",
		},
		{
			name:    "cron and interval together",
			args:    []string{"--cron", "0 3 * * *", "--interval", "24h"},
			wantErr: "--cron and --interval cannot be used together",
		},
		{
			name:    "invalid interval",
			args:    []string{"--interval", "bad"},
			wantErr: "invalid interval",
		},
		{
			name:    "invalid format",
			args:    []string{"--interval", "24h", "--format", "zip"},
			wantErr: "unsupported format",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cmd := newScheduleCommand(CLIOptions{})
			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			if err == nil {
				t.Fatal("Execute() error = nil")
			}

			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Execute() error = %v, want to contain %q", err, tt.wantErr)
			}
		})
	}
}
