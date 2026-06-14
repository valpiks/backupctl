package app

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/valpiks/backupctl/internal/config"
)

func newConfigCommand(opts CLIOptions) *cobra.Command {
	var configPath string
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "config",
		Short: "Print loaded config",
		Long:  "Load the selected config file and print a redacted operational summary.",
		Example: `  backupctl config -c configs/config.yaml
  backupctl config --config configs/config.s3.example.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configPath)
			if err != nil {
				return err
			}

			log := commandLogger(cfg, opts)
			log.Info("config loaded", "path", configPath)

			payload := configPayload(cfg, configPath)
			if jsonOutput {
				return PrintJSON(cmd.OutOrStdout(), payload)
			}

			if !opts.Quiet {
				PrintKV(cmd.OutOrStdout(), "Config loaded", []KV{
					{Key: "path", Value: payload["path"]},
					{Key: "database", Value: payload["database"]},
					{Key: "storage", Value: payload["storage"]},
					{Key: "backup", Value: payload["backup"]},
					{Key: "encryption", Value: payload["encryption"]},
					{Key: "logging", Value: payload["logging"]},
				})
			}

			log.Info("config printed", "db", cfg.Database.ActiveDatabaseName())
			return nil
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "configs/config.yaml", "Path to config file")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Print config summary as JSON")

	return cmd
}

func configPayload(cfg *config.Config, path string) map[string]string {
	return map[string]string{
		"status":     "success",
		"path":       path,
		"database":   fmt.Sprintf("%s/%s", cfg.Database.Type, cfg.Database.ActiveDatabaseName()),
		"storage":    storageSummary(cfg),
		"backup":     backupSummary(cfg),
		"encryption": encryptionStatus(cfg),
		"logging":    cfg.Logging.Level,
	}
}

func storageSummary(cfg *config.Config) string {
	switch cfg.Storage.Type {
	case "local":
		return fmt.Sprintf("local %s", cfg.Storage.Local.Path)
	case "s3":
		return fmt.Sprintf("s3 %s", cfg.Storage.S3.Bucket)
	default:
		return cfg.Storage.Type
	}
}

func backupSummary(cfg *config.Config) string {
	parts := []string{cfg.Backup.Type}
	if cfg.Backup.Compression != "" {
		parts = append(parts, cfg.Backup.Compression)
	}
	return strings.Join(parts, ", ")
}

func encryptionStatus(cfg *config.Config) string {
	if encryptionEnabled(cfg) {
		return "enabled"
	}
	return "disabled"
}
