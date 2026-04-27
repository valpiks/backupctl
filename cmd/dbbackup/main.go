package main

import (
	"os"

	"github.com/valpiks/dbbackup/internal/app"
)

func main() {
	if err := app.NewRootCommand().Execute(); err != nil {
		os.Exit(1)
	}
}
