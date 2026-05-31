package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBackupCommand_ValidateFlags(t *testing.T) {
	tests := []struct {
		name       string
		schemaOnly bool
		dataOnly   bool
		format     string
		wantErr    string
	}{
		{
			name:       "schema-only and data-only together should fail",
			schemaOnly: true,
			dataOnly:   true,
			wantErr:    "cannot be used together",
		},
		{
			name:    "unsupported format should fail",
			format:  "zip",
			wantErr: "unsupported format",
		},
		{
			name:    "valid plain format",
			format:  "plain",
			wantErr: "",
		},
		{
			name:    "valid custom format",
			format:  "custom",
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := t.TempDir()
			path := createTestConfig(t, tmp)

			err := validateFlags(tt.schemaOnly, tt.dataOnly, tt.format, path)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %q, want to contain %q", err.Error(), tt.wantErr)
				}
			} else {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
			}
		})
	}
}

func validateFlags(schemaOnly bool, dataOnly bool, format string, configPath string) error {
	if schemaOnly && dataOnly {
		return newFlagError("--schema-only and --data-only cannot be used together")
	}

	if format != "plain" && format != "custom" {
		return newFlagError("unsupported format: " + format)
	}

	return nil
}

func createTestConfig(t *testing.T, tmpDir string) string {
	t.Helper()

	path := filepath.Join(tmpDir, "test-config.yaml")
	content := `database:
  type: postgres
  postgres:
    host: localhost
    port: 5432
    user: postgres
    password: secret
    name: testdb
  mongo:
    uri: mongodb://localhost:27017
backup:
  type: full
  compression: gzip
storage:
  type: local
  local:
    path: /tmp/backups
logging:
  level: info
`

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write test config: %v", err)
	}

	return path
}

func newFlagError(msg string) error {
	return &flagError{msg: msg}
}

type flagError struct {
	msg string
}

func (e *flagError) Error() string {
	return e.msg
}
