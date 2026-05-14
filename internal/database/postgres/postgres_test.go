package postgres

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"github.com/valpiks/backupctl/internal/config"
	database "github.com/valpiks/backupctl/internal/dbdriver"
)

func TestNewDriver(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     config.DatabaseConfig
		wantErr string
	}{
		{name: "valid postgres config",
			cfg: config.DatabaseConfig{
				Type: "postgres",
				Postgres: config.PostgresConfig{
					Host:     "localhost",
					Port:     5432,
					User:     "postgres",
					Password: "secret",
					Name:     "app",
					SSLMode:  "disable",
				},
			}},
		{name: "unsupported type",
			cfg: config.DatabaseConfig{
				Type: "mongo",
			},
			wantErr: "unsupported database type for postgres driver: mongo"},
		{name: "missing host",
			cfg: config.DatabaseConfig{
				Type: "postgres",
			}, wantErr: "postgres.host is required"},
		{name: "missing port",
			cfg: config.DatabaseConfig{
				Type: "postgres",
				Postgres: config.PostgresConfig{
					Host: "localhost",
				},
			}, wantErr: "postgres.port is required"},
		{name: "missing user",
			cfg: config.DatabaseConfig{
				Type: "postgres",
				Postgres: config.PostgresConfig{
					Host: "localhost",
					Port: 5432,
				},
			}, wantErr: "postgres.user is required"},
		{name: "missing name",
			cfg: config.DatabaseConfig{
				Type: "postgres",
				Postgres: config.PostgresConfig{
					Host: "localhost",
					Port: 5432,
					User: "postgres",
				},
			}, wantErr: "postgres.name is required"},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			driver, err := NewDriver(tt.cfg)

			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("NewDriver() error = %v", err)
				}
				if driver == nil {
					t.Fatal("NewDriver() returned nil driver")
				}
				return
			}

			if err == nil {
				t.Fatalf("NewDriver() error = nil, want %q", tt.wantErr)
			}

			if err.Error() != tt.wantErr {
				t.Fatalf("NewDriver() error = %q, want %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestBackupRejectsUnsupportedType(t *testing.T) {
	t.Parallel()

	driver := &Driver{
		cfg: validPostgresDatabaseConfig(),
	}

	reader, err := driver.Backup(context.Background(), database.BackupOptions{Type: "incremental"})
	if err == nil {
		t.Fatal("Backup() error = nil, want error")
	}

	if reader != nil {
		t.Fatal("Backup() reader != nil")
	}

	if err.Error() != "unsupported postgres backup type: incremental" {
		t.Fatalf("Backup() error = %q", err.Error())
	}
}

func TestRestoreUsesExpectedCommandArgs(t *testing.T) {
	tests := []struct {
		name     string
		targetDB string
		format   string
		wantCmd  string
		wantArgs []string
	}{
		{
			name:     "uses configured database by default",
			targetDB: "",
			format:   "plain",
			wantCmd:  "psql",
			wantArgs: []string{
				"-h", "localhost",
				"-p", "5432",
				"-U", "postgres",
				"-d", "testdb",
			},
		}, {
			name:     "uses target database override",
			targetDB: "restoredb",
			format:   "custom",
			wantCmd:  "pg_restore",
			wantArgs: []string{
				"-h", "localhost",
				"-p", "5432",
				"-U", "postgres",
				"-d", "restoredb",
				"--clean",
				"--if-exists",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			cfg := validPostgresDatabaseConfig()
			cfg.Postgres.Name = "testdb"
			driver := &Driver{cfg: cfg}

			argsFile := createCapturedArgsFile(t)
			restoreExecCommandForTest(t)
			execCommandContext = fakeExecCommandContext(t, argsFile)

			if err := driver.Restore(context.Background(), strings.NewReader("archive"), database.RestoreOptions{
				TargetDatabase: tt.targetDB,
				Format:         tt.format,
			}); err != nil {
				t.Fatalf("Restore() error = %v", err)
			}

			gotCmd := readCapturedArgs(t, argsFile)
			if len(gotCmd) == 0 {
				t.Fatal("no command captured")
			}
			if gotCmd[0] != tt.wantCmd {
				t.Fatalf("Restore() command = %q, want %q", gotCmd[0], tt.wantCmd)
			}

			if !reflect.DeepEqual(gotCmd[1:], tt.wantArgs) {
				t.Fatalf("Restore() args = %v, want %v", gotCmd[1:], tt.wantArgs)
			}
		})
	}
}

func readCapturedArgs(t *testing.T, path string) []string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	content := strings.TrimSpace(string(data))
	if content == "" {
		return nil
	}

	return strings.Split(content, "\n")
}

func restoreExecCommandForTest(t *testing.T) {
	t.Helper()

	previous := execCommandContext
	t.Cleanup(func() {
		execCommandContext = previous
	})
}

func createCapturedArgsFile(t *testing.T) string {
	t.Helper()

	file, err := os.CreateTemp(t.TempDir(), "postgres-args-*.txt")
	if err != nil {
		t.Fatalf("CreateTemp() error = %v", err)
	}

	if err := file.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	return file.Name()
}

func fakeExecCommandContext(t *testing.T, argsFile string) func(context.Context, string, ...string) *exec.Cmd {
	t.Helper()

	return func(ctx context.Context, name string, args ...string) *exec.Cmd {
		t.Helper()

		commandArgs := append([]string{"-test.run=TestHelperProcess", "--", name}, args...)
		cmd := exec.CommandContext(ctx, os.Args[0], commandArgs...)
		cmd.Env = append(os.Environ(),
			"GO_WANT_HELPER_PROCESS=1",
			"BACKUPCTL_POSTGRES_TEST_ARGS_FILE="+argsFile,
		)
		return cmd
	}
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	args := os.Args
	for i, arg := range args {
		if arg == "--" {
			captured := strings.Join(args[i+1:], "\n")
			if err := os.WriteFile(os.Getenv("BACKUPCTL_POSTGRES_TEST_ARGS_FILE"), []byte(captured), 0644); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

			if _, err := io.Copy(io.Discard, os.Stdin); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

			os.Exit(0)
		}
	}

	fmt.Fprintln(os.Stderr, "helper arguments not found")
	os.Exit(1)
}

func validPostgresDatabaseConfig() config.DatabaseConfig {
	return config.DatabaseConfig{
		Type: "postgres",
		Postgres: config.PostgresConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "postgres",
			Password: "secret",
			Name:     "app",
			SSLMode:  "disable",
		},
	}
}
