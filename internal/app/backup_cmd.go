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
	"github.com/valpiks/backupctl/internal/secrets"
	storagefactory "github.com/valpiks/backupctl/internal/storage/factory"
)

func newBackupCommand(opts CLIOptions) *cobra.Command {
	var configPath string
	var schemaOnly bool
	var dataOnly bool
	var tables []string
	var format string
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Create database backup",
		Long:  "Create a database backup using the configured database and storage drivers.",
		Example: `  backupctl backup -c configs/config.yaml
  backupctl backup -c configs/config.yaml --format custom
  backupctl backup -c configs/config.yaml --schema-only
  backupctl backup -c configs/config.yaml --tables users,orders`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()

			if schemaOnly && dataOnly {
				return WithHint(fmt.Errorf("--schema-only and --data-only cannot be used together"), "choose only one backup mode")
			}

			if format != "plain" && format != "custom" {
				return WithHint(fmt.Errorf("unsupported format: %s", format), "use --format plain or --format custom")
			}

			cfg, err := config.Load(configPath)
			if err != nil {
				return err
			}

			knownSecrets := cfg.KnownSecrets()

			log := commandLogger(cfg, opts)
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
			payload := map[string]string{
				"status":    "success",
				"message":   "backup completed",
				"file":      result.FileName,
				"database":  cfg.Database.ActiveDatabaseName(),
				"type":      cfg.Backup.Type,
				"format":    format,
				"duration":  HumanDuration(duration),
				"encrypted": YesNo(encryptionEnabled(cfg)),
			}

			if jsonOutput {
				return PrintJSON(cmd.OutOrStdout(), payload)
			}

			if !opts.Quiet {
				PrintKV(cmd.OutOrStdout(), "Backup completed", []KV{
					{"file", payload["file"]},
					{"database", payload["database"]},
					{"type", payload["type"]},
					{"format", payload["format"]},
					{"duration", payload["duration"]},
					{"encrypted", payload["encrypted"]},
				})
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "configs/config.yaml", "Path to config file")
	cmd.Flags().BoolVar(&schemaOnly, "schema-only", false, "Backup schema only")
	cmd.Flags().BoolVar(&dataOnly, "data-only", false, "Backup data only")
	cmd.Flags().StringSliceVar(&tables, "tables", nil, "Comma-separated list of tables")
	cmd.Flags().StringVar(&format, "format", "plain", "Backup format: plain or custom")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Print backup result as JSON")

	_ = cmd.RegisterFlagCompletionFunc("format", func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		return []string{"plain", "custom"}, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}
