package app

import (
	"fmt"

	"github.com/spf13/cobra"
)

const Version = "0.2.0"

func newVersionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print backupctl version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(Version)
		},
	}

	return cmd
}
