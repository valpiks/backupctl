package app

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/valpiks/backupctl/internal/config"
)

const defaultConfigFileName = "config.yaml"

var systemConfigPath = "/etc/backupctl/config.yaml"

func loadConfig(path string) (*config.Config, error) {
	if err := requireConfigPath(path); err != nil {
		return nil, err
	}

	cfg, err := config.Load(path)
	if err != nil {
		return nil, WithHint(
			err,
			"pass --config <path>, set BACKUPCTL_CONFIG, or run backupctl init",
		)
	}

	return cfg, nil
}

func defaultConfigPath() string {
	if value := os.Getenv("BACKUPCTL_CONFIG"); value != "" {
		return value
	}

	if value := os.Getenv("XDG_CONFIG_HOME"); value != "" {
		return filepath.Join(value, "backupctl", defaultConfigFileName)
	}

	var userPath string
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		userPath = filepath.Join(home, ".config", "backupctl", defaultConfigFileName)
		if fileExists(userPath) {
			return userPath
		}
	}

	if fileExists(systemConfigPath) {
		return systemConfigPath
	}

	if userPath != "" {
		return userPath
	}

	return ""
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func addConfigFlag(cmd *cobra.Command, target *string) {
	*target = defaultConfigPath()
	cmd.Flags().StringVarP(target, "config", "c", *target, "Path to config file")
}

func requireConfigPath(path string) error {
	if path == "" {
		return WithHint(
			fmt.Errorf("config path is required"),
			"pass --config <path>, set BACKUPCTL_CONFIG, or run backupctl init",
		)
	}

	return nil
}
