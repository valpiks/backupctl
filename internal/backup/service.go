package backup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"os"
	"strings"
	"time"

	"github.com/valpiks/backupctl/internal/compression"
	database "github.com/valpiks/backupctl/internal/dbdriver"
	"github.com/valpiks/backupctl/internal/encryption"
	"github.com/valpiks/backupctl/internal/storage"
)

type Options struct {
	DatabaseName       string
	BackupType         string
	SchemaOnly         bool
	DataOnly           bool
	Tables             []string
	Format             string
	EncryptionEnabled  bool
	EncryptionPassword string
	BackupctlVersion   string
}

type Result struct {
	FileName  string
	StartedAt time.Time
	EndedAt   time.Time
}

type Service struct {
	db         database.Driver
	storage    storage.Storage
	compressor *compression.GzipCompressor
	encryptor  encryption.Encryptor
}

type countingHashReader struct {
	reader io.Reader
	hash   hash.Hash
	size   int64
}

func NewService(db database.Driver,
	storage storage.Storage,
	compressor *compression.GzipCompressor,
	encryptor encryption.Encryptor) *Service {
	return &Service{
		db:         db,
		storage:    storage,
		compressor: compressor,
		encryptor:  encryptor,
	}
}

func (s *Service) Run(ctx context.Context, opts Options) (*Result, error) {
	startedAt := time.Now().UTC()

	backupReader, err := s.db.Backup(ctx, database.BackupOptions{
		Type:      opts.BackupType,
		ShemaOnly: opts.SchemaOnly,
		DataOnly:  opts.DataOnly,
		Tables:    opts.Tables,
		Format:    opts.Format,
	})
	if err != nil {
		return nil, err
	}

	var compressedReader io.ReadCloser
	if opts.Format == "plain" {
		compressedReader = s.compressor.Compress(backupReader)
		defer compressedReader.Close()
	} else {
		compressedReader = backupReader
	}

	backupReaderForStorage := compressedReader

	encrypted := false

	if opts.EncryptionEnabled {
		if s.encryptor == nil {
			return nil, fmt.Errorf("backup encryption is enabled but encryptor is not configured")
		}

		encryptedReader, err := s.encryptor.Encrypt(backupReaderForStorage, opts.EncryptionPassword)
		if err != nil {
			return nil, fmt.Errorf("encrypt backup: %w", err)
		}
		defer encryptedReader.Close()

		backupReaderForStorage = encryptedReader
		encrypted = true
	}

	fileName := buildBackupFileName(opts.DatabaseName, startedAt, opts.Format)
	if encrypted {
		fileName += ".enc"
	}

	countingReader := newCountingHashReader(backupReaderForStorage)

	if err := s.storage.Save(ctx, fileName, countingReader); err != nil {
		_ = backupReader.Close()
		if s.storage != nil {
			_ = s.storage.Delete(ctx, fileName)
		}
		return nil, fmt.Errorf("save backup: %w", err)
	}

	if err := backupReader.Close(); err != nil {
		if s.storage != nil {
			_ = s.storage.Delete(ctx, fileName)
		}
		return nil, fmt.Errorf("finish database backup: %w", err)
	}

	endedAt := time.Now().UTC()

	compression := ""
	if opts.Format == "plain" {
		compression = "gzip"
	}

	var encryptionMetadata *EncryptionMetadata
	if encrypted {
		encryptionMetadata = &EncryptionMetadata{
			Enabled:   true,
			Algorithm: "AES-256-GCM",
			KDF:       "argon2id",
		}
	}

	metadata := Metadata{
		DatabaseName: opts.DatabaseName,
		BackupType:   opts.BackupType,
		FileName:     fileName,
		FileSize:     countingReader.Size(),
		SHA256:       countingReader.SHA256(),
		Status:       "success",
		StartedAt:    startedAt,
		EndedAt:      endedAt,
		Duration:     endedAt.Sub(startedAt).String(),
		Format:       opts.Format,
		SchemaOnly:   opts.SchemaOnly,
		DataOnly:     opts.DataOnly,
		Tables:       opts.Tables,
		Compression:  compression,
		Encryption:   encryptionMetadata,
	}

	metadata.BackupctlVersion = opts.BackupctlVersion

	hostname, _ := os.Hostname()
	metadata.Hostname = hostname

	metadataData, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal backup metadata %w", err)
	}

	metadataFileName := buildMetadataFileName(fileName)

	if err := s.storage.Save(ctx, metadataFileName, strings.NewReader(string(metadataData))); err != nil {
		return nil, fmt.Errorf("save backup metadata %w", err)
	}

	return &Result{
		FileName:  fileName,
		StartedAt: startedAt,
		EndedAt:   endedAt,
	}, nil
}

func buildBackupFileName(databaseName string, t time.Time, format string) string {
	base := fmt.Sprintf(
		"%s_%s",
		databaseName,
		t.Format("2006-01-02_15-04-05"),
	)

	switch format {
	case "plain":
		return base + ".sql.gz"
	case "custom":
		return base + ".dump"
	default:
		return base + ".backup"
	}

}

func buildMetadataFileName(fileName string) string {
	name := strings.TrimSuffix(fileName, ".enc")
	name = strings.TrimSuffix(name, ".sql.gz")
	name = strings.TrimSuffix(name, ".sql")
	name = strings.TrimSuffix(name, ".dump")
	return name + ".metadata.json"
}

func newCountingHashReader(reader io.Reader) *countingHashReader {
	return &countingHashReader{
		reader: reader,
		hash:   sha256.New(),
	}
}

func (r *countingHashReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	if n > 0 {
		r.size += int64(n)
		_, _ = r.hash.Write(p[:n])
	}
	return n, err
}

func (r *countingHashReader) Size() int64 {
	return r.size
}

func (r *countingHashReader) SHA256() string {
	return hex.EncodeToString(r.hash.Sum(nil))
}
