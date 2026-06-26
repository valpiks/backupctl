package app

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/valpiks/backupctl/internal/backup"
	"github.com/valpiks/backupctl/internal/compression"
	dbfactory "github.com/valpiks/backupctl/internal/database/factory"
	"github.com/valpiks/backupctl/internal/encryption"
	"github.com/valpiks/backupctl/internal/scheduler"
	storagefactory "github.com/valpiks/backupctl/internal/storage/factory"
)

func newJobsCommand(opts CLIOptions) *cobra.Command {
	_ = opts
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "jobs",
		Short: "Manage scheduled backup jobs",
		Long:  "List and manage scheduled backup jobs stored in .backupctl/jobs.json.",
		Example: `  backupctl jobs
  backupctl jobs status job_20260607_120000
  backupctl jobs delete job_20260607_120000`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			store := scheduler.NewJSONStore(".backupctl/jobs.json")

			jobs, err := store.List(ctx)
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			if jsonOutput {
				return PrintJSON(out, jobs)
			}

			if len(jobs) == 0 {
				fmt.Fprintln(out, "No scheduled jobs found.")
				return nil
			}

			rows := make([][]string, 0, len(jobs))
			for _, job := range jobs {
				schedule := job.Cron
				if schedule == "" {
					schedule = job.Interval
				}

				rows = append(rows, []string{job.ID, job.DatabaseName, schedule, job.Format, job.Status})
			}
			PrintTable(out, []string{"ID", "Database", "Schedule", "Format", "Status"}, rows)

			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Print jobs as JSON")
	cmd.AddCommand(newJobsStatusCommand())
	cmd.AddCommand(newJobsDeleteCommand())
	cmd.AddCommand(newJobsRunCommand(opts))
	cmd.AddCommand(newJobsEnableCommand())
	cmd.AddCommand(newJobsDisableCommand())
	cmd.AddCommand(newJobsLogsCommand())

	return cmd
}

func newJobsStatusCommand() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:     "status <job-id>",
		Short:   "Show scheduled job status",
		Long:    "Show details for one scheduled backup job.",
		Example: `  backupctl jobs status job_20260607_120000`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Second)
			defer cancel()

			store := scheduler.NewJSONStore(".backupctl/jobs.json")

			job, err := store.Get(ctx, args[0])
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			if jsonOutput {
				return PrintJSON(out, job)
			}

			rows := []KV{
				{Key: "id", Value: job.ID},
				{Key: "database", Value: job.DatabaseName},
				{Key: "backup_type", Value: job.BackupType},
				{Key: "format", Value: job.Format},
				{Key: "status", Value: job.Status},
				{Key: "enabled", Value: YesNo(!job.Disabled)},
			}

			if job.LastError != "" {
				rows = append(rows, KV{Key: "last_error", Value: job.LastError})
			}

			if !job.CreatedAt.IsZero() {
				rows = append(rows, KV{Key: "created_at", Value: job.CreatedAt.Format(time.RFC3339)})
			}

			if !job.UpdatedAt.IsZero() {
				rows = append(rows, KV{Key: "updated_at", Value: job.UpdatedAt.Format(time.RFC3339)})
			}

			if job.Cron != "" {
				rows = append(rows, KV{Key: "cron", Value: job.Cron})
			}

			if job.Interval != "" {
				rows = append(rows, KV{Key: "interval", Value: job.Interval})
			}

			if job.LastRun != nil {
				rows = append(rows, KV{Key: "last_run", Value: job.LastRun.Format(time.RFC3339)})
			}

			if job.NextRun != nil {
				rows = append(rows, KV{Key: "next_run", Value: job.NextRun.Format(time.RFC3339)})
			}

			if job.LogFile != "" {
				rows = append(rows, KV{Key: "log_file", Value: job.LogFile})
			}

			PrintKV(out, "Job status", rows)
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Print job status as JSON")
	return cmd
}

func newJobsEnableCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "enable <job-id>",
		Short: "Enable scheduled job",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := setJobDisabled(cmd.Context(), args[0], false); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "job enabled: %s\n", args[0])
			return nil
		},
	}
}

func newJobsDisableCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "disable <job-id>",
		Short: "Disable scheduled job",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := setJobDisabled(cmd.Context(), args[0], true); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "job disabled: %s\n", args[0])
			return nil
		},
	}
}

func setJobDisabled(ctx context.Context, id string, disabled bool) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	store := scheduler.NewJSONStore(".backupctl/jobs.json")
	job, err := store.Get(ctx, id)
	if err != nil {
		return err
	}

	job.Disabled = disabled
	job.UpdatedAt = time.Now().UTC()

	return store.Update(ctx, job)
}

func newJobsLogsCommand() *cobra.Command {
	var tail int

	cmd := &cobra.Command{
		Use:   "logs <job-id>",
		Short: "Print scheduled job logs",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Second)
			defer cancel()

			store := scheduler.NewJSONStore(".backupctl/jobs.json")
			job, err := store.Get(ctx, args[0])
			if err != nil {
				return err
			}

			if job.LogFile == "" {
				return fmt.Errorf("job has no log file configured")
			}

			data, err := os.ReadFile(job.LogFile)
			if err != nil {
				return fmt.Errorf("read job log file: %w", err)
			}

			lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
			if tail > 0 && len(lines) > tail {
				lines = lines[len(lines)-tail:]
			}

			for _, line := range lines {
				if line != "" {
					fmt.Fprintln(cmd.OutOrStdout(), line)
				}
			}

			return nil
		},
	}

	cmd.Flags().IntVar(&tail, "tail", 100, "Number of log lines to print")
	return cmd
}

func newJobsDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "delete <job-id>",
		Aliases: []string{"rm"},
		Short:   "Delete scheduled job",
		Long:    "Delete one scheduled backup job from .backupctl/jobs.json.",
		Example: `  backupctl jobs delete job_20260607_120000
  backupctl jobs rm job_20260607_120000`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Second)
			defer cancel()

			store := scheduler.NewJSONStore(".backupctl/jobs.json")

			if err := store.Delete(ctx, args[0]); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "job deleted: %s\n", args[0])
			return nil
		},
	}
}

func newJobsRunCommand(opts CLIOptions) *cobra.Command {
	var configPath string
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "run <job-id>",
		Short: "Run scheduled job now",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Minute)
			defer cancel()

			cfg, err := loadConfig(configPath)
			if err != nil {
				return err
			}

			knownSecrets := cfg.KnownSecrets()
			log := commandLogger(cfg, opts)

			driver, err := dbfactory.NewDriver(cfg.Database)
			if err != nil {
				return redactError(err, knownSecrets)
			}

			storage, err := storagefactory.NewStorage(cfg.Storage)
			if err != nil {
				return redactError(err, knownSecrets)
			}

			var encryptor encryption.Encryptor
			if encryptionEnabled(cfg) {
				encryptor = encryption.NewAESGCMEncryptor()
			}

			backupService := backup.NewService(
				driver,
				storage,
				compression.NewGzipCompressor(),
				encryptor,
			)

			store := scheduler.NewJSONStore(".backupctl/jobs.json")
			job, err := store.Get(ctx, args[0])
			if err != nil {
				return err
			}

			service := scheduler.NewService(store, backupService, log)
			service.SetEncryption(encryptionEnabled(cfg), encryptionPassword(cfg))
			service.SetBackupctlVersion(Version)

			if err := service.RunJob(ctx, job); err != nil {
				return redactError(err, knownSecrets)
			}

			if jsonOutput {
				return PrintJSON(cmd.OutOrStdout(), job)
			}

			if !opts.Quiet {
				PrintKV(cmd.OutOrStdout(), "Job completed", []KV{
					{Key: "id", Value: job.ID},
					{Key: "database", Value: job.DatabaseName},
					{Key: "status", Value: job.Status},
				})
			}

			return nil
		},
	}

	addConfigFlag(cmd, &configPath)
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Print job as JSON")

	return cmd
}
