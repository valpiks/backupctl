package backup

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/valpiks/backupctl/internal/compression"
	database "github.com/valpiks/backupctl/internal/dbdriver"
	"github.com/valpiks/backupctl/internal/storage"
)

type fakeStorage struct {
	files   map[string][]byte
	saveErr error
}

type fakeDriver struct {
	data      string
	backupErr error
	closeErr  error
}

type fakeReadCloser struct {
	io.Reader
	closeErr error
}

func (d *fakeDriver) Ping(ctx context.Context) error {
	return nil
}

func (d *fakeDriver) Backup(ctx context.Context, opts database.BackupOptions) (io.ReadCloser, error) {
	if d.backupErr != nil {
		return nil, d.backupErr
	}
	return &fakeReadCloser{
		Reader:   strings.NewReader(d.data),
		closeErr: d.closeErr,
	}, nil
}

func (d *fakeDriver) Restore(ctx context.Context, input io.Reader, opts database.RestoreOptions) error {
	return nil
}

func (r *fakeReadCloser) Close() error {
	return r.closeErr
}

func newFakeStorage() *fakeStorage {
	return &fakeStorage{
		files: make(map[string][]byte),
	}
}

func (s *fakeStorage) Save(ctx context.Context, name string, data io.Reader) error {
	if s.saveErr != nil {
		return s.saveErr
	}

	b, err := io.ReadAll(data)
	if err != nil {
		return err
	}

	s.files[name] = b
	return nil
}

func (s *fakeStorage) Open(ctx context.Context, name string) (io.ReadCloser, error) {
	b, ok := s.files[name]
	if !ok {
		return nil, os.ErrNotExist
	}

	return io.NopCloser(bytes.NewReader(b)), nil
}

func (s *fakeStorage) List(ctx context.Context) ([]storage.BackupFile, error) {
	var out []storage.BackupFile
	for name, b := range s.files {
		out = append(out, storage.BackupFile{
			Name: name,
			Size: int64(len(b)),
		})
	}
	return out, nil
}

func (s *fakeStorage) Delete(ctx context.Context, name string) error {
	return nil
}

func TestSerivice_Run_SavesCompressedBackupAndMetadata(t *testing.T) {
	driver := &fakeDriver{
		data: "CREATE TABLE users (id int);",
	}

	st := newFakeStorage()
	compressor := compression.NewGzipCompressor()
	service := NewService(driver, st, compressor)

	result, err := service.Run(context.Background(), Options{
		DatabaseName: "testdb",
		BackupType:   "full",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if result == nil {
		t.Fatalf("Run() returned nil result")
	}

	if !strings.HasSuffix(result.FileName, ".sql.gz") {
		t.Fatalf("unexpected backup file name: %s", result.FileName)
	}

	backupData, ok := st.files[result.FileName]
	if !ok {
		t.Fatalf("backup file %q was not saved", result.FileName)
	}

	gz, err := gzip.NewReader(bytes.NewReader(backupData))
	if err != nil {
		t.Fatalf("gzip reader: %v", err)
	}
	defer gz.Close()

	sqlData, err := io.ReadAll(gz)
	if err != nil {
		t.Fatalf("read gzip data: %v", err)
	}

	if string(sqlData) != "CREATE TABLE users (id int);" {
		t.Fatalf("unexpected backup contents: %q", string(sqlData))
	}

	metadataFile := strings.TrimSuffix(result.FileName, ".sql.gz") + ".metadata.json"

	metadataData, ok := st.files[metadataFile]
	if !ok {
		t.Fatalf("metadata file %q was not saved", metadataFile)
	}

	var meta Metadata
	if err := json.Unmarshal(metadataData, &meta); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}

	if meta.DatabaseName != "testdb" {
		t.Fatalf("metadata database_name = %q", meta.DatabaseName)
	}

	if meta.BackupType != "full" {
		t.Fatalf("metadata backup_type = %q", meta.BackupType)
	}

	if meta.FileName != result.FileName {
		t.Fatalf("metadata file_name = %q, want %q", meta.FileName, result.FileName)
	}

	if meta.Status != "success" {
		t.Fatalf("metadata status = %q", meta.Status)
	}
}

func TestService_Run_ReturnsErrorWhenStorageSaveFails(t *testing.T) {
	driver := &fakeDriver{
		data: "SELECT 1;",
	}

	st := newFakeStorage()
	st.saveErr = errors.New("save failed")

	compressor := compression.NewGzipCompressor()
	service := NewService(driver, st, compressor)

	_, err := service.Run(context.Background(), Options{
		DatabaseName: "testdb",
		BackupType:   "full",
	})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestService_Run_ReturnsErrorWhenBackupRenderCloseFails(t *testing.T) {
	driver := &fakeDriver{
		data:     "SELECT 1;",
		closeErr: errors.New("close failed"),
	}

	st := newFakeStorage()
	compressor := compression.NewGzipCompressor()
	service := NewService(driver, st, compressor)

	_, err := service.Run(context.Background(), Options{
		DatabaseName: "testdb",
		BackupType:   "full",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
