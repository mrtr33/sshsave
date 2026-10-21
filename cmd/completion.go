package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion script",
	Long: `Generate shell completion scripts for sshsave.

To load completions:

Bash:
  sshsave completion bash | sudo tee /etc/bash_completion.d/sshsave > /dev/null
  # or for user-only:
  sshsave completion bash > ~/.local/share/bash-completion/completions/sshsave

Zsh:
  sshsave completion zsh > ~/.zsh/completions/_sshsave
  # Make sure ~/.zsh/completions is in your $fpath

Fish:
  sshsave completion fish > ~/.config/fish/completions/sshsave.fish

PowerShell:
  sshsave completion powershell | Out-String | Invoke-Expression`,
	ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
	Args:      cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return rootCmd.GenBashCompletion(os.Stdout)
		case "zsh":
			return rootCmd.GenZshCompletion(os.Stdout)
		case "fish":
			return rootCmd.GenFishCompletion(os.Stdout, true)
		case "powershell":
			return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
		default:
			return fmt.Errorf("unsupported shell: %s", args[0])
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
