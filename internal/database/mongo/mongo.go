package mongo

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/valpiks/backupctl/internal/config"
)

type Driver struct {
	cfg config.DatabaseConfig
}

func NewDriver(cfg config.DatabaseConfig) (*Driver, error) {
	if cfg.Type != "mongo" {
		return nil, fmt.Errorf("unsupported database type for mongo driver: %s", cfg.Type)
	}

	if cfg.Mongo.URI == "" {
		return nil, fmt.Errorf("mongo.uri is required")
	}

	if cfg.Mongo.Database == "" {
		return nil, fmt.Errorf("mongo.database is required")
	}

	return &Driver{
		cfg: cfg,
	}, nil
}

func (d *Driver) Ping(ctx context.Context) error {
	args := []string{
		d.cfg.Mongo.URI,
		"--eval", "db.adminCommand('ping')",
	}

	cmd := exec.CommandContext(ctx, "mongosh", args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mongo connection failed: %w: %s", err, string(output))
	}

	return nil
}
