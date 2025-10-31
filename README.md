# Keyphy - System Access Control via External Device Authentication

[![Go Version](https://img.shields.io/badge/Go-1.23+-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Linux%20%7C%20macOS-blue.svg)]()

Keyphy is a cross-platform CLI application that blocks access to applications, websites, and files/folders until authenticated with an external USB device.

```
Keyphy blocks apps, websites, and file access until authenticated with external USB device

Usage:
  keyphy [command]

Available Commands:
  add         Add apps, websites, or file paths to blocking list
  device      Manage authentication devices
  help        Help about any command
  list        List all blocked items
  lock        Lock all blocks (requires auth device)
  reset       Reset keyphy - remove all blocks, restore system, and stop service
  service     Manage keyphy daemon service
  unblock     Remove blocking rule for app, website, or path
  unlock      Unlock all blocks (requires auth device)

Flags:
  -h, --help      help for keyphy
  -v, --version   version for keyphy

Use "keyphy [command] --help" for more information about a command.
```


## Installation

### Option 1: Install from Release (Recommended)

#### Linux
```bash
# Download latest release
wget https://github.com/gajzzs/keyphy/releases/latest/download/keyphy-linux-amd64.tar.gz
tar -xzf keyphy-linux-amd64.tar.gz
sudo cp keyphy /usr/local/bin/
sudo chmod +x /usr/local/bin/keyphy
```

#### macOS
```bash
# Download latest release
wget https://github.com/gajzzs/keyphy/releases/latest/download/keyphy-darwin-amd64.tar.gz
tar -xzf keyphy-darwin-amd64.tar.gz
sudo cp keyphy /usr/local/bin/
sudo chmod +x /usr/local/bin/keyphy
```

### Option 2: Install with Go

```bash
go install github.com/gajzzs/keyphy/cmd/keyphy@latest
```

### Option 3: Build from Source

```bash
# Clone the repository
git clone https://github.com/gajzzs/keyphy.git
cd keyphy

# Build for current platform
go build -o keyphy ./cmd/keyphy

# Or cross-compile for specific platforms
# Linux
env GOOS=linux GOARCH=amd64 go build -o keyphy-linux ./cmd/keyphy

# macOS
env GOOS=darwin GOARCH=amd64 go build -o keyphy-darwin ./cmd/keyphy

# Install service (cross-platform)
sudo ./keyphy service install
sudo ./keyphy service start
```

### Service Management

```bash
# Install and start service
sudo keyphy service install
sudo keyphy service start
```

## Requirements

- Root/Administrator privileges
- USB device for authentication
- Go 1.23+ (for building from source)

## Building from Source

```bash
git clone https://github.com/gajzzs/keyphy.git
cd keyphy
go build -o keyphy ./cmd/keyphy
```

## License

Apache License 2.0 - see [LICENSE](LICENSE) file for details.

## Emergency Recovery

```bash
# If you lose access to your authentication device
sudo keyphy reset emergency
```

**Warning**: This command removes all blocks and clears the authentication device configuration.

## Warning

This tool modifies system-level settings including firewall rules (iptables/pfctl), file permissions, and process management. Use with caution and ensure you have proper backups and recovery methods available. Always test in a non-production environment first.