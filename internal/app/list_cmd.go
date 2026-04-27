package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/valpiks/dbbackup/internal/backup"
	"github.com/valpiks/dbbackup/internal/config"
	"github.com/valpiks/dbbackup/internal/storage/local"
)

func newListCommand() *cobra.Command {
	var configPath string
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

			storage, err := local.NewStorage(cfg.Storage.Path)
			if err != nil {
				return err
			}

			files, err := storage.List(ctx)
			if err != nil {
				return err
			}

			if showFiles {
				for _, f := range files {
					fmt.Printf("%s (%d bytes)\n", f.Name, f.Size)
				}
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
					return nil
				}

				var metadata backup.Metadata
				if err := json.Unmarshal(data, &metadata); err != nil {
					return fmt.Errorf("parse metadata %s: %w", file.Name, err)
				}

				metadataList = append(metadataList, metadata)
			}

			if jsonOutput {
				data, err := json.MarshalIndent(metadataList, "", "  ")
				if err != nil {
					return err
				}

				fmt.Println(string(data))
				return nil
			}

			fmt.Printf("%-40s %-15s %-8s %-10s %-10s\n", "FILE", "DATABASE", "TYPE", "STATUS", "DURATION")

			for _, m := range metadataList {
				fmt.Printf("%-40s %-15s %-8s %-10s %-10s\n", m.FileName, m.DatabaseName, m.BackupType, m.Status, m.Duration)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "configs/config.yaml", "Path to config file")
	cmd.Flags().BoolVar(&showFiles, "files", false, "Show raw backup directory files")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Print backups as JSON")

	return cmd
}
