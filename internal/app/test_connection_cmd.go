package app

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	dbfactory "github.com/valpiks/backupctl/internal/database/factory"
	"github.com/valpiks/backupctl/internal/secrets"
)

func newTestCommand(opts CLIOptions) *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "test-connection",
		Short: "Test database connection (deprecated: use doctor)",
		Long:  "Test the configured database connection. This command is deprecated; use backupctl doctor instead.",
		Example: `  backupctl test-connection -c configs/config.yaml
  backupctl doctor -c configs/config.yaml`,
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
			if !opts.Quiet {
				fmt.Fprintln(cmd.OutOrStdout(), "connection successful")
			}
			return nil
		},
	}

	addConfigFlag(cmd, &configPath)

	return cmd
}
