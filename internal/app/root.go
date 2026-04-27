package app

import (
	"github.com/spf13/cobra"
)

func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "backupctl",
		Short: "Database backup CLI utility",
		SilenceUsage: true,
	}

	rootCmd.AddCommand(newVersionCommand())
	rootCmd.AddCommand(newConfigCommand())
	rootCmd.AddCommand(newTestCommand())
	rootCmd.AddCommand(newBackupCommand())
	rootCmd.AddCommand(newListCommand())
	rootCmd.AddCommand(newRestorCommand())

	return rootCmd
}
