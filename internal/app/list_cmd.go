package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/valpiks/backupctl/internal/backup"
	"github.com/valpiks/backupctl/internal/secrets"
	storagepkg "github.com/valpiks/backupctl/internal/storage"
	storagefactory "github.com/valpiks/backupctl/internal/storage/factory"
)

func newListCommand(opts CLIOptions) *cobra.Command {
	var configPath string
	var limit int
	var showFiles bool
	var jsonOutput bool
	var verifiedOnly bool

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List backups",
		Long:    "List backups and backup metadata from configured storage.",
		Example: `  backupctl list -c configs/config.yaml
  backupctl list -c configs/config.yaml --limit 10
  backupctl list -c configs/config.yaml --json
  backupctl list -c configs/config.yaml --files`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
			defer cancel()
			cfg, err := loadConfig(configPath)
			if err != nil {
				return err
			}
			knownSecrets := cfg.KnownSecrets()

			log := commandLogger(cfg, opts)
			log.Info("config loaded", "path", configPath)

			storage, err := storagefactory.NewStorage(cfg.Storage)
			if err != nil {
				log.Error("storage initialization failed", "error", secrets.Redact(err.Error(), knownSecrets))
				return redactError(err, knownSecrets)
			}

			log.Info("listing backups", "show_files", showFiles, "json", jsonOutput)

			files, err := storage.List(ctx)
			if err != nil {
				log.Error("list backups failed", "error", secrets.Redact(err.Error(), knownSecrets))
				return redactError(err, knownSecrets)
			}

			if verifiedOnly {
				filtered := make([]storagepkg.BackupFile, 0, len(files))
				for _, file := range files {
					if strings.HasSuffix(file.Name, ".metadata.json") {
						continue
					}

					if _, err := verifyBackup(ctx, storage, file.Name, verifyOptions{DatabaseType: cfg.Database.Type, DecryptPassword: encryptionPassword(cfg)}); err == nil {
						filtered = append(filtered, file)
					}
				}

				files = filtered
			}

			fileSizes := make(map[string]int64, len(files))
			for _, file := range files {
				fileSizes[file.Name] = file.Size
			}

			if showFiles {
				if jsonOutput {
					return PrintJSON(cmd.OutOrStdout(), files)
				}

				if len(files) == 0 {
					fmt.Fprintln(cmd.OutOrStdout(), "No files found.")
					log.Info("raw backup files listed", "count", len(files))
					return nil
				}

				rows := make([][]string, 0, len(files))
				for _, f := range files {
					rows = append(rows, []string{f.Name, HumanBytes(f.Size)})
				}
				PrintTable(cmd.OutOrStdout(), []string{"File", "Size"}, rows)
				log.Info("raw backup files listed", "count", len(files))
				return nil
			}

			var metadataList []backup.Metadata

			for _, file := range files {
				if !strings.HasSuffix(file.Name, "metadata.json") {
					continue
				}

				reader, err := storage.Open(ctx, file.Name)
				if err != nil {
					return redactError(err, knownSecrets)
				}

				data, err := io.ReadAll(reader)
				_ = reader.Close()
				if err != nil {
					log.Error("read metadata failed", "file", file.Name, "error", secrets.Redact(err.Error(), knownSecrets))
					return redactError(err, knownSecrets)
				}

				var metadata backup.Metadata
				if err := json.Unmarshal(data, &metadata); err != nil {
					log.Error("parse metadata failed", "file", file.Name, "error", secrets.Redact(err.Error(), knownSecrets))
					return redactError(fmt.Errorf("parse metadata %s: %w", file.Name, err), knownSecrets)
				}

				metadataList = append(metadataList, metadata)
			}

			if verifiedOnly {
				filtered := make([]backup.Metadata, 0, len(metadataList))

				for _, meta := range metadataList {
					if _, err := verifyBackup(ctx, storage, meta.FileName, verifyOptions{DatabaseType: cfg.Database.Type, DecryptPassword: encryptionPassword(cfg)}); err == nil {
						filtered = append(filtered, meta)
					}
				}

				metadataList = filtered
			}

			sort.Slice(metadataList, func(i, j int) bool {
				return metadataList[i].StartedAt.After(metadataList[j].StartedAt)
			})

			if limit > 0 && len(metadataList) > limit {
				metadataList = metadataList[:limit]
			}

			if jsonOutput {
				data, err := json.MarshalIndent(metadataList, "", "  ")
				if err != nil {
					log.Error("marshal backups json failed", "error", secrets.Redact(err.Error(), knownSecrets))
					return redactError(err, knownSecrets)
				}

				fmt.Fprintln(cmd.OutOrStdout(), string(data))
				log.Info("backups listed", "files_total", len(files), "metadata_total", len(metadataList), "output", "json")
				return nil
			}

			if len(metadataList) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No backups found.")
				log.Info("backups listed", "files_total", len(files), "metadata_total", len(metadataList), "output", "table")
				return nil
			}

			rows := make([][]string, 0, len(metadataList))
			for _, m := range metadataList {
				rows = append(rows, []string{
					m.StartedAt.Format("2006-01-02 15:04:05"),
					m.DatabaseName,
					m.BackupType,
					m.Format,
					HumanBytes(fileSizes[m.FileName]),
					m.Status,
					m.Duration,
					m.FileName,
				})
			}
			PrintTable(cmd.OutOrStdout(), []string{"Started", "Database", "Type", "Format", "Size", "Status", "Duration", "File"}, rows)

			log.Info("backups listed", "files_total", len(files), "metadata_total", len(metadataList), "output", "table")
			return nil
		},
	}

	addConfigFlag(cmd, &configPath)
	cmd.Flags().IntVar(&limit, "limit", 0, "Limit number of backups shown")
	cmd.Flags().BoolVar(&showFiles, "files", false, "Show raw backup directory files")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Print backups as JSON")
	cmd.Flags().BoolVar(&verifiedOnly, "verified", false, "Show only backups that pass metadata verification")

	return cmd
}
