package service

import (
	"fmt"
	"html"
	"os"
	"os/exec"
	"strings"
)

func launchdPath(opts InstallOptions) string {
	return fmt.Sprintf("~/Library/LaunchAgents/%s.plist", launchdLabel(opts))
}

func renderLaunchdPlist(opts InstallOptions) string {
	label := launchdLabel(opts)

	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>%s</string>
	<key>ProgramArguments</key>
	<array>
		<string>%s</string>
		<string>scheduler</string>
		<string>run</string>
		<string>--config</string>
		<string>%s</string>
	</array>
	<key>WorkingDirectory</key>
	<string>%s</string>
	<key>EnvironmentVariables</key>
	<dict>
		<key>PATH</key>
		<string>/opt/homebrew/bin:/opt/homebrew/sbin:/opt/homebrew/opt/libpq/bin:/usr/local/bin:/usr/local/sbin:/usr/bin:/bin:/usr/sbin:/sbin</string>
	</dict>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<true/>
</dict>
</plist>
`, html.EscapeString(label), html.EscapeString(opts.BinaryPath), html.EscapeString(opts.ConfigPath), html.EscapeString(opts.WorkDir))
}

func launchdLabel(opts InstallOptions) string {
	name := opts.Name

	if strings.HasPrefix(name, "com.backupctl.") {
		return name
	}

	name = strings.TrimPrefix(name, "backupctl-")
	return fmt.Sprintf("com.backupctl.%s", name)
}

func startLaunchdService(opts InstallOptions, path string) error {
	if opts.System {
		return fmt.Errorf("system-level launchd install is not implemented yet")
	}

	runner := commandRunner(opts)
	label := launchdLabel(opts)
	domain := fmt.Sprintf("gui/%d", os.Getuid())
	target := fmt.Sprintf("%s/%s", domain, label)

	runLaunchdBootout(runner, target)

	if err := runner.Run("launchctl", "bootstrap", domain, path); err != nil {
		return fmt.Errorf("launchctl bootstrap: %w", err)
	}

	if err := runner.Run("launchctl", "enable", domain+"/"+label); err != nil {
		return fmt.Errorf("launchctl enable: %w", err)
	}

	if err := runner.Run("launchctl", "kickstart", "-k", domain+"/"+label); err != nil {
		return fmt.Errorf("launchctl kickstart: %w", err)
	}

	return nil
}

func runLaunchdBootout(runner CommandRunner, target string) {
	switch runner.(type) {
	case ExecRunner:
		_ = exec.Command("launchctl", "bootout", target).Run()
	default:
		_ = runner.Run("launchctl", "bootout", target)
	}
}

func statusLaunchdService(opts StatusOptions) error {
	if opts.System {
		return fmt.Errorf("system-level launchd status is not implemented yet; use --user")
	}

	runner := opts.Runner
	label := launchdLabel(InstallOptions{Name: opts.Name})
	domain := fmt.Sprintf("gui/%d", os.Getuid())

	if err := runner.Run("launchctl", "print", domain+"/"+label); err != nil {
		return fmt.Errorf("launchctl print: %w", err)
	}

	return nil
}

func uninstallLaunchdService(opts UninstallOptions) error {
	if opts.System {
		return fmt.Errorf("system-level launchd uninstall is not implemented yet; use --user")
	}

	runner := opts.Runner
	label := launchdLabel(InstallOptions{Name: opts.Name})
	domain := fmt.Sprintf("gui/%d", os.Getuid())
	target := fmt.Sprintf("%s/%s", domain, label)

	_ = runner.Run("launchctl", "bootout", target)

	path := expandHome(fmt.Sprintf("~/Library/LaunchAgents/%s.plist", label))
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove launchd plist: %w", err)
	}

	return nil
}

func restartLaunchdService(opts RestartOptions) error {
	if opts.System {
		return fmt.Errorf("system-level launchd restart is not implemented yet; use --user")
	}

	label := launchdLabel(InstallOptions{Name: opts.Name})
	domain := fmt.Sprintf("gui/%d", os.Getuid())
	target := fmt.Sprintf("%s/%s", domain, label)

	if err := opts.Runner.Run("launchctl", "kickstart", "-k", target); err != nil {
		return fmt.Errorf("launchctl kickstart: %w", err)
	}

	return nil
}

func logsLaunchdService(opts LogsOptions) error {
	if opts.System {
		return fmt.Errorf("system-level launchd logs are not implemented yet; use --user")
	}

	label := launchdLabel(InstallOptions{Name: opts.Name})
	if err := opts.Runner.Run("log", "show", "--predicate", fmt.Sprintf(`process == "backupctl" OR subsystem == "%s"`, label), "--last", "1h", "--style", "compact"); err != nil {
		return fmt.Errorf("log show: %w", err)
	}

	return nil
}
