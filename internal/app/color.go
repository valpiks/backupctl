package app

import (
	"io"
	"os"
)

func colorEnabled(mode string, out io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	if os.Getenv("CLICOLOR_FORCE") != "" {
		return true
	}

	switch mode {
	case "always":
		return true
	case "never":
		return false
	default:
		file, ok := out.(*os.File)
		if !ok {
			return false
		}

		stat, err := file.Stat()
		if err != nil {
			return false
		}

		return stat.Mode()&os.ModeCharDevice != 0
	}
}

func colorStatus(status string, enabled bool) string {
	if !enabled {
		return status
	}

	switch status {
	case "OK":
		return "\033[32mOK\033[0m"
	case "WARN":
		return "\033[33mWARN\033[0m"
	case "FAIL":
		return "\033[31mFAIL\033[0m"
	default:
		return status
	}
}
