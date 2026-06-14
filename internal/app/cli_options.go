package app

import (
	"log/slog"

	"github.com/valpiks/backupctl/internal/config"
	"github.com/valpiks/backupctl/internal/logger"
)

type CLIOptions struct {
	Quiet   bool
	Verbose bool
	Color   string
}

func commandLogger(cfg *config.Config, opts CLIOptions) *slog.Logger {
	level := cfg.Logging.Level
	if opts.Verbose {
		level = "debug"
	}

	return logger.NewCommand(level)
}
