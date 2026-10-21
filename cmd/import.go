package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/mrtr33/sshsave/internal/sshconfig"
	"github.com/mrtr33/sshsave/internal/storage"
	"github.com/spf13/cobra"
)

var (
	importConfigPath string
	importForce      bool
	importPrefix     string
)

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import entries from SSH config file",
	Long: `Import host entries from your SSH config file (~/.ssh/config).

Each concrete Host alias becomes a shortcut that runs "ssh <alias>", so ssh
applies all options from the config itself. Wildcard patterns are skipped.
Existing entries with the same name will be skipped unless --force is used.

Examples:
  sshsave import
  sshsave import --config ~/.ssh/config
  sshsave import --prefix imported- --force`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get config path
		defaultPath, err := sshconfig.GetDefaultSSHConfigPath()
		if err != nil {
			return fmt.Errorf("failed to get SSH config path: %w", err)
		}
		configPath := defaultPath
		// Non-default configs are passed to ssh with -F
		customPath := ""
		if importConfigPath != "" && importConfigPath != defaultPath {
			customPath, err = filepath.Abs(importConfigPath)
			if err != nil {
				return fmt.Errorf("failed to resolve config path: %w", err)
			}
			configPath = customPath
		}

		// Parse SSH config
		fmt.Printf("Parsing SSH config from: %s\n", configPath)
		entries, err := sshconfig.ParseSSHConfig(configPath)
		if err != nil {
			return fmt.Errorf("failed to parse SSH config: %w", err)
		}

		if len(entries) == 0 {
			fmt.Println("No host entries found in SSH config")
			return nil
		}

		// Load current config
		cfg, err := storage.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		imported := 0
		skipped := 0

		for _, entry := range entries {
			// Skip invalid entries (wildcards, patterns, etc.)
			if !entry.IsValidEntry() {
				skipped++
				continue
			}

			// Generate entry name
			name := importPrefix + entry.Host

			// Check if entry already exists
			if _, exists := cfg.Entries[name]; exists && !importForce {
				fmt.Printf("  Skipping '%s' (already exists)\n", name)
				skipped++
				continue
			}

			// Convert to SSH command
			command := entry.ToSSHCommand(customPath)

			// Add entry (keeps secrets and aliases of an existing entry)
			if err := cfg.AddEntry(name, command, nil, true); err != nil {
				fmt.Printf("  Skipping '%s' (%v)\n", name, err)
				skipped++
				continue
			}

			fmt.Printf("  Imported '%s': %s\n", name, command)
			imported++
		}

		// Save config if any entries were imported
		if imported > 0 {
			if err := storage.Save(cfg); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}
		}

		fmt.Printf("\nImported %d entries, skipped %d\n", imported, skipped)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(importCmd)
	importCmd.Flags().StringVar(&importConfigPath, "config", "", "Path to SSH config file (default: ~/.ssh/config)")
	importCmd.Flags().BoolVar(&importForce, "force", false, "Overwrite existing entries")
	importCmd.Flags().StringVar(&importPrefix, "prefix", "", "Prefix to add to imported entry names")
}
