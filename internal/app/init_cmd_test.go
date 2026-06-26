package app

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/valpiks/backupctl/internal/config"
)

func TestInitCommandCreatesConfigAndEnvFile(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "backupctl", "config.yaml")
	envPath := filepath.Join(tmp, "backupctl", "backupctl.env")

	out := executeInitCommand(t,
		"--database", "postgres",
		"--storage", "local",
		"--output", configPath,
	)

	if !strings.Contains(out, "Config created") || !strings.Contains(out, "backupctl doctor") {
		t.Fatalf("init output = %q, want created message and next steps", out)
	}

	assertFileContains(t, configPath, "type: postgres")
	assertFileContains(t, configPath, "password_env: BACKUPCTL_POSTGRES_PASSWORD")
	assertFileContains(t, configPath, "env_file: "+envPath)
	assertFileContains(t, envPath, "BACKUPCTL_POSTGRES_PASSWORD=change-me")

	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("config.Load(%q) error = %v", configPath, err)
	}

	if cfg.Database.Type != "postgres" || cfg.Storage.Type != "local" {
		t.Fatalf("loaded config database/storage = %s/%s, want postgres/local", cfg.Database.Type, cfg.Storage.Type)
	}
}

func TestInitCommandCreatesMongoS3Config(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "config.yaml")

	executeInitCommand(t,
		"--database", "mongo",
		"--storage", "s3",
		"--output", configPath,
		"--env=false",
	)

	assertFileContains(t, configPath, "type: mongo")
	assertFileContains(t, configPath, "uri_env: BACKUPCTL_MONGO_URI")
	assertFileContains(t, configPath, "type: s3")
	assertFileContains(t, configPath, "bucket: your-bucket")
	assertFileNotContains(t, configPath, "env_file:")
}

func TestInitCommandRefusesOverwrite(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "config.yaml")

	executeInitCommand(t, "--output", configPath)

	cmd := newInitCommand(CLIOptions{})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--output", configPath})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want overwrite error")
	}

	if !strings.Contains(err.Error(), "config already exists") {
		t.Fatalf("Execute() error = %v, want config already exists", err)
	}
}

func TestInitCommandForceOverwrites(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "config.yaml")

	if err := os.WriteFile(configPath, []byte("old"), 0600); err != nil {
		t.Fatalf("write old config: %v", err)
	}

	executeInitCommand(t, "--output", configPath, "--force")

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	if string(data) == "old" {
		t.Fatal("config was not overwritten")
	}

	if !strings.Contains(string(data), "database:") {
		t.Fatalf("config = %q, want database section", string(data))
	}
}

func TestInitCommandUsesDefaultConfigPathFromEnv(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "config.yaml")
	t.Setenv("BACKUPCTL_CONFIG", configPath)

	executeInitCommand(t)

	assertFileContains(t, configPath, "database:")
}

func TestInitCommandValidation(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "unsupported database",
			args:    []string{"--database", "mysql", "--output", filepath.Join(t.TempDir(), "config.yaml")},
			wantErr: "unsupported database",
		},
		{
			name:    "unsupported storage",
			args:    []string{"--storage", "ftp", "--output", filepath.Join(t.TempDir(), "config.yaml")},
			wantErr: "unsupported storage",
		},
		{
			name:    "missing output",
			args:    []string{"--output", ""},
			wantErr: "output path is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newInitCommand(CLIOptions{})
			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			if err == nil {
				t.Fatal("Execute() error = nil")
			}

			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Execute() error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func executeInitCommand(t *testing.T, args ...string) string {
	t.Helper()

	cmd := newInitCommand(CLIOptions{})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute(%v) error = %v", args, err)
	}

	return out.String()
}

func assertFileContains(t *testing.T, path string, want string) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	if !strings.Contains(string(data), want) {
		t.Fatalf("%s = %q, want to contain %q", path, string(data), want)
	}
}

func assertFileNotContains(t *testing.T, path string, want string) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	if strings.Contains(string(data), want) {
		t.Fatalf("%s = %q, want not to contain %q", path, string(data), want)
	}
}
