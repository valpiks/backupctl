package app

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/valpiks/backupctl/internal/backup"
	"github.com/valpiks/backupctl/internal/compression"
	"github.com/valpiks/backupctl/internal/config"
	dbfactory "github.com/valpiks/backupctl/internal/database/factory"
	"github.com/valpiks/backupctl/internal/encryption"
	"github.com/valpiks/backupctl/internal/logger"
	"github.com/valpiks/backupctl/internal/secrets"
	storagefactory "github.com/valpiks/backupctl/internal/storage/factory"
)

func newBackupCommand() *cobra.Command {
	var configPath string
	var schemaOnly bool
	var dataOnly bool
	var tables []string
	var format string

	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Create database backup",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()

			if schemaOnly && dataOnly {
				return fmt.Errorf("--schema-only and --data-only cannot be used together")
			}

			if format != "plain" && format != "custom" {
				return fmt.Errorf("unsupported format: %s", format)
			}

			cfg, err := config.Load(configPath)
			if err != nil {
				return err
			}

			knownSecrets := cfg.KnownSecrets()

			log := logger.New(cfg.Logging.Level)
			log.Info("config loaded", "path", configPath)

			driver, err := dbfactory.NewDriver(cfg.Database)
			if err != nil {
				log.Error("database driver initialization failed", "error", secrets.Redact(err.Error(), knownSecrets))
				return redactError(err, knownSecrets)
			}

			storage, err := storagefactory.NewStorage(cfg.Storage)
			if err != nil {
				log.Error("storage initialization failed", "error", secrets.Redact(err.Error(), knownSecrets))
				return redactError(err, knownSecrets)
			}

			compressor := compression.NewGzipCompressor()

			var encryptor encryption.Encryptor
			if encryptionEnabled(cfg) {
				encryptor = encryption.NewAESGCMEncryptor()
			}

			service := backup.NewService(driver, storage, compressor, encryptor)

			log.Info("backup started",
				"db", cfg.Database.ActiveDatabaseName(),
				"type", cfg.Backup.Type,
			)

			result, err := service.Run(ctx, backup.Options{DatabaseName: cfg.Database.ActiveDatabaseName(), BackupType: cfg.Backup.Type, SchemaOnly: schemaOnly, DataOnly: dataOnly, Tables: tables, Format: format, EncryptionEnabled: encryptionEnabled(cfg), EncryptionPassword: encryptionPassword(cfg)})
			if err != nil {
				log.Error("backup failed", "error", secrets.Redact(err.Error(), knownSecrets))
				return redactError(err, knownSecrets)
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
	cmd.Flags().BoolVar(&schemaOnly, "schema-only", false, "Backup schema only")
	cmd.Flags().BoolVar(&dataOnly, "data-only", false, "Backup data only")
	cmd.Flags().StringSliceVar(&tables, "tables", nil, "Comma-separed list of tables")
	cmd.Flags().StringVar(&format, "format", "plain", "Backup format: plain or custom")

	return cmd
}
