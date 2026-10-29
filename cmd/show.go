package cmd

import (
	"fmt"

	"github.com/mrtr33/sshsave/internal/crypto"
	"github.com/mrtr33/sshsave/internal/storage"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Print the stored secrets of a saved shortcut",
	Long: `Decrypt the stored SSH secret and sudo password of an entry with the
master password and print them, for when you need to type one yourself.

Examples:
  sshsave show work`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeEntryNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := storage.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		entry, name, found := cfg.FindEntry(args[0])
		if !found {
			return fmt.Errorf("entry '%s' not found", args[0])
		}

		var fields []secretField
		for _, f := range secretFields(cfg) {
			if f.entry == name {
				fields = append(fields, f)
			}
		}
		if len(fields) == 0 {
			return fmt.Errorf("entry '%s' has no stored secrets", name)
		}

		master, err := crypto.PromptForMasterPassword()
		if err != nil {
			return fmt.Errorf("failed to get master password: %w", err)
		}

		fmt.Printf("'%s': %s\n", name, entry.Command)
		for _, f := range fields {
			plain, err := crypto.Decrypt(*f.value, master)
			if err != nil {
				return err
			}
			fmt.Printf("  %s: %s\n", f.label, plain)
		}

		upgradeSecrets(cfg, master)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(showCmd)
}
