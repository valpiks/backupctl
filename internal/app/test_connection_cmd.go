package app

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/valpiks/dbbackup/internal/config"
	"github.com/valpiks/dbbackup/internal/database/postgres"
)

func newTestCommand() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "test-connection",
		Short: "Test database connection",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
			defer cancel()

			cfg, err := config.Load(configPath)
			if err != nil {
				return err
			}

			driver, err := postgres.NewDriver(cfg.Database)
			if err != nil {
				return err
			}

			if err := driver.TestConnection(ctx); err != nil {
				return err
			}

			fmt.Println("connection successful")
			return nil
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "configs/config.yaml", "Path to config file")

	return cmd
}
