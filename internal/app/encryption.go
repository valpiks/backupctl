package app

import "github.com/valpiks/backupctl/internal/config"

func encryptionEnabled(cfg *config.Config) bool {
	return cfg.Backup.Encryption != nil && cfg.Backup.Encryption.Enabled
}

func encryptionPassword(cfg *config.Config) string {
	if cfg.Backup.Encryption == nil {
		return ""
	}

	return cfg.Backup.Encryption.Password
}
