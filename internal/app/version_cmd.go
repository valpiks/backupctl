package app

import (
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

func newVersionCommand() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print backupctl version",
		Long:  "Print backupctl version and build information.",
		Example: `  backupctl version
  backupctl version --json`,
		Run: func(cmd *cobra.Command, args []string) {
			info := map[string]string{
				"version": Version,
				"commit":  Commit,
				"date":    Date,
				"go":      runtime.Version(),
				"os":      runtime.GOOS,
				"arch":    runtime.GOARCH,
			}

			if jsonOutput {
				data, err := json.MarshalIndent(info, "", "  ")
				if err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "marshal version json: %v\n", err)
					return
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(data))
				return
			}

			PrintKV(cmd.OutOrStdout(), "backupctl version", []KV{
				{Key: "version", Value: info["version"]},
				{Key: "commit", Value: info["commit"]},
				{Key: "date", Value: info["date"]},
				{Key: "go", Value: info["go"]},
				{Key: "platform", Value: fmt.Sprintf("%s/%s", info["os"], info["arch"])},
			})
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Print version as JSON")

	return cmd
}
