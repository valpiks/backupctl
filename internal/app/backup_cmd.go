package app

import (
	"context"
	"time"

	"github.com/spf13/cobra"
	"github.com/valpiks/backupctl/internal/backup"
	"github.com/valpiks/backupctl/internal/compression"
	"github.com/valpiks/backupctl/internal/config"
	dbfactory "github.com/valpiks/backupctl/internal/database/factory"
	"github.com/valpiks/backupctl/internal/logger"
	storagefactory "github.com/valpiks/backupctl/internal/storage/factory"
)

func newBackupCommand() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Create database backup",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()

			cfg, err := config.Load(configPath)
			if err != nil {
				return err
			}

			log := logger.New(cfg.Logging.Level)
			log.Info("config loaded", "path", configPath)

			driver, err := dbfactory.NewDriver(cfg.Database)
			if err != nil {
				log.Error("database driver initialization failed", "error", err)
				return err
			}

			storage, err := storagefactory.NewStorage(cfg.Storage)
			if err != nil {
				log.Error("storage initialization failed", "error", err)
				return err
			}

			compressor := compression.NewGzipCompressor()

			service := backup.NewService(driver, storage, compressor)

			log.Info("backup started",
				"db", cfg.Database.ActiveDatabaseName(),
				"type", cfg.Backup.Type,
			)

			result, err := service.Run(ctx, backup.Options{DatabaseName: cfg.Database.ActiveDatabaseName(), BackupType: cfg.Backup.Type})
			if err != nil {
				log.Error("backup failed", "error", err)
				return err
			}

			duration := result.EndedAt.Sub(result.StartedAt)

			log.Info("backup finished",
				"file", result.FileName,
				"duration", duration.String(),
			)

			return nil
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "configs/config.yaml", "Path to config file")

	return cmd
}
