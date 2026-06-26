package service

import (
	"fmt"
	"os"
)

func systemdPath(opts InstallOptions) string {
	if opts.User {
		return fmt.Sprintf("~/.config/systemd/user/%s.service", opts.Name)
	}

	return fmt.Sprintf("/etc/systemd/system/%s.service", opts.Name)
}

func renderSystemdUnit(opts InstallOptions) string {
	return fmt.Sprintf(`[Unit]
Description=backupctl scheduler
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=%s scheduler run --config %s
Restart=always
RestartSec=5
WorkingDirectory=%s

[Install]
WantedBy=multi-user.target
`, opts.BinaryPath, opts.ConfigPath, opts.WorkDir)
}

func startSystemdService(opts InstallOptions) error {
	serviceName := opts.Name + ".service"

	args := []string{}
	if opts.User {
		args = append(args, "--user")
	}

	runner := commandRunner(opts)

	if err := runner.Run("systemctl", append(args, "daemon-reload")...); err != nil {
		return fmt.Errorf("systemctl daemon-reload: %w", err)
	}

	if err := runner.Run("systemctl", append(args, "enable", serviceName)...); err != nil {
		return fmt.Errorf("systemctl enable: %w", err)
	}

	if err := runner.Run("systemctl", append(args, "start", serviceName)...); err != nil {
		return fmt.Errorf("systemctl start: %w", err)
	}

	return nil
}

func statusSystemdService(opts StatusOptions) error {
	serviceName := opts.Name + ".service"

	args := []string{}
	if opts.User {
		args = append(args, "--user")
	}

	if err := opts.Runner.Run("systemctl", append(args, "status", serviceName)...); err != nil {
		return fmt.Errorf("systemctl status: %w", err)
	}

	return nil
}

func uninstallSystemdService(opts UninstallOptions) error {
	serviceName := opts.Name + ".service"

	args := []string{}
	if opts.User {
		args = append(args, "--user")
	}

	runner := opts.Runner

	_ = runner.Run("systemctl", append(args, "stop", serviceName)...)
	_ = runner.Run("systemctl", append(args, "disable", serviceName)...)

	path := systemdPath(InstallOptions{
		Name:   opts.Name,
		User:   opts.User,
		System: opts.System,
	})

	if err := os.Remove(expandHome(path)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove systemd unit: %w", err)
	}

	if err := runner.Run("systemctl", append(args, "daemon-reload")...); err != nil {
		return fmt.Errorf("systemctl daemon-reload: %w", err)
	}

	return nil
}

func restartSystemdService(opts RestartOptions) error {
	serviceName := opts.Name + ".service"

	args := []string{}
	if opts.User {
		args = append(args, "--user")
	}

	if err := opts.Runner.Run("systemctl", append(args, "restart", serviceName)...); err != nil {
		return fmt.Errorf("systemctl restart: %w", err)
	}

	return nil
}

func logsSystemdService(opts LogsOptions) error {
	serviceName := opts.Name + ".service"

	args := []string{}
	if opts.User {
		args = append(args, "--user")
	}

	args = append(args, "-u", serviceName, "-n", fmt.Sprintf("%d", opts.Tail), "--no-pager")
	if err := opts.Runner.Run("journalctl", args...); err != nil {
		return fmt.Errorf("journalctl logs: %w", err)
	}

	return nil
}
