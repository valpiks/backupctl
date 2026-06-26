package app

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/valpiks/backupctl/internal/backup"
	"github.com/valpiks/backupctl/internal/compression"
	dbfactory "github.com/valpiks/backupctl/internal/database/factory"
	"github.com/valpiks/backupctl/internal/encryption"
	"github.com/valpiks/backupctl/internal/logger"
	"github.com/valpiks/backupctl/internal/scheduler"
	"github.com/valpiks/backupctl/internal/secrets"
	storagefactory "github.com/valpiks/backupctl/internal/storage/factory"
)

func newSchedulerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scheduler",
		Short: "Run backup scheduler",
		Long:  "Run the long-lived backup scheduler process used by service installations.",
		Example: `  backupctl scheduler run -c configs/config.yaml
  backupctl service install --user --config configs/config.yaml`,
	}

	cmd.AddCommand(newSchedulerRunCommand())

	return cmd
}

func newSchedulerRunCommand() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run scheduled backup jobs",
		Long:  "Run scheduled backup jobs in the foreground until interrupted.",
		Example: `  backupctl scheduler run -c configs/config.yaml
  backupctl scheduler run --config /etc/backupctl/config.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, stop := signal.NotifyContext(
				cmd.Context(), os.Interrupt, syscall.SIGTERM,
			)
			defer stop()
			cfg, err := loadConfig(configPath)
			if err != nil {
				return err
			}

			knownSecrets := cfg.KnownSecrets()

			log := logger.New(cfg.Logging.Level, cfg.Logging.Format)

			if cfg.Backup.Scheduler != nil && cfg.Backup.Scheduler.LogFile != "" {
				fileLog, closeLog, err := logger.NewFile(cfg.Logging.Level, cfg.Backup.Scheduler.LogFile, cfg.Logging.Format)
				if err != nil {
					return err
				}
				defer closeLog()

				log = fileLog
			}

			driver, err := dbfactory.NewDriver(cfg.Database)
			if err != nil {
				log.Error("database driver initialization failed", "error", secrets.Redact(err.Error(), knownSecrets))
				return redactError(err, knownSecrets)
			}

			storage, err := storagefactory.NewStorage(cfg.Storage)
			if err != nil {
				log.Error("storage initialization failed", "error", secrets.Redact(err.Error(), knownSecrets))
				return redactError(err, knownSecrets)
			}

			compressor := compression.NewGzipCompressor()
			var encryptor encryption.Encryptor
			if encryptionEnabled(cfg) {
				encryptor = encryption.NewAESGCMEncryptor()
			}

			backupService := backup.NewService(driver, storage, compressor, encryptor)

			store := scheduler.NewJSONStore(".backupctl/jobs.json")
			service := scheduler.NewService(store, backupService, log)
			service.SetEncryption(encryptionEnabled(cfg), encryptionPassword(cfg))
			service.SetBackupctlVersion(Version)

			if err := service.RegisterCronJobs(ctx); err != nil {
				log.Error("register cron jobs failed", "error", secrets.Redact(err.Error(), knownSecrets))
				return redactError(err, knownSecrets)
			}

			service.Start()
			defer service.Stop()

			intervalScheduler := scheduler.NewIntervalScheduler(service, store)

			if err := intervalScheduler.Start(ctx); err != nil {
				log.Error("interval scheduler failed", "error", secrets.Redact(err.Error(), knownSecrets))
				return redactError(err, knownSecrets)
			}

			log.Info("scheduler process started")

			<-ctx.Done()

			log.Info("scheduler process stopped")
			return nil
		},
	}

	addConfigFlag(cmd, &configPath)

	return cmd
}
