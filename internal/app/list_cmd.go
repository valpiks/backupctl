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
	"github.com/valpiks/backupctl/internal/config"
	"github.com/valpiks/backupctl/internal/logger"
	"github.com/valpiks/backupctl/internal/storage/local"
)

func newListCommand() *cobra.Command {
	var configPath string
	var limit int
	var showFiles bool
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List backups",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
			defer cancel()

			cfg, err := config.Load(configPath)
			if err != nil {
				return err
			}

			log := logger.New(cfg.Logging.Level)
			log.Info("config loaded", "path", configPath)

			storage, err := local.NewStorage(cfg.Storage.Path)
			if err != nil {
				log.Error("storage initialization failed", "path", cfg.Storage.Path, "error", err)
				return err
			}

			log.Info("listing backups", "path", cfg.Storage.Path, "show_files", showFiles, "json", jsonOutput)

			files, err := storage.List(ctx)
			if err != nil {
				log.Error("list backups failed", "path", cfg.Storage.Path, "error", err)
				return err
			}

			if showFiles {
				for _, f := range files {
					fmt.Printf("%s (%d bytes)\n", f.Name, f.Size)
				}
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
					return err
				}

				data, err := io.ReadAll(reader)
				_ = reader.Close()
				if err != nil {
					log.Error("read metadata failed", "file", file.Name, "error", err)
					return err
				}

				var metadata backup.Metadata
				if err := json.Unmarshal(data, &metadata); err != nil {
					log.Error("parse metadata failed", "file", file.Name, "error", err)
					return fmt.Errorf("parse metadata %s: %w", file.Name, err)
				}

				metadataList = append(metadataList, metadata)
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
					log.Error("marshal backups json failed", "error", err)
					return err
				}

				fmt.Println(string(data))
				log.Info("backups listed", "files_total", len(files), "metadata_total", len(metadataList), "output", "json")
				return nil
			}

			fmt.Printf("%-40s %-15s %-8s %-10s %-10s\n", "FILE", "DATABASE", "TYPE", "STATUS", "DURATION")

			for _, m := range metadataList {
				fmt.Printf("%-40s %-15s %-8s %-10s %-10s\n", m.FileName, m.DatabaseName, m.BackupType, m.Status, m.Duration)
			}

			log.Info("backups listed", "files_total", len(files), "metadata_total", len(metadataList), "output", "table")
			return nil
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "configs/config.yaml", "Path to config file")
	cmd.Flags().IntVar(&limit, "limit", 0, "Limit number of backups shown")
	cmd.Flags().BoolVar(&showFiles, "files", false, "Show raw backup directory files")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Print backups as JSON")

	return cmd
}
