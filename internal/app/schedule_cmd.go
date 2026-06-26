package app

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/valpiks/backupctl/internal/scheduler"
)

func newScheduleCommand(opts CLIOptions) *cobra.Command {
	var configPath string
	var cronExpr string
	var interval string
	var format string

	cmd := &cobra.Command{
		Use:   "schedule (--cron <expr> | --interval <duration>)",
		Short: "Create scheduled backup job",
		Long:  "Create a scheduled backup job using a cron expression or fixed interval.",
		Example: `  backupctl schedule -c configs/config.yaml --interval 24h
  backupctl schedule -c configs/config.yaml --cron "0 2 * * *"
  backupctl schedule -c configs/config.yaml --interval 12h --format custom`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if cronExpr == "" && interval == "" {
				return WithHint(fmt.Errorf("either --cron or --interval is required"), "example: backupctl schedule --interval 24h")
			}

			if cronExpr != "" && interval != "" {
				return WithHint(fmt.Errorf("--cron and --interval cannot be used together"), "choose one schedule mode")
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
				return WithHint(fmt.Errorf("unsupported format: %s", format), "use --format plain or --format custom")
			}
			cfg, err := loadConfig(configPath)
			if err != nil {
				return err
			}

			store := scheduler.NewJSONStore(".backupctl/jobs.json")

			now := time.Now().UTC()

			job := &scheduler.Job{
				ID:              fmt.Sprintf("job_%s", now.Format("20060102_150405")),
				Cron:            cronExpr,
				Interval:        interval,
				DatabaseName:    cfg.Database.ActiveDatabaseName(),
				BackupType:      cfg.Backup.Type,
				Format:          format,
				Status:          "scheduled",
				Disabled:        false,
				MissedRunPolicy: "run_once",
				ConfigPath:      configPath,
				CreatedAt:       now,
				UpdatedAt:       now,
			}

			if cfg.Backup.Scheduler != nil {
				job.LogFile = cfg.Backup.Scheduler.LogFile
			}

			if err := store.Create(context.Background(), job); err != nil {
				return err
			}

			if !opts.Quiet {
				PrintKV(cmd.OutOrStdout(), "Job created", []KV{
					{Key: "id", Value: job.ID},
					{Key: "database", Value: job.DatabaseName},
					{Key: "type", Value: job.BackupType},
					{Key: "format", Value: job.Format},
					{Key: "schedule", Value: firstNonEmpty(job.Cron, job.Interval)},
					{Key: "status", Value: job.Status},
				})
			}
			return nil
		},
	}

	addConfigFlag(cmd, &configPath)
	cmd.Flags().StringVar(&cronExpr, "cron", "", "Cron expression")
	cmd.Flags().StringVar(&interval, "interval", "", "Backup interval, for example 24h or 30m")
	cmd.Flags().StringVar(&format, "format", "plain", "Backup format: plain or custom")
	_ = cmd.RegisterFlagCompletionFunc("format", func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		return []string{"plain", "custom"}, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
