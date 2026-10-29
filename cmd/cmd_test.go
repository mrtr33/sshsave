package cmd

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mrtr33/sshsave/internal/crypto"
	"github.com/mrtr33/sshsave/internal/storage"
)

func TestEditAliasList(t *testing.T) {
	got, err := editAliasList([]string{"a", "b"}, []string{"c", "a"}, []string{"b"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, ",") != "a,c" {
		t.Errorf("got %v", got)
	}
	if _, err := editAliasList([]string{"a"}, nil, []string{"zz"}); err == nil {
		t.Error("removing a missing alias should fail")
	}
}

func TestVerifyMaster(t *testing.T) {
	cfg := storage.NewConfig()
	if err := verifyMaster(cfg, "anything"); err != nil {
		t.Errorf("no secrets should pass: %v", err)
	}

	cfg.AddEntry("a", "ssh a", nil, false)
	enc, _ := crypto.Encrypt("s", "master")
	cfg.Entries["a"].EncryptedSecret = enc

	if err := verifyMaster(cfg, "master"); err != nil {
		t.Errorf("right master failed: %v", err)
	}
	if err := verifyMaster(cfg, "wrong"); err == nil {
		t.Error("wrong master should fail")
	}
}

// legacyValue builds a value in the PBKDF2 format written by old versions
func legacyValue(t *testing.T, plaintext, password string) string {
	t.Helper()
	salt := make([]byte, 32)
	rand.Read(salt)
	key, err := pbkdf2.Key(sha256.New, password, salt, 100000, 32)
	if err != nil {
		t.Fatal(err)
	}
	block, _ := aes.NewCipher(key)
	gcm, _ := cipher.NewGCM(block)
	nonce := make([]byte, gcm.NonceSize())
	rand.Read(nonce)
	combined := append(salt, nonce...)
	combined = gcm.Seal(combined, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(combined)
}

func TestUpgradeSecrets(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := storage.NewConfig()
	cfg.AddEntry("old", "ssh o", nil, false)
	cfg.AddEntry("new", "ssh n", nil, false)
	cfg.AddEntry("foreign", "ssh f", nil, false)

	current, _ := crypto.Encrypt("s-new", "master")
	cfg.Entries["old"].EncryptedSecret = legacyValue(t, "s-old", "master")
	cfg.Entries["old"].EncryptedSudoPwd = legacyValue(t, "sudo-old", "master")
	cfg.Entries["new"].EncryptedSecret = current
	foreign := legacyValue(t, "s-foreign", "another")
	cfg.Entries["foreign"].EncryptedSecret = foreign

	upgradeSecrets(cfg, "master")

	for _, v := range []string{cfg.Entries["old"].EncryptedSecret, cfg.Entries["old"].EncryptedSudoPwd} {
		if crypto.NeedsUpgrade(v) {
			t.Error("legacy value decryptable with master was not upgraded")
		}
	}
	if got, _ := crypto.Decrypt(cfg.Entries["old"].EncryptedSudoPwd, "master"); got != "sudo-old" {
		t.Errorf("upgraded sudo password decrypts to %q", got)
	}
	if cfg.Entries["new"].EncryptedSecret != current {
		t.Error("current-format value must not change")
	}
	if cfg.Entries["foreign"].EncryptedSecret != foreign {
		t.Error("value of another master password must not change")
	}

	saved, err := storage.Load()
	if err != nil {
		t.Fatal(err)
	}
	if crypto.NeedsUpgrade(saved.Entries["old"].EncryptedSecret) {
		t.Error("upgrade was not saved")
	}
}

func TestUpgradeSecretsNoopDoesNotSave(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := storage.NewConfig()
	cfg.AddEntry("a", "ssh a", nil, false)
	enc, _ := crypto.Encrypt("s", "master")
	cfg.Entries["a"].EncryptedSecret = enc

	upgradeSecrets(cfg, "master")

	if _, err := os.Stat(filepath.Join(home, ".config", "sshlauncher", "sshlauncher.json")); !os.IsNotExist(err) {
		t.Error("config should not be written when nothing is upgraded")
	}
}

func TestCopyIDCommand(t *testing.T) {
	cases := map[string]string{
		"ssh me@host":                       "ssh-copy-id me@host",
		"ssh me@host -p 2222":               "ssh-copy-id -p 2222 me@host",
		"ssh -p2222 me@host":                "ssh-copy-id -p2222 me@host",
		"ssh -i ~/.ssh/k -o Port=22 me@h":   "ssh-copy-id -i ~/.ssh/k -o Port=22 me@h",
		"ssh -4 me@host -J jump@bastion -v": "ssh-copy-id -4 -J jump@bastion -v me@host",
	}
	for in, want := range cases {
		got, err := copyIDCommand(in)
		if err != nil || got != want {
			t.Errorf("copyIDCommand(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"mosh me@host", "ssh", "ssh -p 22", "ssh me@host uptime"} {
		if _, err := copyIDCommand(bad); err == nil {
			t.Errorf("copyIDCommand(%q) should fail", bad)
		}
	}
}
