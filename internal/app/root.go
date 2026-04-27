package app

import (
	"github.com/spf13/cobra"
)

func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "dbbackup",
		Short: "Database backupt CLI utility",
	}

	rootCmd.AddCommand(newVersionCommand())
	rootCmd.AddCommand(newConfigCommand())
	rootCmd.AddCommand(newTestCommand())
	rootCmd.AddCommand(newBackupCommand())
	rootCmd.AddCommand(newListCommand())

	return rootCmd
}
