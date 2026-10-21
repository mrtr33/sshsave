package cmd

import (
	"fmt"

	"github.com/mrtr33/sshsave/internal/storage"
	"github.com/spf13/cobra"
)

var renameCmd = &cobra.Command{
	Use:   "rename <name> <new-name>",
	Short: "Rename a saved SSH shortcut",
	Long: `Rename a saved SSH shortcut. The entry can be given by name or alias;
its command, aliases and secrets are kept.

Examples:
  sshsave rename work office`,
	Args:              cobra.ExactArgs(2),
	ValidArgsFunction: completeEntryNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := storage.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		oldName, err := cfg.RenameEntry(args[0], args[1])
		if err != nil {
			return err
		}

		if err := storage.Save(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("Renamed entry '%s' to '%s'\n", oldName, args[1])
		return nil
	},
}

func init() {
	rootCmd.AddCommand(renameCmd)
}
