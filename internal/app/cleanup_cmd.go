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
	storagefactory "github.com/valpiks/backupctl/internal/storage/factory"
)

func newCleanupCommand() *cobra.Command {
	var configPath string
	var keepLast int
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Cleanup storage directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			if keepLast <= 0 {
				return fmt.Errorf("--keep-last must be greater than 0")
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
			defer cancel()

			cfg, err := config.Load(configPath)
			if err != nil {
				return err
			}

			storage, err := storagefactory.NewStorage(cfg.Storage)
			if err != nil {
				return err
			}

			files, err := storage.List(ctx)
			if err != nil {
				return err
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
					return err
				}

				var metadata backup.Metadata
				if err := json.Unmarshal(data, &metadata); err != nil {
					return fmt.Errorf("parse metadata %s: %w", file.Name, err)
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
				fmt.Println("would delete:")
				for _, m := range metadataList {
					fmt.Printf("- %s\n", m.FileName)
					metaName := strings.TrimSuffix(m.FileName, ".sql.gz") + "metadata.json"
					fmt.Printf("- %s\n", metaName)
				}
				return nil
			}

			for _, m := range metadataList {
				if err := storage.Delete(ctx, m.FileName); err != nil {
					return fmt.Errorf("delete backup %s: %w", m.FileName, err)
				}

				metaName := strings.TrimSuffix(m.FileName, ".sql.gz") + "metadata.json"
				if err := storage.Delete(ctx, metaName); err != nil {
					return fmt.Errorf("delete metadata %s: %w", metaName, err)
				}
			}

			fmt.Println("files deleted successfully")
			return nil
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "configs/config.yaml", "Path to config file")
	cmd.Flags().IntVar(&keepLast, "keep-last", 0, "Set cleanup limit")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview delete files")

	return cmd
}
