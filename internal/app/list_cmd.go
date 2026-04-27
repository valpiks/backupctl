package app

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/valpiks/dbbackup/internal/config"
	"github.com/valpiks/dbbackup/internal/storage/local"
)

func newListCommand() *cobra.Command {
	var configPath string

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

			for _, f := range files {
				fmt.Printf("%s (%d bytes)\n", f.Name, f.Size)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "configs/config.yaml", "Path to config file")

	return cmd
}
