package app

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/valpiks/backupctl/internal/scheduler"
)

func newJobsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "jobs",
		Short: "Manage scheduled backup jobs",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			store := scheduler.NewJSONStore(".backupctl/jobs.json")

			jobs, err := store.List(ctx)
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "%-24s %-15s %-15s %-10s %-10s\n", "ID", "DATABASE", "SCHEDULE", "FORMAT", "STATUS")

			for _, job := range jobs {
				schedule := job.Cron
				if schedule == "" {
					schedule = job.Interval
				}

				fmt.Fprintf(
					out,
					"%-24s %-15s %-15s %-10s %-10s\n", job.ID, job.DatabaseName, schedule, job.Format, job.Status,
				)
			}

			return nil
		},
	}

	cmd.AddCommand(newJobsStatusCommand())
	cmd.AddCommand(newJobsDeleteCommand())

	return cmd
}

func newJobsStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status <job-id>",
		Short: "Show scheduled job status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Second)
			defer cancel()

			store := scheduler.NewJSONStore(".backupctl/jobs.json")

			job, err := store.Get(ctx, args[0])
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "id: %s\n", job.ID)
			fmt.Fprintf(out, "database: %s\n", job.DatabaseName)
			fmt.Fprintf(out, "backup_type: %s\n", job.BackupType)
			fmt.Fprintf(out, "format: %s\n", job.Format)
			fmt.Fprintf(out, "status: %s\n", job.Status)

			if job.Cron != "" {
				fmt.Fprintf(out, "cron: %s\n", job.Cron)
			}

			if job.Interval != "" {
				fmt.Fprintf(out, "interval: %s\n", job.Interval)
			}

			if job.LastRun != nil {
				fmt.Fprintf(out, "last_run: %s\n", job.LastRun.Format(time.RFC3339))
			}

			if job.NextRun != nil {
				fmt.Fprintf(out, "next_run: %s\n", job.NextRun.Format(time.RFC3339))
			}

			if job.LogFile != "" {
				fmt.Fprintf(out, "log_file: %s\n", job.LogFile)
			}

			return nil
		},
	}
}

func newJobsDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <job-id>",
		Short: "Delete scheduled job",
		Args:  cobra.ExactArgs(1),
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
