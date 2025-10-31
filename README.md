# Keyphy - System Access Control via External Device Authentication

[![Go Version](https://img.shields.io/badge/Go-1.23+-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/gajzzs/keyphy)](https://github.com/gajzzs/keyphy/releases)
[![Platform](https://img.shields.io/badge/Platform-Linux%20%7C%20macOS-blue.svg)]()

Keyphy is a cross-platform CLI application that blocks access to applications, websites, and files/folders until authenticated with an external USB device. It uses the device's UUID and name to generate cryptographic keys for secure authentication.

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
  -h, --help   help for keyphy

Use "keyphy [command] --help" for more information about a command.
```


## Features

### Core Functionality
- **Application Blocking**: Block specific applications at the system level with process monitoring
- **Website Blocking**: Block websites using iptables/pfctl and hosts file modification  
- **File/Folder Blocking**: Block access to files and directories using permission changes and filesystem attributes
- **USB Device Authentication**: Use external USB devices as authentication keys
- **Cross-Platform Support**: Works on Linux and macOS with platform-specific implementations

### Security & Monitoring
- **Cryptographic Security**: Generate secure keys from device UUID and name using PBKDF2
- **Device Fingerprinting**: Prevent USB device cloning attacks with enhanced validation
- **Integrity Monitoring**: Continuous monitoring with automatic re-enforcement of blocks
- **Tamper Detection**: Real-time detection of bypass attempts and system modifications
- **System Monitoring**: Comprehensive monitoring of CPU, memory, disk usage, and suspicious processes
- **Session-Based Authentication**: Time-limited tokens with tamper-evident validation

### System Integration
- **System-Level Enforcement**: Cannot be bypassed through normal user operations
- **Daemon Service**: Continuous monitoring and automatic blocking/unblocking
- **Service Watchdog**: Auto-restart functionality to prevent manual service termination
- **Pure Go Implementation**: No shell command dependencies for enhanced security and performance


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

Keyphy includes cross-platform service management:

```bash
# Install and start service
sudo keyphy service install
sudo keyphy service start

# Enable auto-start on boot
# Linux: systemctl enable keyphy
# macOS: launchctl load /Library/LaunchDaemons/com.keyphy.daemon.plist
```

## Platform Support

| Feature | Linux | macOS | Status |
|---------|-------|-------|---------|
| USB Device Detection | ✅ | ✅ | Production Ready |
| Network Blocking | ✅ (iptables) | ✅ (pfctl) | Production Ready |
| Application Blocking | ✅ | ✅ | Production Ready |
| File/Folder Blocking | ✅ (chattr) | ✅ (chflags) | Production Ready |
| Service Management | ✅ (systemd) | ✅ (launchd) | Production Ready |
| System Monitoring | ✅ | ✅ | Production Ready |

## Requirements

- Go 1.23 or later (for building from source)
- Root/Administrator privileges for system-level operations
- USB device for authentication

### Linux Dependencies
- systemd (for service management)
- iptables (for network blocking)

### macOS Dependencies
- launchd (built-in service manager)
- pfctl (built-in firewall)

## Building from Source

```bash
# Install Go 1.23 or later
# Clone repository
git clone https://github.com/gajzzs/keyphy.git
cd keyphy

# Build for current platform
go build -o keyphy ./cmd/keyphy

# Cross-compile for other platforms
env GOOS=linux GOARCH=amd64 go build -o keyphy-linux ./cmd/keyphy
env GOOS=darwin GOARCH=amd64 go build -o keyphy-darwin ./cmd/keyphy
```

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Changelog

See [CHANGELOG.md](CHANGELOG.md) for detailed version history and updates.

## Support

- 🐛 [Issue Tracker](https://github.com/gajzzs/keyphy/issues)
- 📖 [Documentation](https://github.com/gajzzs/keyphy/wiki)
- 🔧 [Cross-Platform Status](CROSS_PLATFORM_STATUS.md)

## Security Features

- **Device Fingerprinting**: Prevents USB device cloning attacks
- **Cryptographic Authentication**: PBKDF2-based key derivation with SHA256 hashing
- **Integrity Monitoring**: Continuous verification of blocked executables and system files
- **Tamper Detection**: Real-time monitoring for bypass attempts
- **Session Tokens**: Time-limited authentication with automatic expiration
- **Process Protection**: Self-monitoring daemon with watchdog functionality
- **Pure Go Implementation**: No shell command dependencies eliminates injection vulnerabilities

## Emergency Recovery

If you lose access to your authentication device:

```bash
# Emergency reset (bypasses device authentication)
sudo keyphy reset emergency
```

**Warning**: This command removes all blocks and clears the authentication device configuration.

## Warning

This tool modifies system-level settings including firewall rules (iptables/pfctl), file permissions, and process management. Use with caution and ensure you have proper backups and recovery methods available. Always test in a non-production environment first.
