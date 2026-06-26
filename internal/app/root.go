package app

import (
	"github.com/spf13/cobra"
)

func NewRootCommand() *cobra.Command {
	var opts CLIOptions

	rootCmd := &cobra.Command{
		Use:   "backupctl",
		Short: "Database backup CLI utility",
		Long: `backupctl creates, stores, restores, and schedules database backups.

It supports PostgreSQL and MongoDB, local and S3-compatible storage,
gzip compression, AES-GCM encryption, scheduled jobs, and background
services through systemd or launchd.`,
		Example: `  backupctl init
  backupctl doctor
  backupctl backup
  backupctl list
  backupctl restore --file app_20260607_120000.sql.gz
  backupctl schedule --interval 24h
  backupctl scheduler run
  backupctl service install --user`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.PersistentFlags().BoolVar(&opts.Quiet, "quiet", false, "Suppress success output")
	rootCmd.PersistentFlags().BoolVar(&opts.Verbose, "verbose", false, "Print diagnostic logs")
	rootCmd.PersistentFlags().StringVar(&opts.Color, "color", "auto", "Color output: auto, always, never")

	rootCmd.AddCommand(newVersionCommand())
	rootCmd.AddCommand(newInitCommand(opts))
	rootCmd.AddCommand(newConfigCommand(opts))
	rootCmd.AddCommand(newTestCommand(opts))
	rootCmd.AddCommand(newBackupCommand(opts))
	rootCmd.AddCommand(newListCommand(opts))
	rootCmd.AddCommand(newRestoreCommand(opts))
	rootCmd.AddCommand(newDoctorCommand(opts))
	rootCmd.AddCommand(newCleanupCommand(opts))
	rootCmd.AddCommand(newVerifyCommand(opts))
	rootCmd.AddCommand(newScheduleCommand(opts))
	rootCmd.AddCommand(newSchedulerCommand())
	rootCmd.AddCommand(newServiceCommand())
	rootCmd.AddCommand(newJobsCommand(opts))
	rootCmd.AddCommand(newCompletionCommand())

	return rootCmd
}
