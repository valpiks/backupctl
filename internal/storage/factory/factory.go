package factory

import (
	"fmt"

	"github.com/valpiks/backupctl/internal/config"
	"github.com/valpiks/backupctl/internal/storage"
	"github.com/valpiks/backupctl/internal/storage/local"
)

func NewStorage(cfg config.StorageConfig) (storage.Storage, error) {
	switch cfg.Type {
	case "local":
		return local.NewStorage(cfg.Path)
	default:
		return nil, fmt.Errorf("unsupported storage format: %s", cfg.Type)

	}
}
