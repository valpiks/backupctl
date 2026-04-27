package app

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/valpiks/dbbackup/internal/config"
)

func NewConfigCommand() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "config",
		Short: "Print loaded config",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configPath)
			if err != nil {
				return err
			}

			fmt.Printf("database type: %s\n", cfg.Database.Type)
			fmt.Printf("database name: %s\n", cfg.Database.Name)
			fmt.Printf("storage path: %s\n", cfg.Storage.Path)

			return nil
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "configs/config.yaml", "Path to config file")

	return cmd
}
