package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfigPathUsesBackupctlConfig(t *testing.T) {
	t.Setenv("BACKUPCTL_CONFIG", "/tmp/backupctl.yaml")
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	if got := defaultConfigPath(); got != "/tmp/backupctl.yaml" {
		t.Fatalf("defaultConfigPath() = %q, want BACKUPCTL_CONFIG", got)
	}
}

func TestDefaultConfigPathUsesXDGConfigHome(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("BACKUPCTL_CONFIG", "")
	t.Setenv("XDG_CONFIG_HOME", tmp)

	want := filepath.Join(tmp, "backupctl", "config.yaml")
	if got := defaultConfigPath(); got != want {
		t.Fatalf("defaultConfigPath() = %q, want %q", got, want)
	}
}

func TestDefaultConfigPathUsesSystemConfigWhenUserConfigMissing(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("BACKUPCTL_CONFIG", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", filepath.Join(tmp, "home"))

	systemPath := filepath.Join(tmp, "etc", "backupctl", "config.yaml")
	oldSystemPath := systemConfigPath
	systemConfigPath = systemPath
	t.Cleanup(func() {
		systemConfigPath = oldSystemPath
	})

	if err := os.MkdirAll(filepath.Dir(systemPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(systemPath, []byte("config"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if got := defaultConfigPath(); got != systemPath {
		t.Fatalf("defaultConfigPath() = %q, want system config %q", got, systemPath)
	}
}

func TestRequireConfigPath(t *testing.T) {
	if err := requireConfigPath("config.yaml"); err != nil {
		t.Fatalf("requireConfigPath() error = %v", err)
	}

	if err := requireConfigPath(""); err == nil {
		t.Fatal("requireConfigPath() error = nil, want error")
	}
}
