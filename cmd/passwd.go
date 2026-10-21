package cmd

import (
	"fmt"
	"strings"

	"github.com/mrtr33/sshsave/internal/crypto"
	"github.com/mrtr33/sshsave/internal/storage"
	"github.com/spf13/cobra"
)

var passwdCmd = &cobra.Command{
	Use:   "passwd",
	Short: "Change the master password",
	Long: `Re-encrypt every stored secret with a new master password.

All secrets must decrypt with the current master password; if any does not,
nothing is changed.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := storage.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		fields := secretFields(cfg)
		if len(fields) == 0 {
			return fmt.Errorf("no stored secrets, nothing to re-encrypt")
		}

		current, err := crypto.PromptForMasterPassword()
		if err != nil {
			return fmt.Errorf("failed to get master password: %w", err)
		}

		// Decrypt everything first so a mismatch changes nothing
		plain := make([]string, len(fields))
		var failed []string
		for i, f := range fields {
			p, err := crypto.Decrypt(*f.value, current)
			if err != nil {
				failed = append(failed, fmt.Sprintf("%s (%s)", f.entry, f.label))
				continue
			}
			plain[i] = p
		}
		if len(failed) > 0 {
			return fmt.Errorf("current master password does not decrypt: %s", strings.Join(failed, ", "))
		}

		newMaster, err := crypto.SetMasterPassword()
		if err != nil {
			return err
		}

		for i, f := range fields {
			enc, err := crypto.Encrypt(plain[i], newMaster)
			if err != nil {
				return fmt.Errorf("failed to encrypt %s of '%s': %w", f.label, f.entry, err)
			}
			*f.value = enc
		}

		if err := storage.Save(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("Master password changed, %d secret(s) re-encrypted\n", len(fields))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(passwdCmd)
}
