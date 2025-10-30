# Keyphy - System Access Control via External Device Authentication

[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/gajzzs/keyphy)](https://github.com/gajzzs/keyphy/releases)

Keyphy is a Linux CLI application that blocks access to applications, websites, and files/folders until authenticated with an external USB device. It uses the device's UUID and name to generate cryptographic keys for secure authentication.

```
Keyphy blocks apps, websites, and file access until authenticated with external USB device

Usage:
  keyphy [command]

Available Commands:
  block       Block apps, websites, or file paths
  device      Manage authentication devices
  help        Help about any command
  list        List all blocked items
  lock        Lock all blocks (requires auth device)
  service     Manage keyphy daemon service
  unblock     Remove blocking rule for app, website, or path (use 'all' to remove everything)
  unlock      Unlock all blocks (requires auth device)

Flags:
  -h, --help   help for keyphy

Use "keyphy [command] --help" for more information about a command.
```


## Features

- **Application Blocking**: Block specific applications at the system level using D-Bus monitoring
- **Website Blocking**: Block websites using iptables and hosts file modification  
- **File/Folder Blocking**: Block access to files and directories using permission changes and filesystem attributes
- **USB Device Authentication**: Use external USB devices as authentication keys
- **Cryptographic Security**: Generate secure keys from device UUID and name using PBKDF2
- **System-Level Enforcement**: Cannot be bypassed through normal user operations
- **Daemon Service**: Continuous monitoring and automatic blocking/unblocking


## Installation

### Option 1: Install from Release (Recommended)

```bash
# Download latest release
wget https://github.com/gajzzs/keyphy/releases/latest/download/keyphy-linux-amd64.tar.gz
tar -xzf keyphy-linux-amd64.tar.gz
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

# Install dependencies
make deps

# Build and install
make install

# Install systemd service (optional)
make service
```

Run
```
make service
```
to make persistent across reboots

Service file: /etc/systemd/system/keyphy.service

Auto-starts on boot when enabled: sudo systemctl enable keyphy

## Building from Source

```bash
# Install Go 1.21 or later
# Clone repository
git clone https://github.com/gajzzs/keyphy.git
cd keyphy

# Build
go build -o keyphy ./cmd/keyphy

# Or use make
make build
```

## Contributing

Contributions are welcome! Please read our [Contributing Guidelines](CONTRIBUTING.md) before submitting pull requests.

## Security

For security-related issues, please see our [Security Policy](SECURITY.md).

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Support

- 📖 [Documentation](https://github.com/gajzzs/keyphy/wiki)
- 🐛 [Issue Tracker](https://github.com/gajzzs/keyphy/issues)
- 💬 [Discussions](https://github.com/gajzzs/keyphy/discussions)

## Warning

This tool modifies system-level settings including iptables rules, file permissions, and process management. Use with caution and ensure you have proper backups and recovery methods available.
