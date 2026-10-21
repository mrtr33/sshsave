package storage

// Entry represents a single SSH shortcut entry
type Entry struct {
	Command          string   `json:"command"`
	Alias            []string `json:"alias"`
	EncryptedSecret  string   `json:"encrypted_secret,omitempty"`   // Encrypted passphrase/password
	EncryptedSudoPwd string   `json:"encrypted_sudo_pwd,omitempty"` // Encrypted sudo password
	UseSecret        bool     `json:"use_secret,omitempty"`         // Whether to prompt for secret
}

// Config represents the entire configuration file structure
type Config struct {
	Entries map[string]*Entry `json:"entries"`
}

// NewConfig creates a new empty configuration
func NewConfig() *Config {
	return &Config{
		Entries: make(map[string]*Entry),
	}
}
