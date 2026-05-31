package envfile

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	path := writeTempEnvFile(t, `
BACKUPCTL_POSTGRES_PASSWORD=postgres-secret
BACKUPCTL_ENCRYPTION_PASSWORD=encryption-secret
EMPTY_VALUE=
`)

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	want := map[string]string{
		"BACKUPCTL_POSTGRES_PASSWORD":   "postgres-secret",
		"BACKUPCTL_ENCRYPTION_PASSWORD": "encryption-secret",
		"EMPTY_VALUE":                   "",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Load() = %#v, want %#v", got, want)
	}
}

func TestLoadIgnoresCommentsAndBlankLines(t *testing.T) {
	path := writeTempEnvFile(t, `
# comment

BACKUPCTL_POSTGRES_PASSWORD=postgres-secret

  # indented comment
`)

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	want := map[string]string{
		"BACKUPCTL_POSTGRES_PASSWORD": "postgres-secret",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Load() = %#v, want %#v", got, want)
	}
}

func TestLoadTrimsQuotedValues(t *testing.T) {
	path := writeTempEnvFile(t, `
BACKUPCTL_POSTGRES_PASSWORD = "postgres secret"
AWS_ACCESS_KEY_ID='access-key'
AWS_SECRET_ACCESS_KEY= secret-key
`)

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	want := map[string]string{
		"BACKUPCTL_POSTGRES_PASSWORD": "postgres secret",
		"AWS_ACCESS_KEY_ID":           "access-key",
		"AWS_SECRET_ACCESS_KEY":       "secret-key",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Load() = %#v, want %#v", got, want)
	}
}

func TestLoadRejectsInvalidLine(t *testing.T) {
	path := writeTempEnvFile(t, `BACKUPCTL_POSTGRES_PASSWORD`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() error = nil")
	}

	if !strings.Contains(err.Error(), "line 1: expected KEY=value") {
		t.Fatalf("Load() error = %v", err)
	}
}

func TestLoadRejectsInvalidKey(t *testing.T) {
	tests := []struct {
		name string
		line string
	}{
		{
			name: "empty",
			line: "=secret",
		},
		{
			name: "hyphen",
			line: "BACKUPCTL-POSTGRES-PASSWORD=secret",
		},
		{
			name: "export prefix",
			line: "export BACKUPCTL_POSTGRES_PASSWORD=secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeTempEnvFile(t, tt.line)

			_, err := Load(path)
			if err == nil {
				t.Fatal("Load() error = nil")
			}

			if !strings.Contains(err.Error(), "invalid key") {
				t.Fatalf("Load() error = %v", err)
			}
		})
	}
}

func TestApply(t *testing.T) {
	t.Setenv("BACKUPCTL_TEST_APPLY", "")

	err := Apply(map[string]string{
		"BACKUPCTL_TEST_APPLY": "from-apply",
	})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	if got := os.Getenv("BACKUPCTL_TEST_APPLY"); got != "from-apply" {
		t.Fatalf("Getenv() = %q, want from-apply", got)
	}
}

func TestLoadAndApply(t *testing.T) {
	t.Setenv("BACKUPCTL_TEST_LOAD_AND_APPLY", "")
	path := writeTempEnvFile(t, `BACKUPCTL_TEST_LOAD_AND_APPLY=from-file`)

	if err := LoadAndApply(path); err != nil {
		t.Fatalf("LoadAndApply() error = %v", err)
	}

	if got := os.Getenv("BACKUPCTL_TEST_LOAD_AND_APPLY"); got != "from-file" {
		t.Fatalf("Getenv() = %q, want from-file", got)
	}
}

func writeTempEnvFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "backupctl.env")
	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	return path
}
