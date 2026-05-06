package factory

import (
	"fmt"

	"github.com/valpiks/backupctl/internal/config"
	"github.com/valpiks/backupctl/internal/storage"
	"github.com/valpiks/backupctl/internal/storage/local"
	"github.com/valpiks/backupctl/internal/storage/s3"
)

func NewStorage(cfg config.StorageConfig) (storage.Storage, error) {
	switch cfg.Type {
	case "local":
		return local.NewStorage(cfg.Local.Path)
	case "s3":
		return s3.NewStorage(s3.Config{
			Bucket:         cfg.S3.Bucket,
			Region:         cfg.S3.Region,
			Prefix:         cfg.S3.Prefix,
			Endpoint:       cfg.S3.Endpoint,
			ForcePathStyle: cfg.S3.ForcePathStyle,
		})
	default:
		return nil, fmt.Errorf("unsupported storage format: %s", cfg.Type)

	}
}
