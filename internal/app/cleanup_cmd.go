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
	storagefactory "github.com/valpiks/backupctl/internal/storage/factory"
)

func newCleanupCommand(opts CLIOptions) *cobra.Command {
	var configPath string
	var keepLast int
	var dryRun bool
	var jsonOutput bool
	var yes bool

	cmd := &cobra.Command{
		Use:   "cleanup --keep-last <count>",
		Short: "Delete old backups",
		Long:  "Delete old backups from configured storage while keeping the newest backups.",
		Example: `  backupctl cleanup -c configs/config.yaml --keep-last 10 --dry-run
  backupctl cleanup -c configs/config.yaml --keep-last 10`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if keepLast <= 0 {
				return WithHint(fmt.Errorf("--keep-last must be greater than 0"), "choose how many recent backups to keep, for example --keep-last 10")
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
			defer cancel()
			cfg, err := loadConfig(configPath)
			if err != nil {
				return err
			}
			knownSecrets := cfg.KnownSecrets()

			storage, err := storagefactory.NewStorage(cfg.Storage)
			if err != nil {
				return redactError(err, knownSecrets)
			}

			files, err := storage.List(ctx)
			if err != nil {
				return redactError(err, knownSecrets)
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
					return redactError(err, knownSecrets)
				}

				var metadata backup.Metadata
				if err := json.Unmarshal(data, &metadata); err != nil {
					return redactError(fmt.Errorf("parse metadata %s: %w", file.Name, err), knownSecrets)
				}

				metadataList = append(metadataList, metadata)
			}

			sort.Slice(metadataList, func(i, j int) bool {
				return metadataList[i].StartedAt.After(metadataList[j].StartedAt)
			})

			if len(metadataList) > keepLast {
				metadataList = metadataList[keepLast:]
			} else {
				metadataList = nil
			}

			if dryRun {
				payload := cleanupPayload(dryRun, len(metadataList))
				if jsonOutput {
					return PrintJSON(cmd.OutOrStdout(), payload)
				}

				if len(metadataList) == 0 {
					fmt.Fprintln(cmd.OutOrStdout(), "No files would be deleted.")
					return nil
				}

				rows := make([][]string, 0, len(metadataList))
				for _, m := range metadataList {
					rows = append(rows, []string{m.FileName, metadataNameForBackup(m.FileName)})
				}
				fmt.Fprintln(cmd.OutOrStdout(), "Files that would be deleted")
				PrintTable(cmd.OutOrStdout(), []string{"Backup file", "Metadata file"}, rows)
				fmt.Fprintf(cmd.OutOrStdout(), "total: %d files\n", len(metadataList)*2)
				return nil
			}

			if !yes && len(metadataList) > 0 {
				return WithHint(
					fmt.Errorf("cleanup deletes files and requires confirmation"),
					"rerun with --dry-run to preview or --yes to confirm deletion",
				)
			}

			for _, m := range metadataList {
				if err := storage.Delete(ctx, m.FileName); err != nil {
					return redactError(fmt.Errorf("delete backup %s: %w", m.FileName, err), knownSecrets)
				}

				metaName := metadataNameForBackup(m.FileName)
				if err := storage.Delete(ctx, metaName); err != nil {
					return redactError(fmt.Errorf("delete metadata %s: %w", metaName, err), knownSecrets)
				}
			}

			payload := cleanupPayload(dryRun, len(metadataList))
			if jsonOutput {
				return PrintJSON(cmd.OutOrStdout(), payload)
			}

			if !opts.Quiet {
				PrintKV(cmd.OutOrStdout(), "Deleted old backups", []KV{
					{Key: "backups", Value: fmt.Sprintf("%d", len(metadataList))},
					{Key: "metadata", Value: fmt.Sprintf("%d", len(metadataList))},
					{Key: "total", Value: fmt.Sprintf("%d", len(metadataList)*2)},
				})
			}
			return nil
		},
	}

	addConfigFlag(cmd, &configPath)
	cmd.Flags().IntVar(&keepLast, "keep-last", 0, "Set cleanup limit")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview delete files")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Print cleanup result as JSON")
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip cleanup confirmation")
	_ = cmd.MarkFlagRequired("keep-last")

	return cmd
}

func cleanupPayload(dryRun bool, count int) map[string]any {
	return map[string]any{
		"status":         "success",
		"dry_run":        dryRun,
		"backups":        count,
		"metadata_files": count,
		"total_files":    count * 2,
	}
}

func metadataNameForBackup(fileName string) string {
	name := strings.TrimSuffix(fileName, ".enc")
	name = strings.TrimSuffix(name, ".sql.gz")
	name = strings.TrimSuffix(name, ".sql")
	name = strings.TrimSuffix(name, ".dump")
	return name + ".metadata.json"
}
