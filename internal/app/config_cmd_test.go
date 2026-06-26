package app

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigParentStillPrints(t *testing.T) {
	tmp := t.TempDir()
	configPath := createTestConfig(t, tmp)

	out := executeConfigCommand(t, "--config", configPath)

	if !strings.Contains(out, "Config loaded") || !strings.Contains(out, "postgres/testdb") {
		t.Fatalf("config output = %q, want config summary", out)
	}
}

func TestConfigPrintSubcommandJSON(t *testing.T) {
	tmp := t.TempDir()
	configPath := createTestConfig(t, tmp)

	out := executeConfigCommand(t, "print", "--config", configPath, "--json")

	var got map[string]string
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v; output = %q", err, out)
	}

	if got["status"] != "success" || got["database"] != "postgres/testdb" {
		t.Fatalf("config print json = %#v, want success postgres/testdb", got)
	}
}

func TestConfigValidateCommandSuccess(t *testing.T) {
	tmp := t.TempDir()
	configPath := createTestConfig(t, tmp)

	out := executeConfigCommand(t, "validate", "--config", configPath)

	if !strings.Contains(out, "Config valid") || !strings.Contains(out, configPath) {
		t.Fatalf("config validate output = %q, want valid message", out)
	}
}

func TestConfigValidateCommandJSON(t *testing.T) {
	tmp := t.TempDir()
	configPath := createTestConfig(t, tmp)

	out := executeConfigCommand(t, "validate", "--config", configPath, "--json")

	var got map[string]string
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v; output = %q", err, out)
	}

	if got["status"] != "success" || got["path"] != configPath {
		t.Fatalf("config validate json = %#v, want success/path", got)
	}
}

func TestConfigValidateCommandFailure(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "config.yaml")
	writeConfigCommandTestFile(t, configPath, `
database:
  type: mysql
backup:
  type: full
storage:
  type: local
  local:
    path: ./backups
logging:
  level: info
`)

	cmd := newConfigCommand(CLIOptions{})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"validate", "--config", configPath})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want invalid config error")
	}

	if !strings.Contains(err.Error(), "invalid config") || !strings.Contains(err.Error(), "unsupported database.type") {
		t.Fatalf("Execute() error = %v, want invalid unsupported database.type", err)
	}
}

func TestConfigValidateCommandFailureJSON(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "config.yaml")
	writeConfigCommandTestFile(t, configPath, `
database:
  type: postgres
  postgres:
    host: localhost
    port: 5432
    user: postgres
    name: app
backup:
  type: full
  compression: zip
storage:
  type: local
  local:
    path: ./backups
logging:
  level: info
`)

	cmd := newConfigCommand(CLIOptions{})
	cmd.SilenceUsage = true
	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs([]string{"validate", "--config", configPath, "--json"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want invalid config error")
	}

	var got map[string]string
	if jsonErr := json.Unmarshal(out.Bytes(), &got); jsonErr != nil {
		t.Fatalf("json.Unmarshal() error = %v; output = %q", jsonErr, out.String())
	}

	if got["status"] != "failed" || !strings.Contains(got["error"], "unsupported backup.compression") {
		t.Fatalf("config validate failure json = %#v, want failed compression error", got)
	}
}

func executeConfigCommand(t *testing.T, args ...string) string {
	t.Helper()

	cmd := newConfigCommand(CLIOptions{})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute(%v) error = %v", args, err)
	}

	return out.String()
}

func writeConfigCommandTestFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)), 0600); err != nil {
		t.Fatalf("write config file: %v", err)
	}
}
