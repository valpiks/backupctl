package app

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/valpiks/backupctl/internal/config"
	dbfactory "github.com/valpiks/backupctl/internal/database/factory"
	"github.com/valpiks/backupctl/internal/logger"
	"github.com/valpiks/backupctl/internal/secrets"
)

func newTestCommand() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "test-connection",
		Short: "Test database connection (deprecated: use doctor)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
			defer cancel()

			cfg, err := config.Load(configPath)
			if err != nil {
				return err
			}
			knownSecrets := cfg.KnownSecrets()

			log := logger.New(cfg.Logging.Level)
			log.Info("config loaded", "path", configPath)
			log.Info("connection test started", "db", cfg.Database.ActiveDatabaseName())

			driver, err := dbfactory.NewDriver(cfg.Database)
			if err != nil {
				log.Error("database driver initialization failed", "error", secrets.Redact(err.Error(), knownSecrets))
				return redactError(err, knownSecrets)
			}

			if err := driver.Ping(ctx); err != nil {
				log.Error("connection test failed", "db", cfg.Database.ActiveDatabaseName(), "error", secrets.Redact(err.Error(), knownSecrets))
				return redactError(err, knownSecrets)
			}

			log.Info("connection test succeeded", "db", cfg.Database.ActiveDatabaseName())
			fmt.Println("connection successful")
			return nil
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "configs/config.yaml", "Path to config file")

	return cmd
}
