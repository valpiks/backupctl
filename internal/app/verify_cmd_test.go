package app

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/valpiks/backupctl/internal/backup"
	"github.com/valpiks/backupctl/internal/storage"
)

type fakeVerifyStorage struct {
	files map[string][]byte
}

func newFakeVerifyStorage() *fakeVerifyStorage {
	return &fakeVerifyStorage{files: make(map[string][]byte)}
}

func (s *fakeVerifyStorage) Save(ctx context.Context, name string, data io.Reader) error {
	b, err := io.ReadAll(data)
	if err != nil {
		return err
	}
	s.files[name] = b
	return nil
}

func (s *fakeVerifyStorage) Open(ctx context.Context, name string) (io.ReadCloser, error) {
	b, ok := s.files[name]
	if !ok {
		return nil, os.ErrNotExist
	}
	return io.NopCloser(bytes.NewReader(b)), nil
}

func (s *fakeVerifyStorage) List(ctx context.Context) ([]storage.BackupFile, error) {
	files := make([]storage.BackupFile, 0, len(s.files))
	for name, data := range s.files {
		files = append(files, storage.BackupFile{Name: name, Size: int64(len(data))})
	}
	return files, nil
}

func (s *fakeVerifyStorage) Delete(ctx context.Context, name string) error {
	delete(s.files, name)
	return nil
}

func (s *fakeVerifyStorage) ReadMetadata(ctx context.Context, name string) ([]byte, error) {
	b, ok := s.files[name]
	if !ok {
		return nil, os.ErrNotExist
	}
	return b, nil
}

func TestVerifyBackupSucceeds(t *testing.T) {
	st := newFakeVerifyStorage()
	fileName := "app_2026-06-26_12-00-00.sql.gz"
	body := []byte("backup-data")
	st.files[fileName] = body
	st.files[metadataNameForBackup(fileName)] = marshalVerifyMetadata(t, fileName, body)

	result, err := verifyBackup(context.Background(), st, fileName)
	if err != nil {
		t.Fatalf("verifyBackup() error = %v", err)
	}

	if result.Size != int64(len(body)) {
		t.Fatalf("size = %d, want %d", result.Size, len(body))
	}
}

func TestVerifyBackupFailsOnChecksumMismatch(t *testing.T) {
	st := newFakeVerifyStorage()
	fileName := "app_2026-06-26_12-00-00.sql.gz"
	st.files[fileName] = []byte("changed-data")
	st.files[metadataNameForBackup(fileName)] = marshalVerifyMetadata(t, fileName, []byte("original-data"))

	_, err := verifyBackup(context.Background(), st, fileName)
	if err == nil {
		t.Fatal("expected checksum mismatch error, got nil")
	}

	if !strings.Contains(err.Error(), "sha256 mismatch") && !strings.Contains(err.Error(), "file size mismatch") {
		t.Fatalf("error = %q, want verification mismatch", err.Error())
	}
}

func TestVerifyBackupDeepChecksGzip(t *testing.T) {
	st := newFakeVerifyStorage()
	fileName := "app_2026-06-26_12-00-00.sql.gz"

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write([]byte("SELECT 1;")); err != nil {
		t.Fatalf("gzip write: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}

	body := buf.Bytes()
	st.files[fileName] = body
	st.files[metadataNameForBackup(fileName)] = marshalVerifyMetadataWithCompression(t, fileName, body, "gzip")

	if _, err := verifyBackup(context.Background(), st, fileName, verifyOptions{Deep: true}); err != nil {
		t.Fatalf("verifyBackup(deep) error = %v", err)
	}
}

func marshalVerifyMetadata(t *testing.T, fileName string, body []byte) []byte {
	t.Helper()

	return marshalVerifyMetadataWithCompression(t, fileName, body, "")
}

func marshalVerifyMetadataWithCompression(t *testing.T, fileName string, body []byte, compression string) []byte {
	t.Helper()

	sum := sha256.Sum256(body)
	data, err := json.Marshal(backup.Metadata{
		FileName:    fileName,
		FileSize:    int64(len(body)),
		SHA256:      hex.EncodeToString(sum[:]),
		Compression: compression,
	})
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}

	return data
}
