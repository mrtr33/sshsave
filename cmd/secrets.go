package cmd

import (
	"fmt"
	"os"
	"sort"

	"github.com/mrtr33/sshsave/internal/crypto"
	"github.com/mrtr33/sshsave/internal/storage"
)

// secretField is one encrypted value stored in an entry
type secretField struct {
	entry string
	label string
	value *string
}

// secretFields returns every stored encrypted value, in a stable order
func secretFields(cfg *storage.Config) []secretField {
	names := make([]string, 0, len(cfg.Entries))
	for name := range cfg.Entries {
		names = append(names, name)
	}
	sort.Strings(names)

	var fields []secretField
	for _, name := range names {
		e := cfg.Entries[name]
		if e.EncryptedSecret != "" {
			fields = append(fields, secretField{name, "SSH secret", &e.EncryptedSecret})
		}
		if e.EncryptedSudoPwd != "" {
			fields = append(fields, secretField{name, "sudo password", &e.EncryptedSudoPwd})
		}
	}
	return fields
}

// verifyMaster checks that master decrypts the existing secrets, so all
// secrets keep sharing one master password. It passes when there are none.
func verifyMaster(cfg *storage.Config, master string) error {
	fields := secretFields(cfg)
	if len(fields) == 0 {
		return nil
	}
	if _, err := crypto.Decrypt(*fields[0].value, master); err != nil {
		return fmt.Errorf("master password does not match the one used for existing secrets")
	}
	return nil
}

// promptAndEncrypt asks for a secret without echo and encrypts it
func promptAndEncrypt(prompt, master string) (string, error) {
	secret, err := crypto.ReadPassword(prompt)
	if err != nil {
		return "", err
	}
	if secret == "" {
		return "", fmt.Errorf("secret cannot be empty")
	}
	return crypto.Encrypt(secret, master)
}

// upgradeSecrets re-encrypts every legacy value that master can decrypt
// with the current format and saves the config. Failures only print a
// warning, since running the entry must not depend on this.
func upgradeSecrets(cfg *storage.Config, master string) {
	upgraded := 0
	for _, f := range secretFields(cfg) {
		if !crypto.NeedsUpgrade(*f.value) {
			continue
		}
		plain, err := crypto.Decrypt(*f.value, master)
		if err != nil {
			continue
		}
		enc, err := crypto.Encrypt(plain, master)
		if err != nil {
			continue
		}
		*f.value = enc
		upgraded++
	}

	if upgraded == 0 {
		return
	}
	if err := storage.Save(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to save upgraded secrets: %v\n", err)
		return
	}
	fmt.Printf("Upgraded encryption of %d stored secret(s)\n", upgraded)
}
