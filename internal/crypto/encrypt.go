package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"syscall"

	"golang.org/x/crypto/argon2"
	"golang.org/x/term"
)

const (
	keySize = 32 // AES-256

	// Current format: "v2:" + base64(salt + nonce + ciphertext + tag),
	// key derived with Argon2id
	v2Prefix     = "v2:"
	v2SaltSize   = 16
	argonTime    = 3
	argonMemory  = 64 * 1024 // KiB
	argonThreads = 4

	// Legacy format: base64(salt + nonce + ciphertext + tag), key derived
	// with PBKDF2-SHA256. Still decrypted, never written.
	legacySaltSize   = 32
	legacyIterations = 100000
)

// ErrWrongPassword is returned when a value does not decrypt, which almost
// always means the master password is wrong
var ErrWrongPassword = errors.New("wrong master password (or corrupted data)")

func deriveKeyV2(password string, salt []byte) []byte {
	return argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, keySize)
}

func deriveKeyLegacy(password string, salt []byte) ([]byte, error) {
	return pbkdf2.Key(sha256.New, password, salt, legacyIterations, keySize)
}

func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}
	return gcm, nil
}

// Encrypt encrypts plaintext using AES-256-GCM with a key derived from
// password with Argon2id
func Encrypt(plaintext, password string) (string, error) {
	salt := make([]byte, v2SaltSize)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	gcm, err := newGCM(deriveKeyV2(password, salt))
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Combine salt + nonce + ciphertext
	combined := append(salt, nonce...)
	combined = gcm.Seal(combined, nonce, []byte(plaintext), nil)

	return v2Prefix + base64.StdEncoding.EncodeToString(combined), nil
}

// Decrypt decrypts a value produced by Encrypt, or by older versions that
// used PBKDF2
func Decrypt(encodedCiphertext, password string) (string, error) {
	legacy := !strings.HasPrefix(encodedCiphertext, v2Prefix)

	combined, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(encodedCiphertext, v2Prefix))
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	saltSize := v2SaltSize
	if legacy {
		saltSize = legacySaltSize
	}
	if len(combined) < saltSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	salt, combined := combined[:saltSize], combined[saltSize:]

	var key []byte
	if legacy {
		key, err = deriveKeyLegacy(password, salt)
		if err != nil {
			return "", fmt.Errorf("failed to derive key: %w", err)
		}
	} else {
		key = deriveKeyV2(password, salt)
	}

	gcm, err := newGCM(key)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(combined) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertext := combined[:nonceSize], combined[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", ErrWrongPassword
	}

	return string(plaintext), nil
}

// NeedsUpgrade reports whether an encrypted value uses the legacy format
func NeedsUpgrade(encodedCiphertext string) bool {
	return encodedCiphertext != "" && !strings.HasPrefix(encodedCiphertext, v2Prefix)
}

// ReadPassword reads a password from stdin without echoing
func ReadPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	password, err := ReadPasswordStdin()
	fmt.Println()
	if err != nil {
		return "", fmt.Errorf("failed to read password: %w", err)
	}
	return string(password), nil
}

// ReadPasswordStdin reads a password from stdin without echoing
func ReadPasswordStdin() ([]byte, error) {
	// Read password from terminal without echoing
	return term.ReadPassword(int(syscall.Stdin))
}

// PromptForMasterPassword prompts for and reads the master password
func PromptForMasterPassword() (string, error) {
	fmt.Print("Enter master password: ")
	password, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return "", fmt.Errorf("failed to read password: %w", err)
	}
	if len(password) == 0 {
		return "", fmt.Errorf("password cannot be empty")
	}
	return string(password), nil
}

// SetMasterPassword prompts for a new master password with confirmation
func SetMasterPassword() (string, error) {
	fmt.Print("Enter new master password: ")
	password1, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return "", fmt.Errorf("failed to read password: %w", err)
	}

	fmt.Print("Confirm master password: ")
	password2, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return "", fmt.Errorf("failed to read password: %w", err)
	}

	if string(password1) != string(password2) {
		return "", fmt.Errorf("passwords do not match")
	}

	if len(password1) == 0 {
		return "", fmt.Errorf("password cannot be empty")
	}

	return string(password1), nil
}
