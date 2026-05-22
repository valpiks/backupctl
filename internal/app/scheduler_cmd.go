package app

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/valpiks/backupctl/internal/backup"
	"github.com/valpiks/backupctl/internal/compression"
	"github.com/valpiks/backupctl/internal/config"
	dbfactory "github.com/valpiks/backupctl/internal/database/factory"
	"github.com/valpiks/backupctl/internal/logger"
	"github.com/valpiks/backupctl/internal/scheduler"
	storagefactory "github.com/valpiks/backupctl/internal/storage/factory"
)

func newSchedulerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scheduler",
		Short: "Run backup scheduler",
	}

	cmd.AddCommand(newSchedulerRunCommand())

	return cmd
}

func newSchedulerRunCommand() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run scheduled backup jobs",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, stop := signal.NotifyContext(
				cmd.Context(), os.Interrupt, syscall.SIGTERM,
			)
			defer stop()

			cfg, err := config.Load(configPath)
			if err != nil {
				return err
			}

			log := logger.New(cfg.Logging.Level)

			if cfg.Backup.Scheduler != nil && cfg.Backup.Scheduler.LogFile != "" {
				fileLog, closeLog, err := logger.NewFile(cfg.Logging.Level, cfg.Backup.Scheduler.LogFile)
				if err != nil {
					return err
				}
				defer closeLog()

				log = fileLog
			}

			driver, err := dbfactory.NewDriver(cfg.Database)
			if err != nil {
				return err
			}

			storage, err := storagefactory.NewStorage(cfg.Storage)
			if err != nil {
				return err
			}

			compressor := compression.NewGzipCompressor()
			backupService := backup.NewService(driver, storage, compressor)

			store := scheduler.NewJSONStore(".backupctl/jobs.json")
			service := scheduler.NewService(store, backupService, log)

			if err := service.RegisterCronJobs(ctx); err != nil {
				return err
			}

			service.Start()
			defer service.Stop()

			intervalScheduler := scheduler.NewIntervalScheduler(service, store)

			if err := intervalScheduler.Start(ctx); err != nil {
				return err
			}

			log.Info("scheduler process started")

			<-ctx.Done()

			log.Info("scheduler process stopped")
			return nil
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "configs/config.yaml", "Path to config file")

	return cmd
}
