package sshconfig

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// HostEntry represents a parsed SSH config host alias.
// Only the alias is kept: imported shortcuts run "ssh <alias>" and let ssh
// apply every option from the config itself (ProxyJump, forwards, Include...).
type HostEntry struct {
	Host string
}

// splitKeyValue splits a config line into keyword and value. ssh_config
// allows both "Key value" and "Key=value" (with optional spaces around "=").
func splitKeyValue(line string) (string, string, bool) {
	i := strings.IndexAny(line, " \t=")
	if i <= 0 {
		return "", "", false
	}
	key := line[:i]
	rest := strings.TrimLeft(line[i:], " \t")
	if strings.HasPrefix(rest, "=") {
		rest = strings.TrimLeft(rest[1:], " \t")
	}
	if rest == "" {
		return "", "", false
	}
	return strings.ToLower(key), rest, true
}

// ParseSSHConfig reads the SSH config file and returns one entry per host
// alias. A "Host a b" line yields both a and b. Hosts inside Include files
// are not listed, but they still work when connecting.
func ParseSSHConfig(configPath string) ([]HostEntry, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open SSH config: %w", err)
	}
	defer file.Close()

	var entries []HostEntry

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := splitKeyValue(line)
		if !ok || key != "host" {
			continue
		}

		for _, pattern := range strings.Fields(value) {
			entries = append(entries, HostEntry{Host: strings.Trim(pattern, `"`)})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading SSH config: %w", err)
	}

	return entries, nil
}

// GetDefaultSSHConfigPath returns the default SSH config file path
func GetDefaultSSHConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".ssh", "config"), nil
}

// shellQuote quotes s for safe use in a bash command line
func shellQuote(s string) string {
	if s != "" && strings.Trim(s, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789@%+=:,./_-") == "" {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// ToSSHCommand converts a HostEntry to an SSH command string. configPath is
// passed with -F unless it is empty (the default config).
func (h *HostEntry) ToSSHCommand(configPath string) string {
	parts := []string{"ssh"}
	if configPath != "" {
		parts = append(parts, "-F", shellQuote(configPath))
	}
	parts = append(parts, shellQuote(h.Host))
	return strings.Join(parts, " ")
}

// IsValidEntry checks if the host entry is a concrete alias, not a pattern
func (h *HostEntry) IsValidEntry() bool {
	return h.Host != "" && !strings.ContainsAny(h.Host, "*?!")
}
