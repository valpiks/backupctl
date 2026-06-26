package backup

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/valpiks/backupctl/internal/compression"
	database "github.com/valpiks/backupctl/internal/dbdriver"
	"github.com/valpiks/backupctl/internal/encryption"
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
	if d.data == "" {
		return nil, fmt.Errorf("no data set for driver")
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

func (s *fakeStorage) ReadMetadata(ctx context.Context, name string) ([]byte, error) {
	b, ok := s.files[name]
	if !ok {
		return nil, os.ErrNotExist
	}
	return b, nil
}

func TestService_Run_SavesCompressedBackupAndMetadata(t *testing.T) {
	driver := &fakeDriver{
		data: "CREATE TABLE users (id int);",
	}

	st := newFakeStorage()
	compressor := compression.NewGzipCompressor()
	service := NewService(driver, st, compressor, nil)

	result, err := service.Run(context.Background(), Options{
		DatabaseName:     "testdb",
		BackupType:       "full",
		Format:           "plain",
		BackupctlVersion: "v1.0.0-test",
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

	if meta.FileSize != int64(len(backupData)) {
		t.Fatalf("metadata file_size = %d, want %d", meta.FileSize, len(backupData))
	}

	sum := sha256.Sum256(backupData)
	wantSHA256 := hex.EncodeToString(sum[:])
	if meta.SHA256 != wantSHA256 {
		t.Fatalf("metadata sha256 = %q, want %q", meta.SHA256, wantSHA256)
	}

	if meta.BackupctlVersion != "v1.0.0-test" {
		t.Fatalf("metadata backupctl_version = %q", meta.BackupctlVersion)
	}
}

func TestService_Run_SavesEncryptedBackupAndMetadata(t *testing.T) {
	driver := &fakeDriver{
		data: "CREATE TABLE users (id int);",
	}

	st := newFakeStorage()
	compressor := compression.NewGzipCompressor()
	encryptor := encryption.NewAESGCMEncryptor()
	service := NewService(driver, st, compressor, encryptor)

	result, err := service.Run(context.Background(), Options{
		DatabaseName:       "testdb",
		BackupType:         "full",
		Format:             "plain",
		EncryptionEnabled:  true,
		EncryptionPassword: "secret",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if !strings.HasSuffix(result.FileName, ".sql.gz.enc") {
		t.Fatalf("unexpected backup file name: %s", result.FileName)
	}

	backupData, ok := st.files[result.FileName]
	if !ok {
		t.Fatalf("backup file %q was not saved", result.FileName)
	}

	decrypted, err := encryptor.Decrypt(bytes.NewReader(backupData), "secret")
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}
	defer decrypted.Close()

	gz, err := gzip.NewReader(decrypted)
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

	metadataFile := strings.TrimSuffix(result.FileName, ".sql.gz.enc") + ".metadata.json"
	metadataData, ok := st.files[metadataFile]
	if !ok {
		t.Fatalf("metadata file %q was not saved", metadataFile)
	}

	var meta Metadata
	if err := json.Unmarshal(metadataData, &meta); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}

	if meta.FileName != result.FileName {
		t.Fatalf("metadata file_name = %q, want %q", meta.FileName, result.FileName)
	}

	if meta.Encryption == nil {
		t.Fatal("metadata encryption = nil")
	}

	if !meta.Encryption.Enabled {
		t.Fatal("metadata encryption enabled = false")
	}

	if meta.Encryption.Algorithm != "AES-256-GCM" {
		t.Fatalf("metadata encryption algorithm = %q", meta.Encryption.Algorithm)
	}

	if meta.Encryption.KDF != "argon2id" {
		t.Fatalf("metadata encryption kdf = %q", meta.Encryption.KDF)
	}
}

func TestService_Run_ReturnsErrorWhenStorageSaveFails(t *testing.T) {
	driver := &fakeDriver{
		data: "SELECT 1;",
	}

	st := newFakeStorage()
	st.saveErr = os.ErrNotExist

	compressor := compression.NewGzipCompressor()
	service := NewService(driver, st, compressor, nil)

	_, err := service.Run(context.Background(), Options{
		DatabaseName: "testdb",
		BackupType:   "full",
		Format:       "plain",
	})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestService_Run_ReturnsErrorWhenBackupRenderCloseFails(t *testing.T) {
	driver := &fakeDriver{
		data:     "SELECT 1;",
		closeErr: os.ErrNotExist,
	}

	st := newFakeStorage()
	compressor := compression.NewGzipCompressor()
	service := NewService(driver, st, compressor, nil)

	_, err := service.Run(context.Background(), Options{
		DatabaseName: "testdb",
		BackupType:   "full",
		Format:       "plain",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestService_Run_SavesMetadataWithFormatAndMode(t *testing.T) {
	driverPlain := &fakeDriver{
		data: "SELECT 1;",
	}

	driverCustom := &fakeDriver{
		data: "PGDUMP BINARY      ",
	}

	compressor := compression.NewGzipCompressor()

	// Тест для plain формат с schema-only
	stPlain := newFakeStorage()
	servicePlain := NewService(driverPlain, stPlain, compressor, nil)
	resultPlain, err := servicePlain.Run(context.Background(), Options{
		DatabaseName: "testdb",
		BackupType:   "full",
		SchemaOnly:   true,
		Format:       "plain",
	})
	if err != nil {
		t.Fatalf("Run(plain schema-only) error = %v", err)
	}

	metadataPlainFile := strings.TrimSuffix(resultPlain.FileName, ".sql.gz") + ".metadata.json"
	metadataPlainData, ok := stPlain.files[metadataPlainFile]
	if !ok {
		t.Fatalf("metadata file %q was not saved", metadataPlainFile)
	}

	var metadataPlain Metadata
	if err := json.Unmarshal(metadataPlainData, &metadataPlain); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}

	if metadataPlain.Format != "plain" {
		t.Fatalf("metadata format = %q, want plain", metadataPlain.Format)
	}

	if !metadataPlain.SchemaOnly {
		t.Fatalf("metadata schema_only should be true")
	}

	if metadataPlain.Compression != "gzip" {
		t.Fatalf("metadata compression = %q, want gzip (plain format is compressed)", metadataPlain.Compression)
	}

	// Тест для custom формат - используем другой сервис с другим драйвером
	stCustom := newFakeStorage()
	serviceCustom := NewService(driverCustom, stCustom, compressor, nil)
	resultCustom, err := serviceCustom.Run(context.Background(), Options{
		DatabaseName: "testdb",
		BackupType:   "full",
		Format:       "custom",
	})
	if err != nil {
		t.Logf("custom backup result: %+v", resultCustom)
		t.Logf("custom backup error: %v", err)
		t.Fatalf("Run(custom) error = %v", err)
	}

	metadataCustomFile := strings.TrimSuffix(resultCustom.FileName, ".dump") + ".metadata.json"
	t.Logf("custom filename: %s", resultCustom.FileName)
	t.Logf("metadata filename: %s", metadataCustomFile)
	metadataCustomData, ok := stCustom.files[metadataCustomFile]
	if !ok {
		t.Fatalf("metadata file %q was not saved", metadataCustomFile)
	}

	var metadataCustom Metadata
	if err := json.Unmarshal(metadataCustomData, &metadataCustom); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}

	if metadataCustom.Format != "custom" {
		t.Fatalf("metadata format = %q, want custom", metadataCustom.Format)
	}

	// Custom формат не должен быть сжат (empty compression)
	if metadataCustom.Compression != "" {
		t.Fatalf("metadata compression = %q, want empty (custom format is not compressed)", metadataCustom.Compression)
	}
}

func TestService_Run_SavesTablesList(t *testing.T) {
	driver := &fakeDriver{
		data: "SELECT 1;",
	}

	st := newFakeStorage()
	compressor := compression.NewGzipCompressor()
	service := NewService(driver, st, compressor, nil)

	result, err := service.Run(context.Background(), Options{
		DatabaseName: "testdb",
		BackupType:   "full",
		Tables:       []string{"users", "orders"},
		Format:       "plain",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	metadataFile := strings.TrimSuffix(result.FileName, ".sql.gz") + ".metadata.json"
	metadataData, ok := st.files[metadataFile]
	if !ok {
		t.Fatalf("metadata file %q was not saved", metadataFile)
	}

	var metadata Metadata
	if err := json.Unmarshal(metadataData, &metadata); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}

	if !stringsEqual(metadata.Tables, []string{"users", "orders"}) {
		t.Fatalf("metadata tables = %v, want [users orders]", metadata.Tables)
	}
}

func stringsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
