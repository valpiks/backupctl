package app

import (
	"fmt"

	"github.com/spf13/cobra"
)

const Version = "0.1.0"

func NewVersionCommand() *cobra.Command{
	cmd := &cobra.Command{
		Use: "version",
		Short: "Print dbbackup version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(Version)
		},
	}

	return cmd
}
