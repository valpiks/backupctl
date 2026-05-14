package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/valpiks/backupctl/internal/compression"
	database "github.com/valpiks/backupctl/internal/dbdriver"
	"github.com/valpiks/backupctl/internal/storage"
)

type Options struct {
	DatabaseName string
	BackupType   string
	SchemaOnly   bool
	DataOnly     bool
	Tables       []string
	Format       string
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
}

func NewService(db database.Driver,
	storage storage.Storage,
	compressor *compression.GzipCompressor) *Service {
	return &Service{
		db:         db,
		storage:    storage,
		compressor: compressor,
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

	fileName := buildBackupFileName(opts.DatabaseName, startedAt, opts.Format)

	if err := s.storage.Save(ctx, fileName, compressedReader); err != nil {
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

	metadata := Metadata{
		DatabaseName: opts.DatabaseName,
		BackupType:   opts.BackupType,
		FileName:     fileName,
		Status:       "success",
		StartedAt:    startedAt,
		EndedAt:      endedAt,
		Duration:     endedAt.Sub(startedAt).String(),
		Format:       opts.Format,
		SchemaOnly:   opts.SchemaOnly,
		DataOnly:     opts.DataOnly,
		Tabels:       opts.Tables,
		Compression:  compression,
	}

	metadataData, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal backup metadata %w", err)
	}

	metadataFileName := strings.TrimSuffix(fileName, ".sql.gz")
	if !strings.HasSuffix(metadataFileName, ".dump") {
		metadataFileName = strings.TrimSuffix(metadataFileName, ".sql") + ".metadata.json"
	} else {
		metadataFileName = strings.TrimSuffix(metadataFileName, ".dump") + ".metadata.json"
	}

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
