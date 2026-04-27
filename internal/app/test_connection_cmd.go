package app

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/valpiks/backupctl/internal/config"
	"github.com/valpiks/backupctl/internal/database/postgres"
	"github.com/valpiks/backupctl/internal/logger"
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

			log := logger.New(cfg.Logging.Level)
			log.Info("config loaded", "path", configPath)
			log.Info("connection test started", "host", cfg.Database.Host, "port", cfg.Database.Port, "db", cfg.Database.Name)

			driver, err := postgres.NewDriver(cfg.Database)
			if err != nil {
				log.Error("database driver initialization failed", "error", err)
				return err
			}

			if err := driver.TestConnection(ctx); err != nil {
				log.Error("connection test failed", "host", cfg.Database.Host, "port", cfg.Database.Port, "db", cfg.Database.Name, "error", err)
				return err
			}

			log.Info("connection test succeeded", "host", cfg.Database.Host, "port", cfg.Database.Port, "db", cfg.Database.Name)
			fmt.Println("connection successful")
			return nil
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "configs/config.yaml", "Path to config file")

	return cmd
}
