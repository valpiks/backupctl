package app

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/spf13/cobra"
)

type initTemplateData struct {
	EnvFile string
}

func newInitCommand(opts CLIOptions) *cobra.Command {
	var database string
	var storage string
	var output string
	var force bool
	var envOutput string
	var writeEnv bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create a backupctl config file",
		Long:  "Create a starter backupctl config file for an installed binary workflow.",
		Example: `  backupctl init
  backupctl init --database postgres --storage local
  backupctl init --database mongo --storage s3
  backupctl init --output ./backupctl.yaml
  backupctl init --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if output == "" {
				return WithHint(
					fmt.Errorf("output path is required"),
					"pass --output <path> or set BACKUPCTL_CONFIG",
				)
			}

			if database != "postgres" && database != "mongo" {
				return WithHint(fmt.Errorf("unsupported database: %s", database), "use postgres or mongo")
			}

			if storage != "local" && storage != "s3" {
				return WithHint(fmt.Errorf("unsupported storage: %s", storage), "use local or s3")
			}

			if envOutput == "" {
				envOutput = filepath.Join(filepath.Dir(output), "backupctl.env")
			}

			if _, err := os.Stat(output); err == nil && !force {
				return WithHint(
					fmt.Errorf("config already exists: %s", output),
					"pass --force to overwrite or choose --output <path>",
				)
			}

			if writeEnv {
				if _, err := os.Stat(envOutput); err == nil && !force {
					return WithHint(
						fmt.Errorf("env file already exists: %s", envOutput),
						"pass --force to overwrite or choose --env-output <path>",
					)
				}
			}

			if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
				return fmt.Errorf("create config directory: %w", err)
			}

			raw, err := selectInitTemplate(database, storage)
			if err != nil {
				return err
			}

			templateData := initTemplateData{}
			if writeEnv {
				templateData.EnvFile = envOutput
			}

			rendered, err := renderTemplate(raw, templateData)
			if err != nil {
				return fmt.Errorf("render config template: %w", err)
			}

			if err := os.WriteFile(output, []byte(rendered), 0600); err != nil {
				return fmt.Errorf("write config: %w", err)
			}

			if writeEnv {
				if err := os.WriteFile(envOutput, []byte(envTemplate), 0600); err != nil {
					return fmt.Errorf("write env file: %w", err)
				}
			}

			if !opts.Quiet {
				rows := []KV{
					{Key: "config", Value: output},
					{Key: "database", Value: database},
					{Key: "storage", Value: storage},
				}
				if writeEnv {
					rows = append(rows, KV{Key: "env", Value: envOutput})
				}
				PrintKV(cmd.OutOrStdout(), "Config created", rows)

				fmt.Fprintln(cmd.OutOrStdout())
				fmt.Fprintln(cmd.OutOrStdout(), "Next:")
				fmt.Fprintf(cmd.OutOrStdout(), "  1. Edit %s\n", output)
				fmt.Fprintf(cmd.OutOrStdout(), "  2. Run backupctl doctor -c %s\n", output)
				fmt.Fprintf(cmd.OutOrStdout(), "  3. Run backupctl backup -c %s\n", output)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&database, "database", "postgres", "Database type: postgres or mongo")
	cmd.Flags().StringVar(&storage, "storage", "local", "Storage type: local or s3")
	cmd.Flags().StringVarP(&output, "output", "o", defaultConfigPath(), "Output config path")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing files")
	cmd.Flags().BoolVar(&writeEnv, "env", true, "Write starter env file")
	cmd.Flags().StringVar(&envOutput, "env-output", "", "Output env file path")

	return cmd
}

func renderTemplate(raw string, data initTemplateData) (string, error) {
	tpl, err := template.New("config").Parse(raw)
	if err != nil {
		return "", err
	}

	var out bytes.Buffer
	if err := tpl.Execute(&out, data); err != nil {
		return "", err
	}

	return out.String(), nil
}

func selectInitTemplate(database string, storage string) (string, error) {
	switch database + "/" + storage {
	case "postgres/local":
		return postgresLocalConfigTemplate, nil
	case "postgres/s3":
		return postgresS3ConfigTemplate, nil
	case "mongo/local":
		return mongoLocalConfigTemplate, nil
	case "mongo/s3":
		return mongoS3ConfigTemplate, nil
	default:
		return "", fmt.Errorf("unsupported init template: database=%s storage=%s", database, storage)
	}
}
