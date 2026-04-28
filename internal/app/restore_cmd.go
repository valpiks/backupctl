package app

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/valpiks/backupctl/internal/compression"
	"github.com/valpiks/backupctl/internal/config"
	dbfactory "github.com/valpiks/backupctl/internal/database/factory"
	database "github.com/valpiks/backupctl/internal/dbdriver"
	"github.com/valpiks/backupctl/internal/logger"
	"github.com/valpiks/backupctl/internal/storage/local"
)

func newRestorCommand() *cobra.Command {
	var configPath string
	var fileName string
	var targetDB string
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

			restoreDB := cfg.Database.Name
			if targetDB != "" {
				restoreDB = targetDB
			}

			log := logger.New(cfg.Logging.Level)
			log.Info("config loaded", "path", configPath)

			if !yes {
				confirmed, err := confirmRestore(fileName, restoreDB)
				if err != nil {
					log.Error("restore confirmation failed", "file", fileName, "error", err)
					return err
				}

				if !confirmed {
					log.Info("restore cancelled", "file", fileName, "db", restoreDB)
					fmt.Println("restore cancelled")
					return nil
				}
			}

			log.Info("restore started", "file", fileName, "db", restoreDB)

			driver, err := dbfactory.NewDriver(cfg.Database)
			if err != nil {
				log.Error("database driver initialization failed", "error", err)
				return err
			}

			storage, err := local.NewStorage(cfg.Storage.Path)
			if err != nil {
				log.Error("storage initialization failed", "path", cfg.Storage.Path, "error", err)
				return err
			}

			reader, err := storage.Open(ctx, fileName)
			if err != nil {
				log.Error("open backup failed", "file", fileName, "error", err)
				return err
			}
			defer reader.Close()

			compressor := compression.NewGzipCompressor()

			decompressionReader, err := compressor.Decompress(reader)
			if err != nil {
				log.Error("decompress backup failed", "file", fileName, "error", err)
				return err
			}
			defer decompressionReader.Close()

			err = driver.Restore(ctx, decompressionReader, database.RestoreOptions{TargetDatabase: restoreDB})
			if err != nil {
				log.Error("restore failed", "file", fileName, "db", restoreDB, "error", err)
				return err
			}

			log.Info("restore finished", "file", fileName, "db", restoreDB)
			fmt.Println("restore complete successfully")
			return nil
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "configs/config.yaml", "Path to config file")
	cmd.Flags().StringVar(&fileName, "file", "", "Backup file name")
	cmd.Flags().StringVar(&targetDB, "target-db", "", "Target database name for restore")
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")

	return cmd
}

func confirmRestore(fileName string, dbName string) (bool, error) {
	fmt.Printf(
		"WARNING: you are about to restore database %q from %q\n",
		dbName,
		fileName,
	)
	fmt.Print("This may overwrite existing data. Continue? [y/N]: ")

	reader := bufio.NewReader(os.Stdin)
	answer, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("read confirmation: %w", err)
	}

	answer = strings.TrimSpace(strings.ToLower(answer))

	return answer == "y" || answer == "yes", nil
}
