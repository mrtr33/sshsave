package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"syscall"

	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/term"
)

const (
	// Encryption parameters
	keySize    = 32 // AES-256
	iterations = 100000
	saltSize   = 32
)

// DeriveKey derives an encryption key from a master password and salt
func DeriveKey(password string, salt []byte) []byte {
	return pbkdf2.Key([]byte(password), salt, iterations, keySize, sha256.New)
}

// GenerateSalt generates a random salt for key derivation
func GenerateSalt() ([]byte, error) {
	salt := make([]byte, saltSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}
	return salt, nil
}

// Encrypt encrypts plaintext using AES-GCM with the given password
// Returns: base64(salt + nonce + ciphertext + tag)
func Encrypt(plaintext, password string) (string, error) {
	// Generate a random salt
	salt, err := GenerateSalt()
	if err != nil {
		return "", err
	}

	// Derive key from password and salt
	key := DeriveKey(password, salt)

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate a random nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt the plaintext
	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)

	// Combine salt + nonce + ciphertext
	combined := append(salt, nonce...)
	combined = append(combined, ciphertext...)

	// Encode to base64
	return base64.StdEncoding.EncodeToString(combined), nil
}

// Decrypt decrypts ciphertext using AES-GCM with the given password
func Decrypt(encodedCiphertext, password string) (string, error) {
	// Decode from base64
	combined, err := base64.StdEncoding.DecodeString(encodedCiphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	// Extract salt
	if len(combined) < saltSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	salt := combined[:saltSize]
	combined = combined[saltSize:]

	// Derive key from password and salt
	key := DeriveKey(password, salt)

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Extract nonce
	nonceSize := gcm.NonceSize()
	if len(combined) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce := combined[:nonceSize]
	ciphertext := combined[nonceSize:]

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
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

// GetMasterPasswordFile returns the path to the master password file
func GetMasterPasswordFile() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/.config/sshlauncher/.master", homeDir), nil
}
