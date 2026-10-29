package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/mrtr33/sshsave/internal/askpass"
	"github.com/mrtr33/sshsave/internal/crypto"
	"github.com/mrtr33/sshsave/internal/executor"
	"github.com/mrtr33/sshsave/internal/storage"
	"github.com/spf13/cobra"
)

var keyCmd = &cobra.Command{
	Use:   "key <name>",
	Short: "Install your SSH key on the host so no password is needed",
	Long: `Copy your public key to the entry's host with ssh-copy-id. When the entry
has a stored secret, ssh-copy-id gets it automatically, so you do not need
to remember the password. Afterwards you can drop the stored secret, and
"sshsave <name>" logs in with the key without asking anything.

Needs a key pair (create one with "ssh-keygen -t ed25519").

Examples:
  sshsave key work`,
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

		command, err := copyIDCommand(entry.Command)
		if err != nil {
			return fmt.Errorf("entry '%s': %w", name, err)
		}
		fmt.Printf("Running: %s\n", command)

		var env []string
		if entry.UseSecret && entry.EncryptedSecret != "" {
			master, err := crypto.PromptForMasterPassword()
			if err != nil {
				return fmt.Errorf("failed to get master password: %w", err)
			}
			secret, err := crypto.Decrypt(entry.EncryptedSecret, master)
			if err != nil {
				return err
			}
			upgradeSecrets(cfg, master)

			env, err = askpass.Env(secret)
			if err != nil {
				return err
			}
		}

		// executor.Run exits with ssh-copy-id's code when it fails
		if err := executor.Run(command, env); err != nil {
			return err
		}

		if entry.EncryptedSecret == "" {
			return nil
		}
		fmt.Print("\nKey installed. Remove the stored SSH secret so logging in asks nothing? [y/N] ")
		answer, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		if strings.ToLower(strings.TrimSpace(answer)) != "y" {
			return nil
		}

		cfg, err = storage.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		entry, _, _ = cfg.FindEntry(name)
		entry.EncryptedSecret = ""
		entry.UseSecret = false
		if err := storage.Save(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
		fmt.Printf("Stored secret removed from '%s'\n", name)
		return nil
	},
}

// sshArgFlags are the ssh options that take a value
const sshArgFlags = "BbcDEeFIiJLlmOoPpQRSWw"

// copyIDCommand turns "ssh [options] host [options]" into the matching
// ssh-copy-id command, which only accepts options before the host
func copyIDCommand(command string) (string, error) {
	words := strings.Fields(command)
	if len(words) < 2 || words[0] != "ssh" {
		return "", fmt.Errorf("command does not start with 'ssh '")
	}

	var opts []string
	host := ""
	for i := 1; i < len(words); i++ {
		w := words[i]
		if !strings.HasPrefix(w, "-") {
			if host != "" {
				return "", fmt.Errorf("command runs a remote command, cannot copy a key with it")
			}
			host = w
			continue
		}
		opts = append(opts, w)
		// "-p 2222" carries its value in the next word, "-p2222" does not
		if len(w) == 2 && strings.ContainsRune(sshArgFlags, rune(w[1])) && i+1 < len(words) {
			i++
			opts = append(opts, words[i])
		}
	}
	if host == "" {
		return "", fmt.Errorf("no host in command")
	}
	return strings.Join(append(append([]string{"ssh-copy-id"}, opts...), host), " "), nil
}

func init() {
	rootCmd.AddCommand(keyCmd)
}
