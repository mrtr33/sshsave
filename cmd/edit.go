package cmd

import (
	"fmt"
	"strings"

	"github.com/mrtr33/sshsave/internal/crypto"
	"github.com/mrtr33/sshsave/internal/storage"
	"github.com/spf13/cobra"
)

var (
	editCommand           string
	editAddAliases        []string
	editRemoveAliases     []string
	editWithSecret        bool
	editClearSecret       bool
	editWithSudoPassword  bool
	editClearSudoPassword bool
)

var editCmd = &cobra.Command{
	Use:   "edit <name>",
	Short: "Change the command, aliases or secrets of a saved shortcut",
	Long: `Change parts of a saved SSH shortcut without re-creating it.
The entry can be given by name or alias.

Examples:
  sshsave edit work --command "ssh me@10.0.0.5 -p 2222"
  sshsave edit work --add-alias w --remove-alias old
  sshsave edit work --with-secret
  sshsave edit work --clear-secret`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeEntryNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		if editWithSecret && editClearSecret {
			return fmt.Errorf("--with-secret and --clear-secret cannot be used together")
		}
		if editWithSudoPassword && editClearSudoPassword {
			return fmt.Errorf("--with-sudo-password and --clear-sudo-password cannot be used together")
		}
		if cmd.Flags().NFlag() == 0 {
			return fmt.Errorf("nothing to change, see 'sshsave edit --help'")
		}

		cfg, err := storage.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		entry, name, found := cfg.FindEntry(args[0])
		if !found {
			return fmt.Errorf("entry '%s' not found", args[0])
		}

		command := entry.Command
		if cmd.Flags().Changed("command") {
			if !strings.HasPrefix(editCommand, "ssh ") {
				return fmt.Errorf("SSH command must start with 'ssh '")
			}
			command = editCommand
		}

		aliases, err := editAliasList(entry.Alias, editAddAliases, editRemoveAliases)
		if err != nil {
			return err
		}

		if err := cfg.UpdateEntry(name, command, aliases); err != nil {
			return err
		}

		if editClearSecret {
			entry.EncryptedSecret = ""
			entry.UseSecret = false
		}
		if editClearSudoPassword {
			entry.EncryptedSudoPwd = ""
		}

		if editWithSecret || editWithSudoPassword {
			masterPassword, err := crypto.PromptForMasterPassword()
			if err != nil {
				return fmt.Errorf("failed to get master password: %w", err)
			}
			if err := verifyMaster(cfg, masterPassword); err != nil {
				return err
			}

			if editWithSecret {
				encrypted, err := promptAndEncrypt("Enter SSH passphrase/password to encrypt: ", masterPassword)
				if err != nil {
					return fmt.Errorf("failed to encrypt secret: %w", err)
				}
				entry.EncryptedSecret = encrypted
				entry.UseSecret = true
			}
			if editWithSudoPassword {
				encrypted, err := promptAndEncrypt("Enter sudo password to encrypt: ", masterPassword)
				if err != nil {
					return fmt.Errorf("failed to encrypt sudo password: %w", err)
				}
				entry.EncryptedSudoPwd = encrypted
			}
		}

		if err := storage.Save(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("Updated entry '%s'\n", name)
		return nil
	},
}

// editAliasList applies alias additions and removals to current
func editAliasList(current, add, remove []string) ([]string, error) {
	out := []string{}
	for _, a := range current {
		keep := true
		for _, r := range remove {
			if a == r {
				keep = false
			}
		}
		if keep {
			out = append(out, a)
		}
	}

	for _, r := range remove {
		found := false
		for _, a := range current {
			if a == r {
				found = true
			}
		}
		if !found {
			return nil, fmt.Errorf("alias '%s' is not set on this entry", r)
		}
	}

	for _, a := range add {
		exists := false
		for _, o := range out {
			if a == o {
				exists = true
			}
		}
		if !exists {
			out = append(out, a)
		}
	}
	return out, nil
}

func init() {
	rootCmd.AddCommand(editCmd)
	editCmd.Flags().StringVar(&editCommand, "command", "", "New SSH command")
	editCmd.Flags().StringSliceVar(&editAddAliases, "add-alias", nil, "Alias to add (can be used multiple times)")
	editCmd.Flags().StringSliceVar(&editRemoveAliases, "remove-alias", nil, "Alias to remove (can be used multiple times)")
	editCmd.Flags().BoolVar(&editWithSecret, "with-secret", false, "Set a new encrypted SSH passphrase/password (prompted)")
	editCmd.Flags().BoolVar(&editClearSecret, "clear-secret", false, "Remove the stored SSH secret")
	editCmd.Flags().BoolVar(&editWithSudoPassword, "with-sudo-password", false, "Set a new encrypted sudo password (prompted)")
	editCmd.Flags().BoolVar(&editClearSudoPassword, "clear-sudo-password", false, "Remove the stored sudo password")
}
