package app

import (
	"context"
	"time"

	"github.com/spf13/cobra"
	"github.com/valpiks/dbbackup/internal/backup"
	"github.com/valpiks/dbbackup/internal/compression"
	"github.com/valpiks/dbbackup/internal/config"
	"github.com/valpiks/dbbackup/internal/database/postgres"
	"github.com/valpiks/dbbackup/internal/logger"
	"github.com/valpiks/dbbackup/internal/storage/local"
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

			driver, err := postgres.NewDriver(cfg.Database)
			if err != nil {
				return err
			}

			storage, err := local.NewStorage(cfg.Storage.Path)
			if err != nil {
				return err
			}

			compressor := compression.NewGzipCompressor()

			service := backup.NewService(driver, storage, compressor)

			log.Info("backup started",
				"db", cfg.Database.Name,
				"type", cfg.Backup.Type,
			)

			result, err := service.Run(ctx, backup.Options{DatabaseName: cfg.Database.Name, BackupType: cfg.Backup.Type})
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
