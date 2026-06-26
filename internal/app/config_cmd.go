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
	var redacted bool

	cmd := &cobra.Command{
		Use:   "config",
		Short: "Print loaded config",
		Long:  "Load the selected config file and print a redacted operational summary.",
		Example: `  backupctl config -c configs/config.yaml
  backupctl config --config configs/config.s3.example.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigPrint(cmd, opts, configPath, jsonOutput)
		},
	}

	addConfigFlag(cmd, &configPath)
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Print config summary as JSON")
	cmd.Flags().BoolVar(&redacted, "redacted", true, "Redact secrets in output")
	cmd.AddCommand(newConfigPrintCommand(opts))
	cmd.AddCommand(newConfigValidateCommand(opts))

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

func newConfigPrintCommand(opts CLIOptions) *cobra.Command {
	var configPath string
	var jsonOutput bool
	var redacted bool

	cmd := &cobra.Command{
		Use:   "print",
		Short: "Print loaded config summary",
		Long:  "Load backupctl config and print a redacted operational summary.",
		Example: `  backupctl config print
    backupctl config print -c ./backupctl.yaml
    backupctl config print --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigPrint(cmd, opts, configPath, jsonOutput)
		},
	}

	addConfigFlag(cmd, &configPath)
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Print config summary as JSON")
	cmd.Flags().BoolVar(&redacted, "redacted", true, "Redact secrets in output")

	return cmd
}

func runConfigPrint(cmd *cobra.Command, opts CLIOptions, configPath string, jsonOutput bool) error {
	cfg, err := loadConfig(configPath)
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
}

func newConfigValidateCommand(opts CLIOptions) *cobra.Command {
	var configPath string
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate config file",
		Long:  "Validate backupctl config file structure without running backup, restore, or scheduler workflows.",
		Example: `  backupctl config validate
    backupctl config validate -c ./backupctl.yaml
    backupctl config validate --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigValidate(cmd, opts, configPath, jsonOutput)
		},
	}

	addConfigFlag(cmd, &configPath)
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Print validation result as JSON")

	return cmd
}

func runConfigValidate(cmd *cobra.Command, opts CLIOptions, configPath string, jsonOutput bool) error {
	if err := requireConfigPath(configPath); err != nil {
		return err
	}

	cfg, err := config.LoadFile(configPath)
	if err != nil {
		return WithHint(
			err,
			"pass --config <path>, set BACKUPCTL_CONFIG, or run backupctl init",
		)
	}

	if err := cfg.ValidateStructure(); err != nil {
		payload := map[string]any{
			"status": "failed",
			"path":   configPath,
			"error":  err.Error(),
		}

		if jsonOutput {
			_ = PrintJSON(cmd.OutOrStdout(), payload)
		}

		return WithHint(
			fmt.Errorf("invalid config: %w", err),
			"run backupctl init to create a starter config or compare with README examples",
		)
	}

	payload := map[string]any{
		"status": "success",
		"path":   configPath,
	}

	if jsonOutput {
		return PrintJSON(cmd.OutOrStdout(), payload)
	}

	if !opts.Quiet {
		PrintKV(cmd.OutOrStdout(), "Config valid", []KV{
			{Key: "path", Value: configPath},
		})
	}

	return nil
}
