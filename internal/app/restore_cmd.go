package app

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/valpiks/backupctl/internal/backup"
	"github.com/valpiks/backupctl/internal/compression"
	dbfactory "github.com/valpiks/backupctl/internal/database/factory"
	database "github.com/valpiks/backupctl/internal/dbdriver"
	"github.com/valpiks/backupctl/internal/encryption"
	"github.com/valpiks/backupctl/internal/secrets"
	storagepkg "github.com/valpiks/backupctl/internal/storage"
	storagefactory "github.com/valpiks/backupctl/internal/storage/factory"
)

type restorePlan struct {
	FileName      string
	SourceDB      string
	TargetDB      string
	Format        string
	Compression   string
	Encrypted     bool
	MetadataFound bool
}

func newRestoreCommand(opts CLIOptions) *cobra.Command {
	var configPath string
	var fileName string
	var targetDB string
	var yes bool
	var jsonOutput bool
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "restore --file <backup-file>",
		Short: "Restore database from backup",
		Long:  "Restore a PostgreSQL or MongoDB database from a backup file stored in configured storage.",
		Example: `  backupctl restore -c configs/config.yaml --file app_20260607_120000.sql.gz
  backupctl restore -c configs/config.yaml --file app_20260607_120000.sql.gz --dry-run
  backupctl restore -c configs/config.yaml --file app_20260607_120000.sql.gz --target-db app_restore
  backupctl restore -c configs/config.yaml --file app_20260607_120000.sql.gz --yes
  backupctl restore -c configs/config.yaml --file app_20260607_120000.sql.gz --yes --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if fileName == "" {
				return WithHint(fmt.Errorf("--file is required"), "pass --file <backup-file> or run backupctl list --files")
			}

			if jsonOutput && !yes && !dryRun {
				return WithHint(fmt.Errorf("--json restore requires --yes"), "rerun with --yes after confirming the restore target")
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			cfg, err := loadConfig(configPath)
			if err != nil {
				return err
			}

			knownSecrets := cfg.KnownSecrets()

			restoreDB := cfg.Database.ActiveDatabaseName()
			if targetDB != "" {
				restoreDB = targetDB
			}

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

			plan, err := buildRestorePlan(ctx, storage, fileName, restoreDB)
			if err != nil {
				log.Error("build restore plan failed", "file", fileName, "error", secrets.Redact(err.Error(), knownSecrets))
				return redactError(err, knownSecrets)
			}

			if plan.Encrypted && !encryptionEnabled(cfg) {
				return WithHint(
					fmt.Errorf("backup is encrypted but encryption is disabled in config"),
					"set backup.encryption.enabled=true and provide backup.encryption.password_env",
				)
			}

			if dryRun {
				payload := restorePlanPayload(plan, true)

				if jsonOutput {
					return PrintJSON(cmd.OutOrStdout(), payload)
				}

				if !opts.Quiet {
					PrintKV(cmd.OutOrStdout(), "Restore dry run passed", []KV{
						{Key: "file", Value: plan.FileName},
						{Key: "source", Value: valueOrUnknown(plan.SourceDB)},
						{Key: "target", Value: plan.TargetDB},
						{Key: "format", Value: plan.Format},
						{Key: "encrypted", Value: YesNo(plan.Encrypted)},
						{Key: "metadata", Value: metadataStatus(plan.MetadataFound)},
					})
				}

				return nil
			}

			if !yes {
				if err := requireInteractiveRestore(cmd, yes); err != nil {
					return err
				}

				confirmed, err := confirmRestore(cmd.InOrStdin(), cmd.OutOrStdout(), plan)
				if err != nil {
					log.Error("restore confirmation failed", "file", fileName, "error", err)
					return err
				}

				if !confirmed {
					log.Info("restore cancelled", "file", fileName, "db", restoreDB)
					fmt.Fprintln(cmd.OutOrStdout(), "restore cancelled")
					return nil
				}
			}

			log.Info("restore started", "file", fileName, "db", restoreDB)

			reader, err := storage.Open(ctx, fileName)
			if err != nil {
				log.Error("open backup failed", "file", fileName, "error", secrets.Redact(err.Error(), knownSecrets))
				return redactError(err, knownSecrets)
			}
			defer reader.Close()

			if plan.Encrypted {
				decryptor := encryption.NewAESGCMEncryptor()

				decryptedReader, err := decryptor.Decrypt(reader, encryptionPassword(cfg))
				if err != nil {
					log.Error("decrypt backup failed", "file", fileName, "error", secrets.Redact(err.Error(), knownSecrets))
					return redactError(err, knownSecrets)
				}
				defer decryptedReader.Close()
				reader = decryptedReader
			}

			plainFileName := backupNameWithoutEncryptionSuffix(fileName)
			if plan.Compression == "gzip" || strings.HasSuffix(plainFileName, ".sql.gz") {
				compressor := compression.NewGzipCompressor()

				decompressionReader, err := compressor.Decompress(reader)
				if err != nil {
					log.Error("decompress backup failed", "file", fileName, "error", secrets.Redact(err.Error(), knownSecrets))
					return redactError(err, knownSecrets)
				}
				defer decompressionReader.Close()
				reader = decompressionReader
			}

			err = driver.Restore(ctx, reader, database.RestoreOptions{TargetDatabase: restoreDB, Format: plan.Format})
			if err != nil {
				log.Error("restore failed", "file", fileName, "db", restoreDB, "error", secrets.Redact(err.Error(), knownSecrets))
				return redactError(err, knownSecrets)
			}

			log.Info("restore finished", "file", fileName, "db", restoreDB)
			payload := restorePlanPayload(plan, false)

			if jsonOutput {
				return PrintJSON(cmd.OutOrStdout(), payload)
			}

			if !opts.Quiet {
				PrintKV(cmd.OutOrStdout(), "Restore completed", []KV{
					{Key: "file", Value: plan.FileName},
					{Key: "source", Value: valueOrUnknown(plan.SourceDB)},
					{Key: "target", Value: plan.TargetDB},
					{Key: "format", Value: plan.Format},
					{Key: "encrypted", Value: YesNo(plan.Encrypted)},
					{Key: "metadata", Value: metadataStatus(plan.MetadataFound)},
				})
			}
			return nil
		},
	}

	addConfigFlag(cmd, &configPath)
	cmd.Flags().StringVar(&fileName, "file", "", "Backup file name")
	cmd.Flags().StringVar(&targetDB, "target-db", "", "Target database name for restore")
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Print restore result as JSON")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Validate restore inputs without modifying the database")

	_ = cmd.MarkFlagRequired("file")

	return cmd
}

func confirmRestore(in io.Reader, out io.Writer, plan *restorePlan) (bool, error) {
	PrintKV(out, "You are about to restore:", []KV{
		{Key: "file", Value: plan.FileName},
		{Key: "source", Value: valueOrUnknown(plan.SourceDB)},
		{Key: "target", Value: plan.TargetDB},
		{Key: "format", Value: plan.Format},
		{Key: "encrypted", Value: YesNo(plan.Encrypted)},
		{Key: "metadata", Value: metadataStatus(plan.MetadataFound)},
	})

	fmt.Fprintln(out)
	fmt.Fprint(out, "This may overwrite existing data. Continue? [y/N]: ")

	reader := bufio.NewReader(in)
	answer, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("read confirmation: %w", err)
	}

	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes", nil
}

func detectFormatFromName(fileName string) string {
	fileName = backupNameWithoutEncryptionSuffix(fileName)

	switch {
	case strings.HasSuffix(fileName, ".dump"):
		return "custom"
	case strings.HasSuffix(fileName, ".sql.gz"), strings.HasSuffix(fileName, ".sql"):
		return "plain"
	default:
		return "plain"
	}
}

func backupNameWithoutEncryptionSuffix(fileName string) string {
	return strings.TrimSuffix(fileName, ".enc")
}

func restorePlanPayload(plan *restorePlan, dryRun bool) map[string]any {
	return map[string]any{
		"status":          "success",
		"message":         restorePlanMessage(dryRun),
		"dry_run":         dryRun,
		"file":            plan.FileName,
		"source_database": plan.SourceDB,
		"target_database": plan.TargetDB,
		"format":          plan.Format,
		"compression":     plan.Compression,
		"encrypted":       plan.Encrypted,
		"metadata_found":  plan.MetadataFound,
	}
}

func restorePlanMessage(dryRun bool) string {
	if dryRun {
		return "restore dry run passed"
	}
	return "restore completed"
}

func valueOrUnknown(value string) string {
	if value == "" {
		return "unknown"
	}
	return value
}

func metadataStatus(found bool) string {
	if found {
		return "found"
	}
	return "missing"
}

func buildRestorePlan(ctx context.Context, storage storagepkg.Storage, fileName string, targetDB string) (*restorePlan, error) {
	reader, err := storage.Open(ctx, fileName)
	if err != nil {
		return nil, WithHint(
			fmt.Errorf("backup file not found: %s", fileName),
			"run backupctl list --files to see available files",
		)
	}
	_ = reader.Close()

	plan := &restorePlan{
		FileName:  fileName,
		TargetDB:  targetDB,
		Format:    detectFormatFromName(fileName),
		Encrypted: strings.HasSuffix(fileName, ".enc"),
	}

	metadataData, err := storage.ReadMetadata(ctx, metadataNameForBackup(fileName))
	if err != nil {
		return plan, nil
	}

	var metadata backup.Metadata
	if err := json.Unmarshal(metadataData, &metadata); err != nil {
		return nil, fmt.Errorf("parse metadata for %s: %w", fileName, err)
	}

	plan.MetadataFound = true
	plan.SourceDB = metadata.DatabaseName

	if metadata.Format != "" {
		plan.Format = metadata.Format
	}

	if metadata.Compression != "" {
		plan.Compression = metadata.Compression
	}

	if metadata.Encryption != nil {
		plan.Encrypted = metadata.Encryption.Enabled
	}

	return plan, nil
}

func requireInteractiveRestore(cmd *cobra.Command, yes bool) error {
	if yes {
		return nil
	}

	in := cmd.InOrStdin()
	file, ok := in.(*os.File)
	if !ok {
		return WithHint(
			fmt.Errorf("restore confirmation requires interactive input"),
			"pass --yes to confirm restore explicitly",
		)
	}

	stat, err := file.Stat()
	if err != nil {
		return WithHint(
			fmt.Errorf("restore confirmation requires interactive input"),
			"pass --yes to confirm restore explicitly",
		)
	}

	if stat.Mode()&os.ModeCharDevice == 0 {
		return WithHint(
			fmt.Errorf("restore confirmation requires interactive terminal"),
			"pass --yes to confirm restore explicitly",
		)
	}

	return nil
}
