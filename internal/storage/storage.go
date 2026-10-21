package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/mrtr33/sshsave/internal/config"
)

const (
	// The config holds encrypted secrets, so it must only be readable by the owner
	fileMode = 0600
	dirMode  = 0700
)

// Load reads the configuration from disk
func Load() (*Config, error) {
	configPath, err := config.GetConfigPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get config path: %w", err)
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Return empty config if file doesn't exist
		return NewConfig(), nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Initialize entries map if nil
	if cfg.Entries == nil {
		cfg.Entries = make(map[string]*Entry)
	}

	return &cfg, nil
}

// Save writes the configuration to disk with backup.
// Files are written to a temp file and renamed into place, so the config
// is never missing or half-written.
func Save(cfg *Config) error {
	if err := config.EnsureConfigDir(); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configDir, err := config.GetConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}
	// Tighten permissions of directories created by older versions
	if err := os.Chmod(configDir, dirMode); err != nil {
		return fmt.Errorf("failed to set config directory permissions: %w", err)
	}

	configPath, err := config.GetConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	// Create backup if config file exists
	if old, err := os.ReadFile(configPath); err == nil {
		backupPath, err := config.GetBackupPath()
		if err != nil {
			return fmt.Errorf("failed to get backup path: %w", err)
		}

		if err := writeFileAtomic(backupPath, old); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Marshal config to JSON with indentation
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := writeFileAtomic(configPath, data); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// writeFileAtomic writes data to a temp file in the same directory with
// owner-only permissions, then renames it over path
func writeFileAtomic(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath) // no-op after a successful rename

	if err := tmp.Chmod(fileMode); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	return os.Rename(tmpPath, path)
}

// sortedNames returns entry names in a stable order
func (c *Config) sortedNames() []string {
	names := make([]string, 0, len(c.Entries))
	for name := range c.Entries {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// FindEntry looks up an entry by name or alias
func (c *Config) FindEntry(nameOrAlias string) (*Entry, string, bool) {
	// First, try direct name lookup
	if entry, ok := c.Entries[nameOrAlias]; ok {
		return entry, nameOrAlias, true
	}

	// Then search through aliases, in sorted order so the result is deterministic
	for _, name := range c.sortedNames() {
		for _, alias := range c.Entries[name].Alias {
			if alias == nameOrAlias {
				return c.Entries[name], name, true
			}
		}
	}

	return nil, "", false
}

// checkConflicts returns an error if name or any of aliases is already used
// as a name or alias by an entry other than name itself
func (c *Config) checkConflicts(name string, aliases []string) error {
	wanted := append([]string{name}, aliases...)
	seen := make(map[string]bool, len(wanted))
	for _, w := range wanted {
		if seen[w] {
			return fmt.Errorf("'%s' is given more than once", w)
		}
		seen[w] = true
	}

	for _, other := range c.sortedNames() {
		if other == name {
			continue
		}
		for _, w := range wanted {
			if w == other {
				return fmt.Errorf("'%s' is already used as the name of entry '%s'", w, other)
			}
			for _, alias := range c.Entries[other].Alias {
				if w == alias {
					return fmt.Errorf("'%s' is already used as an alias of entry '%s'", w, other)
				}
			}
		}
	}

	return nil
}

// AddEntry adds or updates an entry.
// When updating with force, stored secrets are kept and aliases are only
// replaced if new ones are given.
func (c *Config) AddEntry(name string, command string, aliases []string, force bool) error {
	existing, exists := c.Entries[name]
	if exists && !force {
		return fmt.Errorf("entry '%s' already exists, use --force to overwrite", name)
	}

	if exists && len(aliases) == 0 {
		aliases = existing.Alias
	}
	if aliases == nil {
		aliases = []string{}
	}

	if exists {
		return c.UpdateEntry(name, command, aliases)
	}

	if err := c.checkConflicts(name, aliases); err != nil {
		return err
	}

	c.Entries[name] = &Entry{
		Command: command,
		Alias:   aliases,
	}

	return nil
}

// UpdateEntry sets the command and exact alias list of an existing entry,
// keeping its secrets
func (c *Config) UpdateEntry(name string, command string, aliases []string) error {
	entry, exists := c.Entries[name]
	if !exists {
		return fmt.Errorf("entry '%s' not found", name)
	}
	if aliases == nil {
		aliases = []string{}
	}

	if err := c.checkConflicts(name, aliases); err != nil {
		return err
	}

	entry.Command = command
	entry.Alias = aliases
	return nil
}

// RenameEntry renames the entry found by name or alias and returns its old name
func (c *Config) RenameEntry(nameOrAlias string, newName string) (string, error) {
	entry, oldName, found := c.FindEntry(nameOrAlias)
	if !found {
		return "", fmt.Errorf("entry '%s' not found", nameOrAlias)
	}
	if newName == oldName {
		return oldName, nil
	}
	if _, exists := c.Entries[newName]; exists {
		return "", fmt.Errorf("entry '%s' already exists", newName)
	}

	// Check against all other entries, including the new name clashing with
	// one of this entry's own aliases
	delete(c.Entries, oldName)
	if err := c.checkConflicts(newName, entry.Alias); err != nil {
		c.Entries[oldName] = entry
		return "", err
	}

	c.Entries[newName] = entry
	return oldName, nil
}

// RemoveEntry removes an entry by name or alias and returns the removed name
func (c *Config) RemoveEntry(nameOrAlias string) (string, error) {
	_, name, found := c.FindEntry(nameOrAlias)
	if !found {
		return "", fmt.Errorf("entry '%s' not found", nameOrAlias)
	}

	delete(c.Entries, name)
	return name, nil
}

// Names returns all entry names and aliases, for shell completion
func (c *Config) Names() []string {
	var out []string
	for _, name := range c.sortedNames() {
		out = append(out, name)
		out = append(out, c.Entries[name].Alias...)
	}
	return out
}
