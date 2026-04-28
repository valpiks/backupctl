package app

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/spf13/cobra"
	"github.com/valpiks/backupctl/internal/config"
	"github.com/valpiks/backupctl/internal/database/factory"
	"github.com/valpiks/backupctl/internal/storage/local"
)

func newDoctorCommand() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Run environment and configuration checks",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
			defer cancel()

			cfg, err := config.Load(configPath)
			if err != nil {
				return err
			}

			fmt.Printf("[OK] config loaded")

			driver, err := factory.NewDriver(cfg.Database)
			if err != nil {
				fmt.Printf("[FAIL] driver: %v\n", err)
				return err
			}
			fmt.Println("[OK] driver initialized")

			if err := driver.Ping(ctx); err != nil {
				fmt.Printf("[FAIL] database ping: %v\n", err)
				return err
			}
			fmt.Println("[OK] database ping")

			if _, err := local.NewStorage(cfg.Storage.Path); err != nil {
				fmt.Printf("[FAIL] storage: %v\n", err)
				return err
			}
			fmt.Println("[OK] storage initialized")

			if _, err := exec.LookPath("pg_dump"); err != nil {
				fmt.Printf("[FAIL] pg_dump not found: %v\n", err)
				return err
			}
			fmt.Println("[OK] pg_dump found")

			if _, err := exec.LookPath("psql"); err != nil {
				fmt.Printf("[FAIL] psql not found: %v\n", err)
				return err
			}
			fmt.Println("[OK] psql found")

			fmt.Println("doctor checks passed")
			return nil
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "configs/config.yaml", "Path to config file")

	return cmd
}
