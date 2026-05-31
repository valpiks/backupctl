package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	DefaultName = "backupctl-scheduler"
)

type InstallOptions struct {
	Name       string
	BinaryPath string
	ConfigPath string
	WorkDir    string
	User       bool
	System     bool
	DryRun     bool
	Force      bool
	Start      bool
	Runner     CommandRunner
}

type StatusOptions struct {
	Name   string
	User   bool
	System bool
	Runner CommandRunner
}

type UninstallOptions struct {
	Name   string
	User   bool
	System bool
	Runner CommandRunner
}

type RenderedService struct {
	Path    string
	Content string
	Started bool
}

type ExecRunner struct{}

type CommandRunner interface {
	Run(name string, args ...string) error
}

func (ExecRunner) Run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func commandRunner(opts InstallOptions) CommandRunner {
	if opts.Runner != nil {
		return opts.Runner
	}

	return ExecRunner{}
}

func RenderCurrentOS(opts InstallOptions) (RenderedService, error) {
	return Render(runtime.GOOS, opts)
}

func Render(goos string, opts InstallOptions) (RenderedService, error) {
	var err error
	opts, err = prepareInstallOptions(opts)
	if err != nil {
		return RenderedService{}, err
	}

	opts = withPlatformDefaults(goos, opts)

	if err := validateOptions(opts); err != nil {
		return RenderedService{}, err
	}

	if err := validatePlatformScope(goos, opts.User, opts.System); err != nil {
		return RenderedService{}, err
	}

	switch goos {
	case "linux":
		return RenderedService{
			Path:    systemdPath(opts),
			Content: renderSystemdUnit(opts),
		}, nil
	case "darwin":
		return RenderedService{
			Path:    launchdPath(opts),
			Content: renderLaunchdPlist(opts),
		}, nil
	default:
		return RenderedService{}, fmt.Errorf("service install is not supported on %s", goos)
	}
}

func withPlatformDefaults(goos string, opts InstallOptions) InstallOptions {
	if opts.User || opts.System {
		return opts
	}

	if goos == "darwin" {
		opts.User = true
		return opts
	}

	opts.System = true
	return opts
}

func withDefaults(opts InstallOptions) InstallOptions {
	if opts.Name == "" {
		opts.Name = DefaultName
	}

	if opts.Runner == nil {
		opts.Runner = ExecRunner{}
	}

	return opts
}

func prepareInstallOptions(opts InstallOptions) (InstallOptions, error) {
	opts = withDefaults(opts)

	if opts.WorkDir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return InstallOptions{}, fmt.Errorf("get working directory: %w", err)
		}
		opts.WorkDir = wd
	}

	var err error

	opts.BinaryPath, err = absPathIfSet(opts.BinaryPath)
	if err != nil {
		return InstallOptions{}, fmt.Errorf("resolve binary path: %w", err)
	}

	opts.ConfigPath, err = absPathIfSet(opts.ConfigPath)
	if err != nil {
		return InstallOptions{}, fmt.Errorf("resolve config path: %w", err)
	}

	opts.WorkDir, err = absPathIfSet(opts.WorkDir)
	if err != nil {
		return InstallOptions{}, fmt.Errorf("resolve working directory: %w", err)
	}

	return opts, nil
}

func absPathIfSet(path string) (string, error) {
	if path == "" || filepath.IsAbs(path) {
		return path, nil
	}

	return filepath.Abs(path)
}

func validateOptions(opts InstallOptions) error {
	if opts.Name == "" {
		return fmt.Errorf("service name is required")
	}

	if opts.BinaryPath == "" {
		return fmt.Errorf("binary path is required")
	}

	if opts.ConfigPath == "" {
		return fmt.Errorf("config path is required")
	}

	if opts.User && opts.System {
		return fmt.Errorf("--user and --system cannot be used together")
	}

	return nil
}

func InstallCurrentOS(opts InstallOptions) (RenderedService, error) {
	return Install(runtime.GOOS, opts)
}

func Install(goos string, opts InstallOptions) (RenderedService, error) {
	var err error
	opts, err = prepareInstallOptions(opts)
	if err != nil {
		return RenderedService{}, err
	}

	opts = withPlatformDefaults(goos, opts)

	rendered, err := Render(goos, opts)
	if err != nil {
		return RenderedService{}, err
	}

	if opts.DryRun {
		return rendered, nil
	}

	path := expandHome(rendered.Path)

	if _, err := os.Stat(path); err == nil && !opts.Force {
		return RenderedService{}, fmt.Errorf("service file already exists: %s; use --force to overwrite", path)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return RenderedService{}, err
	}

	if err := os.WriteFile(path, []byte(rendered.Content), 0o644); err != nil {
		return RenderedService{}, err
	}

	rendered.Path = path

	if opts.Start {
		if err := startService(goos, opts, rendered.Path); err != nil {
			return RenderedService{}, err
		}

		rendered.Started = true
	}

	return rendered, nil
}

func startService(goos string, opts InstallOptions, path string) error {
	switch goos {
	case "linux":
		return startSystemdService(opts)
	case "darwin":
		return startLaunchdService(opts, path)
	default:
		return fmt.Errorf("service start is not supported on %s", goos)
	}
}

func expandHome(path string) string {
	if !strings.HasPrefix(path, "~/") {
		return path
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	return filepath.Join(home, strings.TrimPrefix(path, "~/"))
}

func StatusCurrentOS(opts StatusOptions) error {
	return Status(runtime.GOOS, opts)
}

func Status(goos string, opts StatusOptions) error {
	if opts.Name == "" {
		opts.Name = DefaultName
	}

	if opts.Runner == nil {
		opts.Runner = ExecRunner{}
	}

	if err := validateServiceScope(opts.User, opts.System); err != nil {
		return err
	}

	if err := validatePlatformScope(goos, opts.User, opts.System); err != nil {
		return err
	}

	switch goos {
	case "linux":
		return statusSystemdService(opts)
	case "darwin":
		return statusLaunchdService(opts)
	default:
		return fmt.Errorf("service status is not supported on %s", goos)
	}
}

func UninstallCurrentOS(opts UninstallOptions) error {
	return Uninstall(runtime.GOOS, opts)
}

func Uninstall(goos string, opts UninstallOptions) error {
	if opts.Name == "" {
		opts.Name = DefaultName
	}

	if opts.Runner == nil {
		opts.Runner = ExecRunner{}
	}

	if err := validateServiceScope(opts.User, opts.System); err != nil {
		return err
	}

	if err := validatePlatformScope(goos, opts.User, opts.System); err != nil {
		return err
	}

	switch goos {
	case "linux":
		return uninstallSystemdService(opts)
	case "darwin":
		return uninstallLaunchdService(opts)
	default:
		return fmt.Errorf("service uninstall is not supported on %s", goos)
	}
}

func validateServiceScope(user bool, system bool) error {
	if user && system {
		return fmt.Errorf("--user and --system cannot be used together")
	}

	return nil
}

func validatePlatformScope(goos string, user bool, system bool) error {
	if goos == "darwin" && system {
		return fmt.Errorf("system-level launchd service is not implemented yet; use --user")
	}

	return nil
}
