package app

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/valpiks/backupctl/internal/scheduler"
)

func TestJobsCommandListStatusDelete(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)

	store := scheduler.NewJSONStore(filepath.Join(tmp, ".backupctl", "jobs.json"))
	if err := store.Create(context.Background(), &scheduler.Job{
		ID:           "job_1",
		Interval:     "24h",
		DatabaseName: "app",
		BackupType:   "full",
		Format:       "plain",
		Status:       "scheduled",
	}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	listOut := executeJobsCommand(t)
	if !strings.Contains(listOut, "job_1") || !strings.Contains(listOut, "24h") {
		t.Fatalf("jobs output = %q, want job row", listOut)
	}

	statusOut := executeJobsCommand(t, "status", "job_1")
	if !strings.Contains(statusOut, "Job status") || !strings.Contains(statusOut, "job_1") || !strings.Contains(statusOut, "24h") {
		t.Fatalf("jobs status output = %q, want job details", statusOut)
	}

	deleteOut := executeJobsCommand(t, "delete", "job_1")
	if !strings.Contains(deleteOut, "job deleted: job_1") {
		t.Fatalf("jobs delete output = %q, want delete confirmation", deleteOut)
	}

	jobs, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(jobs) != 0 {
		t.Fatalf("jobs len after delete = %d, want 0", len(jobs))
	}
}

func TestJobsCommandEmptyState(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)

	out := executeJobsCommand(t)
	if !strings.Contains(out, "No scheduled jobs found.") {
		t.Fatalf("jobs output = %q, want empty state", out)
	}
}

func executeJobsCommand(t *testing.T, args ...string) string {
	t.Helper()

	cmd := newJobsCommand(CLIOptions{})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute(%v) error = %v", args, err)
	}

	return out.String()
}
