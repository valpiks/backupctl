package app

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newCompletionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion script",
		Long:  "Generate shell completion scripts for bash, zsh, fish, or powershell.",
		Example: `  backupctl completion zsh
  source <(backupctl completion zsh)
  backupctl completion bash > /etc/bash_completion.d/backupctl
  backupctl completion fish > ~/.config/fish/completions/backupctl.fish`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := cmd.Root()
			out := cmd.OutOrStdout()

			switch args[0] {
			case "bash":
				return root.GenBashCompletion(out)
			case "zsh":
				return root.GenZshCompletion(out)
			case "fish":
				return root.GenFishCompletion(out, true)
			case "powershell":
				return root.GenPowerShellCompletion(out)
			default:
				return fmt.Errorf("unsupported shell: %s", args[0])
			}
		},
	}

	return cmd
}
