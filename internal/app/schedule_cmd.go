package app

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/valpiks/backupctl/internal/config"
	"github.com/valpiks/backupctl/internal/scheduler"
)

func newScheduleCommand() *cobra.Command {
	var configPath string
	var cronExpr string
	var interval string
	var format string

	cmd := &cobra.Command{
		Use:   "schedule",
		Short: "Create scheduled backup job",
		RunE: func(cmd *cobra.Command, args []string) error {
			if cronExpr == "" && interval == "" {
				return fmt.Errorf("either --cron or --interval is required")
			}

			if cronExpr != "" && interval != "" {
				return fmt.Errorf("--cron and --interval cannot be used together")
			}

			if interval != "" {
				if _, err := time.ParseDuration(interval); err != nil {
					return fmt.Errorf("invalid interval: %w", err)
				}
			}

			if format == "" {
				format = "plain"
			}

			if format != "plain" && format != "custom" {
				return fmt.Errorf("unsupported format: %s", format)
			}

			cfg, err := config.Load(configPath)
			if err != nil {
				return err
			}

			store := scheduler.NewJSONStore(".backupctl/jobs.json")

			now := time.Now().UTC()

			job := &scheduler.Job{
				ID:           fmt.Sprintf("job_%s", now.Format("20060102_150405")),
				Cron:         cronExpr,
				Interval:     interval,
				DatabaseName: cfg.Database.ActiveDatabaseName(),
				BackupType:   cfg.Backup.Type,
				Format:       format,
				Status:       "scheduled",
			}

			if cfg.Backup.Scheduler != nil {
				job.LogFile = cfg.Backup.Scheduler.LogFile
			}

			if err := store.Create(context.Background(), job); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "job created: %s\n", job.ID)
			return nil
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "configs/config.yaml", "Path to config file")
	cmd.Flags().StringVar(&cronExpr, "cron", "", "Cron expression")
	cmd.Flags().StringVar(&interval, "interval", "", "Backup interval, for example 24h or 30m")
	cmd.Flags().StringVar(&format, "format", "plain", "Backup format: plain or custom")

	return cmd
}
