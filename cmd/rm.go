package cmd

import (
	"fmt"

	"github.com/mrtr33/sshsave/internal/storage"
	"github.com/spf13/cobra"
)

var rmCmd = &cobra.Command{
	Use:   "rm <name>",
	Short: "Remove a saved SSH shortcut",
	Long: `Delete a saved SSH shortcut entry by name or alias.

Examples:
  sshsave rm work
  sshsave rm w    # using an alias`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeEntryNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		nameOrAlias := args[0]

		// Load config
		cfg, err := storage.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Remove entry
		name, err := cfg.RemoveEntry(nameOrAlias)
		if err != nil {
			return err
		}

		// Save config
		if err := storage.Save(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("Removed entry '%s'\n", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(rmCmd)
}
