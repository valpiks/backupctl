package app

import (
	"github.com/spf13/cobra"
)

func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "dbbackup",
		Short: "Database backupt CLI utility",
	}

	rootCmd.AddCommand(NewVersionCommand())
	rootCmd.AddCommand(NewConfigCommand())
	rootCmd.AddCommand(newTestCommand())
	rootCmd.AddCommand(newBackupCommand())

	return rootCmd
}
