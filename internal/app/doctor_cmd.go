package app

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/valpiks/backupctl/internal/config"
	dbfactory "github.com/valpiks/backupctl/internal/database/factory"
	"github.com/valpiks/backupctl/internal/secrets"
	storagefactory "github.com/valpiks/backupctl/internal/storage/factory"
)

type DoctorCheck struct {
	Status  string `json:"status"`
	Check   string `json:"check"`
	Details string `json:"details,omitempty"`
}

func newDoctorCommand(opts CLIOptions) *cobra.Command {
	var configPath string
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Run environment and configuration checks",
		Long:  "Run configuration, database, storage, and native tool checks for the selected backup setup.",
		Example: `  backupctl doctor -c configs/config.yaml
  backupctl doctor --config configs/config.mongo.example.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
			defer cancel()
			out := cmd.OutOrStdout()
			checks := make([]DoctorCheck, 0, 8)

			cfg, err := config.Load(configPath)
			if err != nil {
				return err
			}
			knownSecrets := cfg.KnownSecrets()

			checks = append(checks, DoctorCheck{Status: "OK", Check: "config loaded", Details: configPath})

			driver, err := dbfactory.NewDriver(cfg.Database)
			if err != nil {
				checks = append(checks, DoctorCheck{Status: "FAIL", Check: "database driver", Details: secrets.Redact(err.Error(), knownSecrets)})
				_ = printDoctorChecks(out, checks, jsonOutput, opts.Color)
				return redactError(err, knownSecrets)
			}
			checks = append(checks, DoctorCheck{Status: "OK", Check: "database driver", Details: cfg.Database.Type})

			if err := driver.Ping(ctx); err != nil {
				checks = append(checks, DoctorCheck{Status: "FAIL", Check: "database ping", Details: secrets.Redact(err.Error(), knownSecrets)})
				_ = printDoctorChecks(out, checks, jsonOutput, opts.Color)
				return redactError(err, knownSecrets)
			}
			checks = append(checks, DoctorCheck{Status: "OK", Check: "database ping", Details: cfg.Database.ActiveDatabaseName()})

			if _, err := storagefactory.NewStorage(cfg.Storage); err != nil {
				checks = append(checks, DoctorCheck{Status: "FAIL", Check: "storage", Details: secrets.Redact(err.Error(), knownSecrets)})
				_ = printDoctorChecks(out, checks, jsonOutput, opts.Color)
				return redactError(err, knownSecrets)
			}
			checks = append(checks, DoctorCheck{Status: "OK", Check: "storage", Details: cfg.Storage.Type})

			dbTools := requiredDatabaseTools(cfg.Database.Type)

			if len(dbTools) == 0 {
				return WithHint(fmt.Errorf("unknown db type: %s", cfg.Database.Type), "set database.type to postgres or mongo")
			}

			for _, tool := range dbTools {
				path, err := exec.LookPath(tool)
				if err != nil {
					checks = append(checks, DoctorCheck{Status: "FAIL", Check: "tool: " + tool, Details: "not found in PATH"})
					_ = printDoctorChecks(out, checks, jsonOutput, opts.Color)
					return WithHint(err, "install PostgreSQL client tools or MongoDB Database Tools and make sure they are in PATH")
				}

				details := path
				if version := toolVersion(ctx, tool); version != "" {
					details = fmt.Sprintf("%s (%s)", path, version)
				}
				checks = append(checks, DoctorCheck{Status: "OK", Check: "tool: " + tool, Details: details})
			}

			if err := printDoctorChecks(out, checks, jsonOutput, opts.Color); err != nil {
				return err
			}

			if !jsonOutput && !opts.Quiet {
				fmt.Fprintln(out, "doctor checks passed")
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "configs/config.yaml", "Path to config file")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Print doctor checks as JSON")

	return cmd
}

func printDoctorChecks(out io.Writer, checks []DoctorCheck, jsonOutput bool, colorMode string) error {
	if jsonOutput {
		return PrintJSON(out, checks)
	}

	color := colorEnabled(colorMode, out)
	rows := make([][]string, 0, len(checks))
	for _, check := range checks {
		rows = append(rows, []string{colorStatus(check.Status, color), check.Check, check.Details})
	}
	PrintTable(out, []string{"Status", "Check", "Details"}, rows)
	return nil
}

func toolVersion(ctx context.Context, tool string) string {
	out, err := exec.CommandContext(ctx, tool, "--version").CombinedOutput()
	if err != nil {
		return ""
	}

	line := strings.TrimSpace(string(out))
	if idx := strings.IndexByte(line, '\n'); idx >= 0 {
		line = line[:idx]
	}

	return line
}

func requiredDatabaseTools(databaseType string) []string {
	switch databaseType {
	case "postgres":
		return []string{"psql", "pg_dump"}
	case "mongo":
		return []string{"mongosh", "mongodump", "mongorestore"}
	default:
		return []string{}
	}
}
