# sshsave

A lightweight CLI tool for managing SSH connection shortcuts with support for aliases, shell auto-completion, encrypted secrets, and importing from SSH config files.

## Features

- Save named SSH shortcuts
- Support for aliases (short names)
- **Encrypt and store SSH passphrases/passwords**
- List all saved shortcuts
- Quick execution of saved shortcuts
- **Import entries from ~/.ssh/config**
- Shell auto-completion (bash, zsh, fish, powershell)
- No passwords or sensitive info stored in plain text
- JSON-based storage with automatic backups

> **Disclaimer:** provided as is, without warranty. Use at your own risk; see [Disclaimer](#disclaimer).

## Installation

### Prebuilt binaries

Download the archive for your platform (Linux or macOS, amd64 or arm64) from the
[releases page](https://github.com/mrtr33/sshsave/releases), then:

```bash
tar xzf sshsave_*_linux_amd64.tar.gz
install -m 0755 sshsave ~/.local/bin/
```

### With `go install`

```bash
go install github.com/mrtr33/sshsave@latest
```

The binary is placed in `$(go env GOPATH)/bin`, which must be on your `PATH`.

### Quick Install (Using Makefile)

```bash
# Build and install system-wide
make install

# Install bash completion
make install-completion
```

### Manual Installation

```bash
# Build
go build -o sshsave .

# Install to system path
sudo install -m 0755 sshsave /usr/local/bin/sshsave

# Install completion manually
sshsave completion bash | sudo tee /etc/bash_completion.d/sshsave > /dev/null
```

### Updating

**Using Makefile (Recommended):**
```bash
# Pull latest changes and update
git pull
make update
```

**Using update script:**
```bash
sudo ./update.sh
```

**Manual update:**
```bash
# Backup current version
sudo cp /usr/local/bin/sshsave /usr/local/bin/sshsave.bak

# Build and install new version
go build -o sshsave .
sudo install -m 0755 sshsave /usr/local/bin/sshsave
```

### Uninstalling

```bash
# Remove system-wide installation
make uninstall

# Or manually
sudo rm /usr/local/bin/sshsave
sudo rm /etc/bash_completion.d/sshsave

# Optionally remove config (careful!)
rm -rf ~/.config/sshlauncher
```

### Shell Completion Installation

**Bash:**
```bash
sshsave completion bash | sudo tee /etc/bash_completion.d/sshsave > /dev/null
# or
make install-completion
```

**Zsh:**
```bash
sshsave completion zsh > ~/.zsh/completions/_sshsave
```

**Fish:**
```bash
sshsave completion fish > ~/.config/fish/completions/sshsave.fish
```

## Quick Start

### Basic Usage

```bash
# Add an SSH shortcut
sshsave add work "ssh user@192.168.10.10"

# Add with aliases
sshsave add work "ssh user@192.168.10.10" --alias w --alias wk

# List all shortcuts
sshsave list

# Run a shortcut
sshsave run work
# or use an alias, or leave out "run"
sshsave w

# Remove a shortcut
sshsave rm work
```

### Advanced Features

#### Importing from SSH Config

Import all your existing SSH config entries:

```bash
# Import from default location (~/.ssh/config)
sshsave import

# Import from custom location
sshsave import --config /path/to/config

# Import with a prefix to avoid conflicts
sshsave import --prefix imported-

# Overwrite existing entries
sshsave import --force
```

**Note:** Wildcard hosts (e.g., `Host *.example.com`) are automatically skipped during import.

#### Encrypted Secrets

Store encrypted SSH passphrases or passwords for hosts requiring authentication:

```bash
# Add entry with encrypted secret (will prompt for passphrase)
sshsave add secure "ssh user@secure-host.com" --with-secret

# Add encrypted sudo password
sshsave add admin "ssh root@server.com" --with-sudo-password
```

**How it works:**
1. When you add an entry with `--with-secret`, you'll be prompted for:
   - Your master password (used to encrypt/decrypt all secrets)
   - The SSH passphrase/password to encrypt
2. The secret is encrypted using AES-256-GCM and stored in the config file
3. When running the entry with `sshsave run`, you'll be prompted for the master password
4. The decrypted secret is handed to ssh through `SSH_ASKPASS` (sshsave acts as its own askpass helper), so it is never shown on screen. Host key confirmations are still asked from you on the terminal. Requires OpenSSH 8.4 or newer.

**Security notes:**
- Secrets are encrypted using AES-256-GCM with Argon2id key derivation
- Master password is never stored, only used for encryption/decryption
- Each secret uses a unique salt and nonce
- All secrets share one master password; adding a secret with a different one is refused
- Secrets written by versions before 0.2 (PBKDF2) are still read, and are re-encrypted with Argon2id the next time you run an entry with the master password
- Encrypted data is stored in `~/.config/sshlauncher/sshlauncher.json` (mode 0600, directory 0700)
- Secrets can only be entered at a prompt, never as command-line arguments

## Commands

### `sshsave add <name> <ssh-command>`

Add or update an SSH shortcut entry.

**Flags:**
- `--alias <name>` - Add alias(es) for this entry (can be used multiple times)
- `--force` - Overwrite existing entry (keeps stored secrets, and aliases unless new ones are given)
- `--with-secret` - Prompt for and encrypt SSH passphrase
- `--with-sudo-password` - Prompt for a sudo password to encrypt and store

**Examples:**
```bash
sshsave add work "ssh user@192.168.10.10"
sshsave add home "ssh me@10.0.0.2 -p 2222" --alias h
sshsave add secure "ssh user@host.com" --with-secret
```

### `sshsave run <name>`

Execute a saved SSH shortcut by name or alias.

**Examples:**
```bash
sshsave run work
sshsave run w  # using an alias
sshsave work   # "run" can be left out
```

If the entry has an encrypted secret, you'll be prompted for your master password.

Commands take precedence over entry names, so an entry called e.g. `list` has to be run with `sshsave run list`.

### `sshsave list`

Display all saved SSH shortcuts with their commands, aliases, and encryption status.

Entries with encrypted secrets are marked with `[encrypted]`.

### `sshsave rm <name>`

Remove a saved SSH shortcut entry (by name or alias).

**Example:**
```bash
sshsave rm work
```

### `sshsave rename <name> <new-name>`

Rename an entry. Its command, aliases and secrets are kept.

```bash
sshsave rename work office
```

### `sshsave edit <name>`

Change parts of an entry without re-creating it.

**Flags:**
- `--command <cmd>` - New SSH command
- `--add-alias <name>` / `--remove-alias <name>` - Change aliases (can be repeated)
- `--with-secret` / `--clear-secret` - Set (prompted) or remove the SSH secret
- `--with-sudo-password` / `--clear-sudo-password` - Set (prompted) or remove the sudo password

```bash
sshsave edit work --command "ssh me@10.0.0.5 -p 2222" --add-alias w
sshsave edit work --with-secret
```

### `sshsave passwd`

Change the master password. Every stored secret is decrypted with the current
master password and re-encrypted with the new one; if any secret does not
decrypt, nothing is changed.

### `sshsave --version`

Print the version.

### `sshsave import`

Import host entries from SSH config file.

**Flags:**
- `--config <path>` - Path to SSH config file (default: ~/.ssh/config)
- `--force` - Overwrite existing entries
- `--prefix <prefix>` - Prefix to add to imported entry names

**Examples:**
```bash
sshsave import
sshsave import --config ~/.ssh/config
sshsave import --prefix imported- --force
```

Each concrete `Host` alias becomes a shortcut that runs `ssh <alias>` (with `-F <path>` for a non-default config), so ssh itself applies every option from the config, including `ProxyJump`, port forwards, `Match` and `Include`. A line like `Host a b` creates both `a` and `b`; wildcard patterns are skipped.

### `sshsave completion <shell>`

Generate shell completion script.

**Supported shells:** bash, zsh, fish, powershell

**Example:**
```bash
sshsave completion bash > /tmp/sshsave-completion.bash
```

## Configuration

### Storage Location

Configuration is stored in `~/.config/sshlauncher/sshlauncher.json`

### Config File Format

```json
{
  "entries": {
    "work": {
      "command": "ssh user@192.168.10.10",
      "alias": ["w", "wk"]
    },
    "secure": {
      "command": "ssh user@secure-host.com",
      "alias": [],
      "encrypted_secret": "base64-encoded-encrypted-data",
      "use_secret": true
    },
    "home": {
      "command": "ssh me@10.0.0.2 -p 2222",
      "alias": ["h"]
    }
  }
}
```

### Backups

Before each save operation, the current config is backed up to:
`~/.config/sshlauncher/sshlauncher.json.bak`

Both files are written to a temporary file first and renamed into place, so an interrupted save never leaves the config missing or half-written.

## Security Considerations

- No passwords or SSH keys are stored in plain text
- Encrypted secrets use AES-256-GCM with Argon2id key derivation
- Master password is never stored on disk
- Each encrypted value has a unique salt and nonce
- Only literal commands provided by user are stored
- No telemetry or network communication
- Follows XDG base directory specification

## Development

### Building

```bash
go build -o sshsave .
```

### Running tests

```bash
go test ./...
```

### Releasing

Pushing a `v*` tag builds binaries with [GoReleaser](https://goreleaser.com) and
publishes them as a GitHub release:

```bash
git tag v0.2.0
git push origin v0.2.0
```

### Project Structure

```
sshsave/
├── cmd/                    # Cobra commands
│   ├── root.go            # Root command setup
│   ├── add.go             # Add/update entries
│   ├── list.go            # List entries
│   ├── run.go             # Execute entries
│   ├── rm.go              # Remove entries
│   ├── rename.go          # Rename entries
│   ├── edit.go            # Edit entries
│   ├── passwd.go          # Change master password
│   ├── secrets.go         # Shared secret handling and upgrades
│   ├── import.go          # Import from SSH config
│   └── completion.go      # Shell completion
├── internal/
│   ├── config/            # Config path management
│   ├── storage/           # JSON storage and data models
│   ├── executor/          # SSH command execution
│   ├── askpass/           # Hands secrets to ssh via SSH_ASKPASS
│   ├── crypto/            # Encryption/decryption
│   └── sshconfig/         # SSH config parser
├── main.go
└── README.md
```

## Requirements

- Go 1.25+ (to build)
- OpenSSH 8.4+ (for stored secrets, which use `SSH_ASKPASS_REQUIRE`)
- Linux, macOS, or Windows with WSL

## Disclaimer

sshsave is a hobby project provided "as is", without warranty of any kind. The
author is not responsible for any damage, data loss, lost access to servers,
leaked credentials or other consequences of using it. Use it at your own risk.

- The encryption has not been independently audited. Do not rely on it as your
  only protection for important credentials.
- If you forget the master password, stored secrets cannot be recovered.
- Keep your own backups of `~/.config/sshlauncher/` and of the credentials it holds.
- Where possible, prefer SSH keys with `ssh-agent` over stored passwords.

See the [LICENSE](LICENSE) for the full terms.

## License

MIT, see [LICENSE](LICENSE).

## Contributing

Contributions are welcome! Please feel free to open an issue or a pull request.

## Future Enhancements

- [ ] TUI mode for interactive management
- [ ] Remote sync using Git
- [ ] Export to SSH config format
- [ ] Support for SSH connection profiles
