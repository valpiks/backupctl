package factory

import (
	"fmt"

	"github.com/valpiks/backupctl/internal/config"
	"github.com/valpiks/backupctl/internal/database/mongo"
	"github.com/valpiks/backupctl/internal/database/postgres"
	database "github.com/valpiks/backupctl/internal/dbdriver"
)

func NewDriver(cfg config.DatabaseConfig) (database.Driver, error) {
	switch cfg.Type {
	case "postgres":
		return postgres.NewDriver(cfg)
	case "mongo":
		return mongo.NewDriver(cfg)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.Type)
	}
}
