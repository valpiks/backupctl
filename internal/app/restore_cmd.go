package app

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/valpiks/dbbackup/internal/compression"
	"github.com/valpiks/dbbackup/internal/config"
	"github.com/valpiks/dbbackup/internal/database"
	"github.com/valpiks/dbbackup/internal/database/postgres"
	"github.com/valpiks/dbbackup/internal/storage/local"
)

func newRestorCommand() *cobra.Command {
	var configPath string
	var fileName string
	var yes bool

	cmd := &cobra.Command{
		Use:   "restore",
		Short: "Restore database from backup",
		RunE: func(cmd *cobra.Command, args []string) error {
			if fileName == "" {
				return fmt.Errorf("--file is required")
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()

			cfg, err := config.Load(configPath)
			if err != nil {
				return err
			}

			if !yes {
				confirmed, err := confirmRestore(fileName)
				if err != nil {
					return err
				}

				if !confirmed {
					fmt.Println("restore cancelled")
					return nil
				}
			}

			driver, err := postgres.NewDriver(cfg.Database)
			if err != nil {
				return err
			}

			storage, err := local.NewStorage(cfg.Storage.Path)
			if err != nil {
				return err
			}

			reader, err := storage.Open(ctx, fileName)
			if err != nil {
				return err
			}
			defer reader.Close()

			compressor := compression.NewGzipCompressor()

			decompressionReader, err := compressor.Decompress(reader)
			if err != nil {
				return err
			}
			defer decompressionReader.Close()

			err = driver.Restore(ctx, decompressionReader, database.RestoreOptions{TargetDatabase: cfg.Database.Name})
			if err != nil {
				return err
			}

			fmt.Println("restore complete successfully")
			return nil
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "configs/config.yaml", "Path to config file")
	cmd.Flags().StringVar(&fileName, "file", "", "Backup file name")
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")

	return cmd
}

func confirmRestore(fileName string) (bool, error) {
	fmt.Printf("Are you sure you want to restore from %q? [y/N]: ", fileName)

	reader := bufio.NewReader(os.Stdin)
	answer, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("read confirmation: %w", err)
	}

	answer = strings.TrimSpace(strings.ToLower(answer))

	return answer == "y" || answer == "yes", nil
}
