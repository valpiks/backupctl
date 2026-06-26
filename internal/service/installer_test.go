package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeRunner struct {
	calls []string
}

func (r *fakeRunner) Run(name string, args ...string) error {
	call := name
	for _, arg := range args {
		call += " " + arg
	}
	r.calls = append(r.calls, call)
	return nil
}

func TestInstallNoStartWritesFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	runner := &fakeRunner{}

	rendered, err := Install("darwin", InstallOptions{
		Name:       "backupctl-scheduler",
		BinaryPath: "/usr/local/bin/backupctl",
		ConfigPath: "/etc/backupctl/config.yaml",
		User:       true,
		Force:      true,
		Start:      false,
		Runner:     runner,
	})
	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}

	wantPath := filepath.Join(tmp, "Library", "LaunchAgents", "com.backupctl.scheduler.plist")
	if rendered.Path != wantPath {
		t.Fatalf("path = %q, want %q", rendered.Path, wantPath)
	}

	if rendered.Started {
		t.Fatal("Started = true, want false")
	}

	if len(runner.calls) != 0 {
		t.Fatalf("runner calls = %v, want none", runner.calls)
	}

	content, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if !strings.Contains(string(content), "<string>com.backupctl.scheduler</string>") {
		t.Fatalf("service file content missing label:\n%s", string(content))
	}
}

func TestRenderSystemd(t *testing.T) {
	rendered, err := Render("linux", InstallOptions{
		Name:       "backupctl-scheduler",
		BinaryPath: "/usr/local/bin/backupctl",
		ConfigPath: "/etc/backupctl/config.yaml",
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if rendered.Path != "/etc/systemd/system/backupctl-scheduler.service" {
		t.Fatalf("path = %q", rendered.Path)
	}

	for _, want := range []string{
		"ExecStart=/usr/local/bin/backupctl scheduler run --config /etc/backupctl/config.yaml",
		"Restart=always",
		"WorkingDirectory=",
		"WantedBy=multi-user.target",
	} {
		if !strings.Contains(rendered.Content, want) {
			t.Fatalf("content does not contain %q:\n%s", want, rendered.Content)
		}
	}
}

func TestRenderLaunchd(t *testing.T) {
	rendered, err := Render("darwin", InstallOptions{
		Name:       "backupctl-scheduler",
		BinaryPath: "/usr/local/bin/backupctl",
		ConfigPath: "/etc/backupctl/config.yaml",
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if rendered.Path != "~/Library/LaunchAgents/com.backupctl.scheduler.plist" {
		t.Fatalf("path = %q", rendered.Path)
	}

	for _, want := range []string{
		"<string>/usr/local/bin/backupctl</string>",
		"<string>scheduler</string>",
		"<string>run</string>",
		"<string>/etc/backupctl/config.yaml</string>",
		"<key>WorkingDirectory</key>",
		"<key>EnvironmentVariables</key>",
		"/opt/homebrew/opt/libpq/bin",
		"<key>KeepAlive</key>",
	} {
		if !strings.Contains(rendered.Content, want) {
			t.Fatalf("content does not contain %q:\n%s", want, rendered.Content)
		}
	}
}

func TestRenderLaunchdResolvesRelativePaths(t *testing.T) {
	tmp := t.TempDir()

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldwd); err != nil {
			t.Fatalf("Chdir() cleanup error = %v", err)
		}
	})

	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}

	rendered, err := Render("darwin", InstallOptions{
		Name:       "backupctl-scheduler",
		BinaryPath: "./backupctl",
		ConfigPath: "configs/config.yaml",
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	for _, want := range []string{
		"<string>" + filepath.Join(wd, "backupctl") + "</string>",
		"<string>" + filepath.Join(wd, "configs", "config.yaml") + "</string>",
		"<key>WorkingDirectory</key>",
		"<string>" + wd + "</string>",
	} {
		if !strings.Contains(rendered.Content, want) {
			t.Fatalf("content does not contain %q:\n%s", want, rendered.Content)
		}
	}
}

func TestRenderUnsupportedOS(t *testing.T) {
	_, err := Render("windows", InstallOptions{
		BinaryPath: "/usr/local/bin/backupctl",
		ConfigPath: "/etc/backupctl/config.yaml",
	})
	if err == nil {
		t.Fatal("Render() error = nil")
	}
}

func TestRenderDarwinRejectsSystemService(t *testing.T) {
	_, err := Render("darwin", InstallOptions{
		BinaryPath: "/usr/local/bin/backupctl",
		ConfigPath: "/etc/backupctl/config.yaml",
		System:     true,
	})
	if err == nil {
		t.Fatal("Render() error = nil")
	}

	if !strings.Contains(err.Error(), "system-level launchd service is not implemented yet") {
		t.Fatalf("error = %q", err)
	}
}

func TestStatusRejectsUserAndSystem(t *testing.T) {
	err := Status("linux", StatusOptions{
		User:   true,
		System: true,
		Runner: &fakeRunner{},
	})
	if err == nil {
		t.Fatal("Status() error = nil")
	}

	if !strings.Contains(err.Error(), "--user and --system cannot be used together") {
		t.Fatalf("error = %q", err)
	}
}

func TestUninstallRejectsUserAndSystem(t *testing.T) {
	err := Uninstall("linux", UninstallOptions{
		User:   true,
		System: true,
		Runner: &fakeRunner{},
	})
	if err == nil {
		t.Fatal("Uninstall() error = nil")
	}

	if !strings.Contains(err.Error(), "--user and --system cannot be used together") {
		t.Fatalf("error = %q", err)
	}
}

func TestStatusDarwinRejectsSystemService(t *testing.T) {
	err := Status("darwin", StatusOptions{
		System: true,
		Runner: &fakeRunner{},
	})
	if err == nil {
		t.Fatal("Status() error = nil")
	}

	if !strings.Contains(err.Error(), "system-level launchd service is not implemented yet") {
		t.Fatalf("error = %q", err)
	}
}

func TestUninstallDarwinRejectsSystemService(t *testing.T) {
	err := Uninstall("darwin", UninstallOptions{
		System: true,
		Runner: &fakeRunner{},
	})
	if err == nil {
		t.Fatal("Uninstall() error = nil")
	}

	if !strings.Contains(err.Error(), "system-level launchd service is not implemented yet") {
		t.Fatalf("error = %q", err)
	}
}

func TestStartLaunchdService(t *testing.T) {
	runner := &fakeRunner{}

	err := startLaunchdService(InstallOptions{
		Name:   "backupctl-scheduler",
		User:   true,
		Runner: runner,
	}, "/tmp/com.backupctl.scheduler.plist")
	if err != nil {
		t.Fatalf("startLaunchdService() error = %v", err)
	}

	got := strings.Join(runner.calls, "\n")

	for _, want := range []string{
		"launchctl bootout gui/",
		"launchctl bootstrap gui/",
		"launchctl enable gui/",
		"launchctl kickstart -k gui/",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("calls do not contain %q:\n%s", want, got)
		}
	}
}

func TestStatusLaunchdService(t *testing.T) {
	runner := &fakeRunner{}

	err := Status("darwin", StatusOptions{
		Name:   "backupctl-scheduler",
		User:   true,
		Runner: runner,
	})
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}

	got := strings.Join(runner.calls, "\n")

	if !strings.Contains(got, "launchctl print gui/") {
		t.Fatalf("calls do not contain launchctl print:\n%s", got)
	}

	if !strings.Contains(got, "/com.backupctl.scheduler") {
		t.Fatalf("calls do not contain launchd label:\n%s", got)
	}
}

func TestStatusSystemdService(t *testing.T) {
	runner := &fakeRunner{}

	err := Status("linux", StatusOptions{
		Name:   "backupctl-scheduler",
		System: true,
		Runner: runner,
	})
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}

	got := strings.Join(runner.calls, "\n")
	want := "systemctl status backupctl-scheduler.service"

	if !strings.Contains(got, want) {
		t.Fatalf("calls do not contain %q:\n%s", want, got)
	}
}

func TestStatusSystemdUserService(t *testing.T) {
	runner := &fakeRunner{}

	err := Status("linux", StatusOptions{
		Name:   "backupctl-scheduler",
		User:   true,
		Runner: runner,
	})
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}

	got := strings.Join(runner.calls, "\n")
	want := "systemctl --user status backupctl-scheduler.service"

	if !strings.Contains(got, want) {
		t.Fatalf("calls do not contain %q:\n%s", want, got)
	}
}

func TestUninstallLaunchdService(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	plistPath := filepath.Join(tmp, "Library", "LaunchAgents", "com.backupctl.scheduler.plist")
	if err := os.MkdirAll(filepath.Dir(plistPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	if err := os.WriteFile(plistPath, []byte("plist"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	runner := &fakeRunner{}

	err := Uninstall("darwin", UninstallOptions{
		Name:   "backupctl-scheduler",
		User:   true,
		Runner: runner,
	})
	if err != nil {
		t.Fatalf("Uninstall() error = %v", err)
	}

	if _, err := os.Stat(plistPath); !os.IsNotExist(err) {
		t.Fatalf("plist still exists or stat failed with unexpected error: %v", err)
	}

	got := strings.Join(runner.calls, "\n")

	if !strings.Contains(got, "launchctl bootout gui/") {
		t.Fatalf("calls do not contain launchctl bootout:\n%s", got)
	}

	if !strings.Contains(got, "/com.backupctl.scheduler") {
		t.Fatalf("calls do not contain launchd label:\n%s", got)
	}
}

func TestUninstallSystemdUserService(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	unitPath := filepath.Join(tmp, ".config", "systemd", "user", "backupctl-scheduler.service")
	if err := os.MkdirAll(filepath.Dir(unitPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	if err := os.WriteFile(unitPath, []byte("unit"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	runner := &fakeRunner{}

	err := Uninstall("linux", UninstallOptions{
		Name:   "backupctl-scheduler",
		User:   true,
		Runner: runner,
	})
	if err != nil {
		t.Fatalf("Uninstall() error = %v", err)
	}

	if _, err := os.Stat(unitPath); !os.IsNotExist(err) {
		t.Fatalf("unit still exists or stat failed with unexpected error: %v", err)
	}

	got := strings.Join(runner.calls, "\n")

	for _, want := range []string{
		"systemctl --user stop backupctl-scheduler.service",
		"systemctl --user disable backupctl-scheduler.service",
		"systemctl --user daemon-reload",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("calls do not contain %q:\n%s", want, got)
		}
	}
}

func TestRestartSystemdUserService(t *testing.T) {
	runner := &fakeRunner{}

	err := Restart("linux", RestartOptions{
		Name:   "backupctl-scheduler",
		User:   true,
		Runner: runner,
	})
	if err != nil {
		t.Fatalf("Restart() error = %v", err)
	}

	got := strings.Join(runner.calls, "\n")
	want := "systemctl --user restart backupctl-scheduler.service"
	if !strings.Contains(got, want) {
		t.Fatalf("calls do not contain %q:\n%s", want, got)
	}
}

func TestLogsSystemdUserService(t *testing.T) {
	runner := &fakeRunner{}

	err := Logs("linux", LogsOptions{
		Name:   "backupctl-scheduler",
		User:   true,
		Tail:   25,
		Runner: runner,
	})
	if err != nil {
		t.Fatalf("Logs() error = %v", err)
	}

	got := strings.Join(runner.calls, "\n")
	want := "journalctl --user -u backupctl-scheduler.service -n 25 --no-pager"
	if !strings.Contains(got, want) {
		t.Fatalf("calls do not contain %q:\n%s", want, got)
	}
}
