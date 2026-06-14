package main

import (
	"os"

	"github.com/valpiks/backupctl/internal/app"
)

func main() {
	cmd := app.NewRootCommand()
	if err := cmd.Execute(); err != nil {
		app.PrintError(cmd.ErrOrStderr(), err)
		os.Exit(1)
	}
}
