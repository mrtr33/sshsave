package cmd

import (
	"fmt"

	"github.com/mrtr33/sshsave/internal/askpass"
	"github.com/mrtr33/sshsave/internal/crypto"
	"github.com/mrtr33/sshsave/internal/executor"
	"github.com/mrtr33/sshsave/internal/storage"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run <name>",
	Short: "Run a saved SSH shortcut",
	Long: `Execute a saved SSH shortcut by name or alias.

The command will pass through stdin/stdout/stderr for full SSH interactivity
including password prompts and terminal sessions.

If the entry has an encrypted secret, it is decrypted with the master password
and handed to ssh through SSH_ASKPASS, so it is never shown on screen.

Examples:
  sshsave run work
  sshsave run w    # using an alias`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeEntryNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		nameOrAlias := args[0]

		// Load config
		cfg, err := storage.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Find entry by name or alias
		entry, name, found := cfg.FindEntry(nameOrAlias)
		if !found {
			return fmt.Errorf("entry '%s' not found", nameOrAlias)
		}

		fmt.Printf("Running '%s': %s\n", name, entry.Command)

		// Handle encrypted secrets if present
		var env []string
		if entry.UseSecret && entry.EncryptedSecret != "" {
			// Get master password
			masterPassword, err := crypto.PromptForMasterPassword()
			if err != nil {
				return fmt.Errorf("failed to get master password: %w", err)
			}

			// Decrypt the secret
			secret, err := crypto.Decrypt(entry.EncryptedSecret, masterPassword)
			if err != nil {
				return fmt.Errorf("failed to decrypt secret: %w", err)
			}

			env, err = askpass.Env(secret)
			if err != nil {
				return err
			}
		}

		return executor.Run(entry.Command, env)
	},
}

// completeEntryNames completes saved entry names and aliases
func completeEntryNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	cfg, err := storage.Load()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return cfg.Names(), cobra.ShellCompDirectiveNoFileComp
}

func init() {
	rootCmd.AddCommand(runCmd)
}
