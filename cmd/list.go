package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mrtr33/sshsave/internal/storage"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all saved SSH shortcuts",
	Long:  `Display all saved SSH shortcut entries with their commands and aliases.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := storage.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if len(cfg.Entries) == 0 {
			fmt.Println("No entries found. Use 'sshsave add' to create one.")
			return nil
		}

		// Sort entries by name for consistent output
		names := make([]string, 0, len(cfg.Entries))
		for name := range cfg.Entries {
			names = append(names, name)
		}
		sort.Strings(names)

		fmt.Println("Saved SSH shortcuts:")
		fmt.Println()

		for _, name := range names {
			entry := cfg.Entries[name]
			fmt.Printf("  %s", name)

			// Show encryption indicator
			if entry.UseSecret || entry.EncryptedSecret != "" || entry.EncryptedSudoPwd != "" {
				fmt.Printf(" [encrypted]")
			}
			fmt.Println()

			fmt.Printf("    Command: %s\n", entry.Command)

			if len(entry.Alias) > 0 {
				fmt.Printf("    Aliases: %s\n", strings.Join(entry.Alias, ", "))
			}

			if entry.UseSecret && entry.EncryptedSecret != "" {
				fmt.Printf("    Encrypted: SSH passphrase stored\n")
			}
			if entry.EncryptedSudoPwd != "" {
				fmt.Printf("    Encrypted: Sudo password stored\n")
			}

			fmt.Println()
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
