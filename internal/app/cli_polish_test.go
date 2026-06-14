package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/valpiks/backupctl/internal/scheduler"
)

func TestRootHelpContainsExamplesAndCompletion(t *testing.T) {
	cmd := NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	got := out.String()
	for _, want := range []string{"backupctl doctor", "backupctl backup", "completion"} {
		if !strings.Contains(got, want) {
			t.Fatalf("help output = %q, want %q", got, want)
		}
	}
}

func TestCompletionCommandZsh(t *testing.T) {
	cmd := NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"completion", "zsh"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !strings.Contains(out.String(), "#compdef") {
		t.Fatalf("completion output = %q, want zsh completion", out.String())
	}
}

func TestVersionCommandJSON(t *testing.T) {
	cmd := newVersionCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var got map[string]string
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v; output = %q", err, out.String())
	}

	if got["version"] == "" || got["go"] == "" || got["os"] == "" || got["arch"] == "" {
		t.Fatalf("version json = %#v, want version/go/os/arch", got)
	}
}

func TestConfigCommandJSON(t *testing.T) {
	tmp := t.TempDir()
	configPath := createTestConfig(t, tmp)

	cmd := newConfigCommand(CLIOptions{})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--config", configPath, "--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var got map[string]string
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v; output = %q", err, out.String())
	}

	if got["database"] != "postgres/testdb" || got["status"] != "success" {
		t.Fatalf("config json = %#v, want database/status", got)
	}
}

func TestJobsCommandJSON(t *testing.T) {
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

	cmd := newJobsCommand(CLIOptions{})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var got []scheduler.Job
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v; output = %q", err, out.String())
	}

	if len(got) != 1 || got[0].ID != "job_1" {
		t.Fatalf("jobs json = %#v, want job_1", got)
	}
}

func TestPrintErrorWithHint(t *testing.T) {
	var out bytes.Buffer

	PrintError(&out, WithHint(errors.New("failed"), "try this"))

	got := out.String()
	if !strings.Contains(got, "failed") || !strings.Contains(got, "hint: try this") {
		t.Fatalf("PrintError output = %q, want error and hint", got)
	}
}

func TestPrintDoctorChecksJSON(t *testing.T) {
	var out bytes.Buffer
	checks := []DoctorCheck{{Status: "OK", Check: "config loaded", Details: "configs/config.yaml"}}

	if err := printDoctorChecks(&out, checks, true, "never"); err != nil {
		t.Fatalf("printDoctorChecks() error = %v", err)
	}

	var got []DoctorCheck
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v; output = %q", err, out.String())
	}

	if len(got) != 1 || got[0].Status != "OK" {
		t.Fatalf("doctor json = %#v, want OK check", got)
	}
}
