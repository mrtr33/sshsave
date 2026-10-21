package crypto

import "testing"

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
