package app

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/spf13/cobra"
	"github.com/valpiks/backupctl/internal/config"
	dbfactory "github.com/valpiks/backupctl/internal/database/factory"
	"github.com/valpiks/backupctl/internal/secrets"
	storagefactory "github.com/valpiks/backupctl/internal/storage/factory"
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
			knownSecrets := cfg.KnownSecrets()

			fmt.Println("[OK] config loaded")

			driver, err := dbfactory.NewDriver(cfg.Database)
			if err != nil {
				fmt.Printf("[FAIL] driver: %s\n", secrets.Redact(err.Error(), knownSecrets))
				return redactError(err, knownSecrets)
			}
			fmt.Println("[OK] driver initialized")

			if err := driver.Ping(ctx); err != nil {
				fmt.Printf("[FAIL] database ping: %s\n", secrets.Redact(err.Error(), knownSecrets))
				return redactError(err, knownSecrets)
			}
			fmt.Println("[OK] database ping")

			if _, err := storagefactory.NewStorage(cfg.Storage); err != nil {
				fmt.Printf("[FAIL] storage: %s\n", secrets.Redact(err.Error(), knownSecrets))
				return redactError(err, knownSecrets)
			}
			fmt.Println("[OK] storage initialized")

			dbTools := requiredDatabaseTools(cfg.Database.Type)

			if len(dbTools) == 0 {
				return fmt.Errorf("unknown db type")
			}

			for _, tool := range dbTools {
				if _, err := exec.LookPath(tool); err != nil {
					fmt.Printf("[FAIL] %s not found: %v\n", tool, err)
					return err
				}
				fmt.Printf("[OK] %s found\n", tool)
			}

			fmt.Println("doctor checks passed")
			return nil
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "configs/config.yaml", "Path to config file")

	return cmd
}

func requiredDatabaseTools(databaseType string) []string {
	switch databaseType {
	case "postgres":
		return []string{"psql", "pg_dump"}
	case "mongo":
		return []string{"mongosh", "mongodump", "mongorestore"}
	default:
		return []string{}
	}
}
