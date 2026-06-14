package app

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/valpiks/backupctl/internal/backup"
	"github.com/valpiks/backupctl/internal/compression"
	"github.com/valpiks/backupctl/internal/config"
	dbfactory "github.com/valpiks/backupctl/internal/database/factory"
	database "github.com/valpiks/backupctl/internal/dbdriver"
	"github.com/valpiks/backupctl/internal/encryption"
	"github.com/valpiks/backupctl/internal/secrets"
	storagefactory "github.com/valpiks/backupctl/internal/storage/factory"
)

func newRestoreCommand(opts CLIOptions) *cobra.Command {
	var configPath string
	var fileName string
	var targetDB string
	var yes bool
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "restore --file <backup-file>",
		Short: "Restore database from backup",
		Long:  "Restore a PostgreSQL or MongoDB database from a backup file stored in configured storage.",
		Example: `  backupctl restore -c configs/config.yaml --file app_20260607_120000.sql.gz
  backupctl restore -c configs/config.yaml --file app_20260607_120000.sql.gz --target-db app_restore
  backupctl restore -c configs/config.yaml --file app_20260607_120000.sql.gz --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if fileName == "" {
				return WithHint(fmt.Errorf("--file is required"), "pass --file <backup-file> or run backupctl list --files")
			}

			if jsonOutput && !yes {
				return WithHint(fmt.Errorf("--json restore requires --yes"), "rerun with --yes after confirming the restore target")
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()

			cfg, err := config.Load(configPath)
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

			if !yes {
				confirmed, err := confirmRestore(cmd.InOrStdin(), cmd.OutOrStdout(), fileName, restoreDB)
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

			reader, err := storage.Open(ctx, fileName)
			if err != nil {
				log.Error("open backup failed", "file", fileName, "error", secrets.Redact(err.Error(), knownSecrets))
				return redactError(err, knownSecrets)
			}
			defer reader.Close()

			var format string
			var compressionFlag string
			encrypted := strings.HasSuffix(fileName, ".enc")
			var metadata backup.Metadata
			metadataData, err := storage.ReadMetadata(ctx, fileName)
			if err != nil {
				format = detectFormatFromName(fileName)
			} else {
				if err := json.Unmarshal(metadataData, &metadata); err == nil {
					format = metadata.Format
					compressionFlag = metadata.Compression
					encrypted = metadata.Encryption != nil && metadata.Encryption.Enabled
				} else {
					format = detectFormatFromName(fileName)
				}
			}

			if encrypted {
				if !encryptionEnabled(cfg) {
					return WithHint(
						fmt.Errorf("backup is encrypted but encryption is disabled in config"),
						"set backup.encryption.enabled=true and provide backup.encryption.password_env",
					)
				}

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
			if compressionFlag == "gzip" || strings.HasSuffix(plainFileName, ".sql.gz") {
				compressor := compression.NewGzipCompressor()

				decompressionReader, err := compressor.Decompress(reader)
				if err != nil {
					log.Error("decompress backup failed", "file", fileName, "error", secrets.Redact(err.Error(), knownSecrets))
					return redactError(err, knownSecrets)
				}
				defer decompressionReader.Close()
				reader = decompressionReader
			}

			err = driver.Restore(ctx, reader, database.RestoreOptions{TargetDatabase: restoreDB, Format: format})
			if err != nil {
				log.Error("restore failed", "file", fileName, "db", restoreDB, "error", secrets.Redact(err.Error(), knownSecrets))
				return redactError(err, knownSecrets)
			}

			log.Info("restore finished", "file", fileName, "db", restoreDB)
			payload := map[string]string{
				"status":    "success",
				"message":   "restore completed",
				"file":      fileName,
				"database":  restoreDB,
				"format":    format,
				"encrypted": YesNo(encrypted),
			}

			if jsonOutput {
				return PrintJSON(cmd.OutOrStdout(), payload)
			}

			if !opts.Quiet {
				PrintKV(cmd.OutOrStdout(), "Restore completed", []KV{
					{"file", payload["file"]},
					{"database", payload["database"]},
					{"format", payload["format"]},
					{"encrypted", payload["encrypted"]},
				})
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "configs/config.yaml", "Path to config file")
	cmd.Flags().StringVar(&fileName, "file", "", "Backup file name")
	cmd.Flags().StringVar(&targetDB, "target-db", "", "Target database name for restore")
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Print restore result as JSON")

	_ = cmd.MarkFlagRequired("file")

	return cmd
}

func confirmRestore(in io.Reader, out io.Writer, fileName string, dbName string) (bool, error) {
	fmt.Fprintf(
		out,
		"You are about to restore database %q from:\n  %s\n",
		dbName,
		fileName,
	)
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
