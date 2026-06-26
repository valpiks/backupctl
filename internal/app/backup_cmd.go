package app

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/valpiks/backupctl/internal/backup"
	"github.com/valpiks/backupctl/internal/compression"
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
	var verify bool
	var dryRun bool

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
			cfg, err := loadConfig(configPath)
			if err != nil {
				return err
			}

			knownSecrets := cfg.KnownSecrets()

			log := commandLogger(cfg, opts)
			log.Info("config loaded", "path", configPath)

			if dryRun {
				payload := map[string]any{
					"status":        "success",
					"dry_run":       true,
					"database":      cfg.Database.ActiveDatabaseName(),
					"database_type": cfg.Database.Type,
					"backup_type":   cfg.Backup.Type,
					"format":        format,
					"schema_only":   schemaOnly,
					"data_only":     dataOnly,
					"tables":        tables,
					"storage":       cfg.Storage.Type,
					"compression":   compressionForFormat(format),
					"encrypted":     encryptionEnabled(cfg),
				}

				if jsonOutput {
					return PrintJSON(cmd.OutOrStdout(), payload)
				}

				if !opts.Quiet {
					PrintKV(cmd.OutOrStdout(), "Backup dry run", []KV{
						{Key: "database", Value: cfg.Database.ActiveDatabaseName()},
						{Key: "database_type", Value: cfg.Database.Type},
						{Key: "backup_type", Value: cfg.Backup.Type},
						{Key: "format", Value: format},
						{Key: "storage", Value: cfg.Storage.Type},
						{Key: "compression", Value: compressionForFormat(format)},
						{Key: "encrypted", Value: YesNo(encryptionEnabled(cfg))},
					})
				}

				return nil
			}

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

			result, err := service.Run(ctx, backup.Options{DatabaseName: cfg.Database.ActiveDatabaseName(), BackupType: cfg.Backup.Type, SchemaOnly: schemaOnly, DataOnly: dataOnly, Tables: tables, Format: format, EncryptionEnabled: encryptionEnabled(cfg), EncryptionPassword: encryptionPassword(cfg), BackupctlVersion: Version})
			if err != nil {
				log.Error("backup failed", "error", secrets.Redact(err.Error(), knownSecrets))
				return redactError(err, knownSecrets)
			}

			if verify {
				decryptPassword := ""
				if encryptionEnabled(cfg) {
					decryptPassword = encryptionPassword(cfg)
				}
				if _, err := verifyBackup(ctx, storage, result.FileName, verifyOptions{DatabaseType: cfg.Database.Type, DecryptPassword: decryptPassword}); err != nil {
					log.Error("backup verification failed", "file", result.FileName, "error", secrets.Redact(err.Error(), knownSecrets))
					return redactError(fmt.Errorf("backup created but verification failed: %w", err), knownSecrets)
				}
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
				"verified":  YesNo(verify),
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
					{"verified", payload["verified"]},
				})
			}
			return nil
		},
	}

	addConfigFlag(cmd, &configPath)
	cmd.Flags().BoolVar(&schemaOnly, "schema-only", false, "Backup schema only")
	cmd.Flags().BoolVar(&dataOnly, "data-only", false, "Backup data only")
	cmd.Flags().StringSliceVar(&tables, "tables", nil, "Comma-separated list of tables")
	cmd.Flags().StringVar(&format, "format", "plain", "Backup format: plain or custom")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Print backup result as JSON")
	cmd.Flags().BoolVar(&verify, "verify", false, "Verify backup after upload")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show backup plan without creating a backup")

	_ = cmd.RegisterFlagCompletionFunc("format", func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		return []string{"plain", "custom"}, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}

func compressionForFormat(format string) string {
	if format == "plain" {
		return "gzip"
	}
	return ""
}
