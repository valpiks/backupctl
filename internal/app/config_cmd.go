package app

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/valpiks/backupctl/internal/config"
	"github.com/valpiks/backupctl/internal/logger"
)

func newConfigCommand() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "config",
		Short: "Print loaded config",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configPath)
			if err != nil {
				return err
			}

			log := logger.New(cfg.Logging.Level)
			log.Info("config loaded", "path", configPath)

			fmt.Printf("database type: %s\n", cfg.Database.Type)
			fmt.Printf("database name: %s\n", cfg.Database.ActiveDatabaseName())

			log.Info("config printed", "db", cfg.Database.ActiveDatabaseName())
			return nil
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "configs/config.yaml", "Path to config file")

	return cmd
}
