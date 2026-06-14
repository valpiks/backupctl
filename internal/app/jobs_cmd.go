package app

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/valpiks/backupctl/internal/scheduler"
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
