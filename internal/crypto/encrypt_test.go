package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"strings"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	enc, err := Encrypt("passphrase", "master")
	if err != nil {
		t.Fatal(err)
	}
	got, err := Decrypt(enc, "master")
	if err != nil {
		t.Fatal(err)
	}
	if got != "passphrase" {
		t.Errorf("got %q", got)
	}
}

func TestDecryptWrongPasswordFails(t *testing.T) {
	enc, _ := Encrypt("passphrase", "master")
	if _, err := Decrypt(enc, "wrong"); err == nil {
		t.Error("expected error with wrong master password")
	}
}

func TestEncryptUsesFreshSaltAndNonce(t *testing.T) {
	a, _ := Encrypt("same", "master")
	b, _ := Encrypt("same", "master")
	if a == b {
		t.Error("two encryptions of the same value should differ")
	}
}

// encryptLegacy produces the PBKDF2 format written by older versions
func encryptLegacy(t *testing.T, plaintext, password string) string {
	t.Helper()
	salt := make([]byte, legacySaltSize)
	rand.Read(salt)
	key, err := deriveKeyLegacy(password, salt)
	if err != nil {
		t.Fatal(err)
	}
	gcm, err := newGCM(key)
	if err != nil {
		t.Fatal(err)
	}
	nonce := make([]byte, gcm.NonceSize())
	rand.Read(nonce)
	combined := append(salt, nonce...)
	combined = gcm.Seal(combined, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(combined)
}

func TestEncryptWritesV2(t *testing.T) {
	enc, _ := Encrypt("x", "master")
	if !strings.HasPrefix(enc, v2Prefix) {
		t.Errorf("new value should start with %q: %s", v2Prefix, enc)
	}
	if NeedsUpgrade(enc) {
		t.Error("v2 value should not need upgrade")
	}
}

func TestDecryptLegacy(t *testing.T) {
	enc := encryptLegacy(t, "old secret", "master")
	if !NeedsUpgrade(enc) {
		t.Error("legacy value should need upgrade")
	}
	got, err := Decrypt(enc, "master")
	if err != nil {
		t.Fatal(err)
	}
	if got != "old secret" {
		t.Errorf("got %q", got)
	}
	if _, err := Decrypt(enc, "wrong"); err == nil {
		t.Error("expected error with wrong master password")
	}
}

func TestNeedsUpgradeEmpty(t *testing.T) {
	if NeedsUpgrade("") {
		t.Error("empty value should not need upgrade")
	}
}
