package app

import (
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/valpiks/backupctl/internal/backup"
	"github.com/valpiks/backupctl/internal/encryption"
	storagepkg "github.com/valpiks/backupctl/internal/storage"
	storagefactory "github.com/valpiks/backupctl/internal/storage/factory"
)

type verifyResult struct {
	FileName       string
	Size           int64
	ExpectedSize   int64
	SHA256         string
	ExpectedSHA256 string
}

type verifyOptions struct {
	DatabaseType    string
	DecryptPassword string
	Deep            bool
}

func newVerifyCommand(opts CLIOptions) *cobra.Command {
	var configPath string
	var fileName string
	var jsonOutput bool
	var deep bool

	cmd := &cobra.Command{
		Use:   "verify --file <backup-file>",
		Short: "Verify backup file against metadata",
		RunE: func(cmd *cobra.Command, args []string) error {
			if fileName == "" {
				return WithHint(fmt.Errorf("--file is required"), "pass --file <backup-file>")
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()

			cfg, err := loadConfig(configPath)
			if err != nil {
				return err
			}

			storage, err := storagefactory.NewStorage(cfg.Storage)
			if err != nil {
				return err
			}

			decryptPassword := ""
			if encryptionEnabled(cfg) {
				decryptPassword = encryptionPassword(cfg)
			}

			result, err := verifyBackup(ctx, storage, fileName, verifyOptions{
				DatabaseType:    cfg.Database.Type,
				DecryptPassword: decryptPassword,
				Deep:            deep,
			})
			ok := err == nil

			payload := map[string]any{
				"status":          statusFromBool(ok),
				"deep":            deep,
				"file":            fileName,
				"size":            result.Size,
				"expected_size":   result.ExpectedSize,
				"sha256":          result.SHA256,
				"expected_sha256": result.ExpectedSHA256,
			}

			if jsonOutput {
				if printErr := PrintJSON(cmd.OutOrStdout(), payload); printErr != nil {
					return printErr
				}
				return err
			}

			if !ok {
				return err
			}

			if !opts.Quiet {
				PrintKV(cmd.OutOrStdout(), "Backup verified", []KV{
					{Key: "file", Value: fileName},
					{Key: "size", Value: fmt.Sprintf("%d", result.Size)},
					{Key: "sha256", Value: result.SHA256},
				})
			}

			return nil
		},
	}

	addConfigFlag(cmd, &configPath)
	cmd.Flags().StringVar(&fileName, "file", "", "Backup file name")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Print result as JSON")
	cmd.Flags().BoolVar(&deep, "deep", false, "Run format-level verification")

	return cmd
}

func statusFromBool(ok bool) string {
	if ok {
		return "success"
	}
	return "failed"
}

func verifyBackup(ctx context.Context, storage storagepkg.Storage, fileName string, opts ...verifyOptions) (*verifyResult, error) {
	var options verifyOptions
	if len(opts) > 0 {
		options = opts[0]
	}

	result := &verifyResult{FileName: fileName}

	metaData, err := storage.ReadMetadata(ctx, metadataNameForBackup(fileName))
	if err != nil {
		return result, fmt.Errorf("read metadata: %w", err)
	}

	var meta backup.Metadata
	if err := json.Unmarshal(metaData, &meta); err != nil {
		return result, fmt.Errorf("parse metadata: %w", err)
	}

	result.ExpectedSize = meta.FileSize
	result.ExpectedSHA256 = meta.SHA256

	reader, err := storage.Open(ctx, fileName)
	if err != nil {
		return result, fmt.Errorf("open backup: %w", err)
	}
	defer reader.Close()

	hash := sha256.New()
	size, err := io.Copy(hash, reader)
	if err != nil {
		return result, fmt.Errorf("read backup: %w", err)
	}

	result.Size = size
	result.SHA256 = hex.EncodeToString(hash.Sum(nil))

	if result.Size != result.ExpectedSize {
		return result, fmt.Errorf("backup verification failed: file size mismatch")
	}

	if result.SHA256 != result.ExpectedSHA256 {
		return result, fmt.Errorf("backup verification failed: sha256 mismatch")
	}

	if options.Deep {
		if err := verifyBackupFormat(ctx, storage, fileName, meta, options); err != nil {
			return result, err
		}
	}

	return result, nil
}

func verifyBackupFormat(ctx context.Context, storage storagepkg.Storage, fileName string, meta backup.Metadata, options verifyOptions) error {
	reader, err := storage.Open(ctx, fileName)
	if err != nil {
		return fmt.Errorf("open backup for format verification: %w", err)
	}
	defer reader.Close()

	plainName := backupNameWithoutEncryptionSuffix(fileName)
	var formatReader io.Reader = reader

	if meta.Encryption != nil && meta.Encryption.Enabled {
		if options.DecryptPassword == "" {
			return fmt.Errorf("backup is encrypted but encryption password is not configured")
		}

		decryptor := encryption.NewAESGCMEncryptor()
		decrypted, err := decryptor.Decrypt(reader, options.DecryptPassword)
		if err != nil {
			return fmt.Errorf("decrypt verification failed: %w", err)
		}
		defer decrypted.Close()
		formatReader = decrypted
	}

	if meta.Compression == "gzip" || strings.HasSuffix(plainName, ".sql.gz") {
		gz, err := gzip.NewReader(formatReader)
		if err != nil {
			return fmt.Errorf("gzip verification failed: %w", err)
		}
		defer gz.Close()

		if _, err := io.Copy(io.Discard, gz); err != nil {
			return fmt.Errorf("gzip verification failed: %w", err)
		}
	}

	if options.DatabaseType == "postgres" && meta.Format == "custom" {
		if err := verifyPostgresCustomDump(ctx, formatReader); err != nil {
			return err
		}
	}

	return nil
}

func verifyPostgresCustomDump(ctx context.Context, reader io.Reader) error {
	tmp, err := os.CreateTemp("", "backupctl-verify-*.dump")
	if err != nil {
		return fmt.Errorf("create temp dump file: %w", err)
	}
	path := tmp.Name()
	defer os.Remove(path)

	if _, err := io.Copy(tmp, reader); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp dump file: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp dump file: %w", err)
	}

	out, err := exec.CommandContext(ctx, "pg_restore", "--list", path).CombinedOutput()
	if err != nil {
		return fmt.Errorf("pg_restore --list failed: %s: %w", strings.TrimSpace(string(out)), err)
	}

	return nil
}
