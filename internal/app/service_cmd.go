package app

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/valpiks/backupctl/internal/service"
)

func newServiceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "service",
		Aliases: []string{"svc"},
		Short:   "Manage backupctl background service",
		Long:    "Install, uninstall, and inspect the backupctl scheduler as a systemd or launchd background service.",
		Example: `  backupctl service install --user --config configs/config.yaml
  backupctl service status --user
  backupctl service uninstall --user`,
	}

	cmd.AddCommand(newServiceInstallCommand())
	cmd.AddCommand(newServiceUninstallCommand())
	cmd.AddCommand(newServiceStatusCommand())

	return cmd
}

func newServiceInstallCommand() *cobra.Command {
	var configPath string
	var binaryPath string
	var name string
	var dryRun bool
	var user bool
	var system bool
	var force bool
	var noStart bool

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install backupctl scheduler as a background service",
		Long:  "Install backupctl scheduler as a launchd service on macOS or a systemd service on Linux.",
		Example: `  backupctl service install --user --config configs/config.yaml
  backupctl service install --system --config /etc/backupctl/config.yaml --binary /usr/local/bin/backupctl
  backupctl service install --user --dry-run`,
		RunE: func(cmd *cobra.Command, args []string) error {
			rendered, err := service.InstallCurrentOS(
				service.InstallOptions{
					Name:       name,
					BinaryPath: binaryPath,
					ConfigPath: configPath,
					User:       user,
					System:     system,
					DryRun:     dryRun,
					Force:      force,
					Start:      !noStart,
				})
			if err != nil {
				return err
			}

			if dryRun {
				fmt.Fprintf(cmd.OutOrStdout(), "# %s\n", rendered.Path)
				fmt.Fprint(cmd.OutOrStdout(), rendered.Content)
				return nil
			}

			fmt.Fprintf(cmd.OutOrStdout(), "service file installed: %s\n", rendered.Path)

			if rendered.Started {
				fmt.Fprintln(cmd.OutOrStdout(), "service started")
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "service not started")
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&configPath, "config", "configs/config.yaml", "Path to config file")
	cmd.Flags().StringVar(&binaryPath, "binary", defaultServiceBinaryPath(), "Path to backupctl binary")
	cmd.Flags().StringVar(&name, "name", service.DefaultName, "Service name")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print service file without installing it")
	cmd.Flags().BoolVar(&user, "user", false, "Install user-level service")
	cmd.Flags().BoolVar(&system, "system", false, "Install system-level service")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing service file")
	cmd.Flags().BoolVar(&noStart, "no-start", false, "Install service file without starting it")

	return cmd
}

func defaultServiceBinaryPath() string {
	path, err := os.Executable()
	if err != nil || path == "" {
		return "/usr/local/bin/backupctl"
	}

	return path
}

func newServiceUninstallCommand() *cobra.Command {
	var name string
	var user bool
	var system bool

	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall backupctl scheduler background service",
		Long:  "Remove the installed backupctl scheduler service for the selected user or system scope.",
		Example: `  backupctl service uninstall --user
  backupctl service uninstall --system
  backupctl svc uninstall --user`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := service.UninstallCurrentOS(service.UninstallOptions{
				Name:   name,
				User:   user,
				System: system,
			}); err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), "service uninstalled")
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", service.DefaultName, "Service name")
	cmd.Flags().BoolVar(&user, "user", false, "Uninstall user-level service")
	cmd.Flags().BoolVar(&system, "system", false, "Uninstall system-level service")

	return cmd
}

func newServiceStatusCommand() *cobra.Command {
	var name string
	var user bool
	var system bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show backupctl scheduler service status",
		Long:  "Show status for the installed backupctl scheduler service in the selected user or system scope.",
		Example: `  backupctl service status --user
  backupctl service status --system
  backupctl svc status --user`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return service.StatusCurrentOS(service.StatusOptions{
				Name:   name,
				User:   user,
				System: system,
			})
		},
	}

	cmd.Flags().StringVar(&name, "name", service.DefaultName, "Service name")
	cmd.Flags().BoolVar(&user, "user", false, "Show user-level service status")
	cmd.Flags().BoolVar(&system, "system", false, "Show system-level service status")

	return cmd
}
