package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/valpiks/backupctl/internal/compression"
	database "github.com/valpiks/backupctl/internal/dbdriver"
	"github.com/valpiks/backupctl/internal/storage"
	"github.com/valpiks/backupctl/internal/storage/local"
)

type Options struct {
	DatabaseName string
	BackupType   string
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
		Type: opts.BackupType,
	})
	if err != nil {
		return nil, err
	}

	compressedReader := s.compressor.Compress(backupReader)

	fileName := buildBackupFileName(opts.DatabaseName, startedAt)

	localStorage, _ := s.storage.(*local.Storage)

	if err := s.storage.Save(ctx, fileName, compressedReader); err != nil {
		_ = backupReader.Close()
		if localStorage != nil {
			_ = localStorage.Delete(fileName)
		}
		return nil, fmt.Errorf("save backup: %w", err)
	}

	if err := backupReader.Close(); err != nil {
		if localStorage != nil {
			_ = localStorage.Delete(fileName)
		}
		return nil, fmt.Errorf("finish database backup: %w", err)
	}

	endedAt := time.Now().UTC()

	metadata := Metadata{
		DatabaseName: opts.DatabaseName,
		BackupType:   opts.BackupType,
		FileName:     fileName,
		Status:       "success",
		StartedAt:    startedAt,
		EndedAt:      endedAt,
		Duration:     endedAt.Sub(startedAt).String(),
	}

	metadataData, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal backup metadata %w", err)
	}

	metadataFileName := strings.TrimSuffix(fileName, ".sql.gz") + "metadata.json"

	if err := s.storage.Save(ctx, metadataFileName, strings.NewReader(string(metadataData))); err != nil {
		return nil, fmt.Errorf("save backup metadata %w", err)
	}

	return &Result{
		FileName:  fileName,
		StartedAt: startedAt,
		EndedAt:   endedAt,
	}, nil
}

func buildBackupFileName(databaseName string, t time.Time) string {
	return fmt.Sprintf(
		"%s_%s.sql.gz",
		databaseName,
		t.Format("2006-01-02_15-04-05"),
	)
}
