package app

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/valpiks/backupctl/internal/backup"
	"github.com/valpiks/backupctl/internal/storage"
)

type fakeRestoreStorage struct {
	files    map[string]string
	metadata map[string][]byte
}

func (s fakeRestoreStorage) Save(ctx context.Context, name string, data io.Reader) error {
	return nil
}

func (s fakeRestoreStorage) Open(ctx context.Context, name string) (io.ReadCloser, error) {
	value, ok := s.files[name]
	if !ok {
		return nil, os.ErrNotExist
	}
	return io.NopCloser(strings.NewReader(value)), nil
}

func (s fakeRestoreStorage) List(ctx context.Context) ([]storage.BackupFile, error) {
	return nil, nil
}

func (s fakeRestoreStorage) Delete(ctx context.Context, name string) error {
	return nil
}

func (s fakeRestoreStorage) ReadMetadata(ctx context.Context, name string) ([]byte, error) {
	value, ok := s.metadata[name]
	if !ok {
		return nil, os.ErrNotExist
	}
	return value, nil
}

func TestBuildRestorePlanFromMetadata(t *testing.T) {
	metadata := backup.Metadata{
		DatabaseName: "app",
		Format:       "custom",
		Compression:  "",
		Encryption: &backup.EncryptionMetadata{
			Enabled: true,
		},
	}
	data, err := json.Marshal(metadata)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	st := fakeRestoreStorage{
		files: map[string]string{
			"app.dump.enc": "backup",
		},
		metadata: map[string][]byte{
			"app.metadata.json": data,
		},
	}

	plan, err := buildRestorePlan(context.Background(), st, "app.dump.enc", "app_restore")
	if err != nil {
		t.Fatalf("buildRestorePlan() error = %v", err)
	}

	if !plan.MetadataFound {
		t.Fatal("MetadataFound = false, want true")
	}

	if plan.SourceDB != "app" {
		t.Fatalf("SourceDB = %q, want app", plan.SourceDB)
	}

	if plan.TargetDB != "app_restore" {
		t.Fatalf("TargetDB = %q, want app_restore", plan.TargetDB)
	}

	if plan.Format != "custom" {
		t.Fatalf("Format = %q, want custom", plan.Format)
	}

	if !plan.Encrypted {
		t.Fatal("Encrypted = false, want true")
	}
}

func TestBuildRestorePlanFallbackFromFileName(t *testing.T) {
	st := fakeRestoreStorage{
		files: map[string]string{
			"app.sql.gz.enc": "backup",
		},
	}

	plan, err := buildRestorePlan(context.Background(), st, "app.sql.gz.enc", "app")
	if err != nil {
		t.Fatalf("buildRestorePlan() error = %v", err)
	}

	if plan.MetadataFound {
		t.Fatal("MetadataFound = true, want false")
	}

	if plan.Format != "plain" {
		t.Fatalf("Format = %q, want plain", plan.Format)
	}

	if !plan.Encrypted {
		t.Fatal("Encrypted = false, want true")
	}

	if plan.TargetDB != "app" {
		t.Fatalf("TargetDB = %q, want app", plan.TargetDB)
	}
}

func TestBuildRestorePlanMissingFile(t *testing.T) {
	st := fakeRestoreStorage{}

	_, err := buildRestorePlan(context.Background(), st, "missing.sql.gz", "app")
	if err == nil {
		t.Fatal("buildRestorePlan() error = nil")
	}

	if !strings.Contains(err.Error(), "backup file not found") {
		t.Fatalf("error = %v, want backup file not found", err)
	}
}

func TestConfirmRestoreYes(t *testing.T) {
	plan := &restorePlan{
		FileName:      "app.sql.gz",
		SourceDB:      "app",
		TargetDB:      "app_restore",
		Format:        "plain",
		MetadataFound: true,
	}

	var out bytes.Buffer
	ok, err := confirmRestore(strings.NewReader("yes\n"), &out, plan)
	if err != nil {
		t.Fatalf("confirmRestore() error = %v", err)
	}

	if !ok {
		t.Fatal("confirmRestore() = false, want true")
	}

	got := out.String()
	if !strings.Contains(got, "You are about to restore") ||
		!strings.Contains(got, "app_restore") ||
		!strings.Contains(got, "metadata:") {
		t.Fatalf("output = %q, want restore summary", got)
	}
}

func TestConfirmRestoreNo(t *testing.T) {
	plan := &restorePlan{FileName: "app.sql.gz", TargetDB: "app", Format: "plain"}

	var out bytes.Buffer
	ok, err := confirmRestore(strings.NewReader("no\n"), &out, plan)
	if err != nil {
		t.Fatalf("confirmRestore() error = %v", err)
	}

	if ok {
		t.Fatal("confirmRestore() = true, want false")
	}
}

func TestRestorePlanPayloadDryRun(t *testing.T) {
	plan := &restorePlan{
		FileName:      "app.sql.gz",
		SourceDB:      "app",
		TargetDB:      "app_restore",
		Format:        "plain",
		Compression:   "gzip",
		Encrypted:     true,
		MetadataFound: true,
	}

	payload := restorePlanPayload(plan, true)

	if payload["message"] != "restore dry run passed" {
		t.Fatalf("message = %v, want restore dry run passed", payload["message"])
	}

	if payload["dry_run"] != true {
		t.Fatalf("dry_run = %v, want true", payload["dry_run"])
	}

	if payload["metadata_found"] != true {
		t.Fatalf("metadata_found = %v, want true", payload["metadata_found"])
	}
}

func TestRequireInteractiveRestoreRejectsNonInteractiveInput(t *testing.T) {
	cmd := newRestoreCommand(CLIOptions{})
	cmd.SetIn(strings.NewReader("yes\n"))

	err := requireInteractiveRestore(cmd, false)
	if err == nil {
		t.Fatal("requireInteractiveRestore() error = nil")
	}

	if !strings.Contains(err.Error(), "restore confirmation requires interactive input") {
		t.Fatalf("error = %v, want interactive input error", err)
	}
}

func TestRequireInteractiveRestoreAllowsYes(t *testing.T) {
	cmd := newRestoreCommand(CLIOptions{})

	if err := requireInteractiveRestore(cmd, true); err != nil {
		t.Fatalf("requireInteractiveRestore() error = %v", err)
	}
}

func TestRestoreCommandDryRunWithLocalStorage(t *testing.T) {
	tmp := t.TempDir()
	backupsDir := filepath.Join(tmp, "backups")
	if err := os.MkdirAll(backupsDir, 0755); err != nil {
		t.Fatalf("mkdir backups: %v", err)
	}

	backupFile := "app_2026-06-14_12-00-00.sql.gz"
	if err := os.WriteFile(filepath.Join(backupsDir, backupFile), []byte("backup"), 0600); err != nil {
		t.Fatalf("write backup file: %v", err)
	}

	metadata := backup.Metadata{
		DatabaseName: "app",
		FileName:     backupFile,
		Format:       "plain",
		Compression:  "gzip",
	}
	metadataData, err := json.Marshal(metadata)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	if err := os.WriteFile(filepath.Join(backupsDir, metadataNameForBackup(backupFile)), metadataData, 0600); err != nil {
		t.Fatalf("write metadata file: %v", err)
	}

	configPath := filepath.Join(tmp, "config.yaml")
	writeRestoreConfig(t, configPath, backupsDir)

	cmd := newRestoreCommand(CLIOptions{})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--config", configPath, "--file", backupFile, "--dry-run"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Restore dry run passed") ||
		!strings.Contains(got, "source:") ||
		!strings.Contains(got, "app") ||
		!strings.Contains(got, "metadata:  found") {
		t.Fatalf("restore dry-run output = %q, want restore plan", got)
	}
}

func TestRestoreCommandDryRunJSONWithLocalStorage(t *testing.T) {
	tmp := t.TempDir()
	backupsDir := filepath.Join(tmp, "backups")
	if err := os.MkdirAll(backupsDir, 0755); err != nil {
		t.Fatalf("mkdir backups: %v", err)
	}

	backupFile := "app_2026-06-14_12-00-00.dump"
	if err := os.WriteFile(filepath.Join(backupsDir, backupFile), []byte("backup"), 0600); err != nil {
		t.Fatalf("write backup file: %v", err)
	}

	configPath := filepath.Join(tmp, "config.yaml")
	writeRestoreConfig(t, configPath, backupsDir)

	cmd := newRestoreCommand(CLIOptions{})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--config", configPath, "--file", backupFile, "--dry-run", "--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v; output = %q", err, out.String())
	}

	if got["message"] != "restore dry run passed" || got["format"] != "custom" || got["metadata_found"] != false {
		t.Fatalf("restore dry-run json = %#v, want custom fallback without metadata", got)
	}
}

func writeRestoreConfig(t *testing.T, path string, backupsDir string) {
	t.Helper()

	content := `database:
  type: postgres
  postgres:
    host: localhost
    port: 5432
    user: postgres
    name: app
    sslmode: disable
backup:
  type: full
  compression: gzip
storage:
  type: local
  local:
    path: ` + backupsDir + `
logging:
  level: info
`

	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}
