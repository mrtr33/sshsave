#!/bin/bash
# SSHLauncher update script

set -e

echo "Building sshsave..."
go build -ldflags "-X github.com/mrtr33/sshsave/cmd.version=$(git describe --tags --always --dirty 2>/dev/null || echo dev)" -o sshsave .

echo "Build successful"
echo ""

# Check if running with sudo
if [ "$EUID" -ne 0 ]; then
    echo "This script needs sudo privileges to install to /usr/local/bin/"
    echo "Run: sudo ./update.sh"
    exit 1
fi

# Backup existing installation if it exists
if [ -f /usr/local/bin/sshsave ]; then
    echo "Backing up current installation..."
    cp /usr/local/bin/sshsave /usr/local/bin/sshsave.bak
    echo "   Backup saved to: /usr/local/bin/sshsave.bak"
fi

# Install new version
echo "Installing to /usr/local/bin/..."
install -m 0755 sshsave /usr/local/bin/sshsave

echo ""
echo "Update complete!"
echo ""
echo "Current version info:"
/usr/local/bin/sshsave --help | head -2
echo ""
echo "Run 'sshsave --help' to see available commands"
