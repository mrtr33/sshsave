package cmd

import (
	"fmt"
	"strings"

	"github.com/mrtr33/sshsave/internal/crypto"
	"github.com/mrtr33/sshsave/internal/storage"
	"github.com/spf13/cobra"
)

var (
	addAliases      []string
	addForce        bool
	addWithSecret   bool
	addSecretValue  string
	addSudoValue    string
	addSudoPassword bool
)

var addCmd = &cobra.Command{
	Use:   "add <name> <ssh-command>",
	Short: "Add or update an SSH shortcut entry",
	Long: `Add a new SSH shortcut or update an existing one.

The SSH command should be the full command you would normally type,
including "ssh" and all arguments.

Updating with --force keeps stored secrets, and keeps aliases unless
new ones are given.

Secrets are always asked interactively so they never end up in shell
history or the process list.

Examples:
  sshsave add work ssh user@192.168.10.10
  sshsave add home ssh me@10.0.0.2 -p 2222 --alias h
  sshsave add work ssh user@newhost.com --force
  sshsave add secure ssh user@host.com --with-secret
  sshsave add admin ssh root@server.com --with-sudo-password`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		command := strings.Join(args[1:], " ")

		if addSecretValue != "" {
			return fmt.Errorf("--secret is no longer supported because it leaks the secret into shell history; use --with-secret to be prompted")
		}
		if addSudoValue != "" {
			return fmt.Errorf("--sudo-password is no longer supported because it leaks the password into shell history; use --with-sudo-password to be prompted")
		}

		// Validate that command starts with ssh
		if !strings.HasPrefix(command, "ssh ") {
			return fmt.Errorf("SSH command must start with 'ssh '")
		}

		// Load existing config
		cfg, err := storage.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Add or update entry
		if err := cfg.AddEntry(name, command, addAliases, addForce); err != nil {
			return err
		}

		// Handle encrypted secrets if requested
		if addWithSecret || addSudoPassword {
			entry := cfg.Entries[name]

			// Get master password
			masterPassword, err := crypto.PromptForMasterPassword()
			if err != nil {
				return fmt.Errorf("failed to get master password: %w", err)
			}

			// Encrypt SSH secret/passphrase if requested
			if addWithSecret {
				secret, err := crypto.ReadPassword("Enter SSH passphrase/password to encrypt: ")
				if err != nil {
					return fmt.Errorf("failed to read secret: %w", err)
				}

				encrypted, err := crypto.Encrypt(secret, masterPassword)
				if err != nil {
					return fmt.Errorf("failed to encrypt secret: %w", err)
				}
				entry.EncryptedSecret = encrypted
				entry.UseSecret = true
				fmt.Println("SSH secret encrypted successfully")
			}

			// Encrypt sudo password if requested
			if addSudoPassword {
				sudoPassword, err := crypto.ReadPassword("Enter sudo password to encrypt: ")
				if err != nil {
					return fmt.Errorf("failed to read sudo password: %w", err)
				}

				encrypted, err := crypto.Encrypt(sudoPassword, masterPassword)
				if err != nil {
					return fmt.Errorf("failed to encrypt sudo password: %w", err)
				}
				entry.EncryptedSudoPwd = encrypted
				fmt.Println("Sudo password encrypted successfully")
			}
		}

		// Save config
		if err := storage.Save(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		if addForce {
			fmt.Printf("Updated entry '%s'\n", name)
		} else {
			fmt.Printf("Added entry '%s'\n", name)
		}

		if aliases := cfg.Entries[name].Alias; len(aliases) > 0 {
			fmt.Printf("Aliases: %s\n", strings.Join(aliases, ", "))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
	addCmd.Flags().StringSliceVar(&addAliases, "alias", []string{}, "Alias names for this entry (can be used multiple times)")
	addCmd.Flags().BoolVar(&addForce, "force", false, "Overwrite existing entry")
	addCmd.Flags().BoolVar(&addWithSecret, "with-secret", false, "Encrypt and store SSH passphrase/password (prompted)")
	addCmd.Flags().StringVar(&addSecretValue, "secret", "", "Removed: use --with-secret")
	addCmd.Flags().MarkHidden("secret")
	addCmd.Flags().StringVar(&addSudoValue, "sudo-password", "", "Removed: use --with-sudo-password")
	addCmd.Flags().MarkHidden("sudo-password")
	addCmd.Flags().BoolVar(&addSudoPassword, "with-sudo-password", false, "Encrypt and store sudo password (prompted)")
}
